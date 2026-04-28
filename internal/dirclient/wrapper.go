package dirclient

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"google.golang.org/protobuf/encoding/protojson"
)

// Config holds the connection configuration for a directory server.
type Config struct {
	ServerAddress string
	AuthMode      string
	TLSSkipVerify bool
	TLSCAFile     string
	TLSCertFile   string
	TLSKeyFile    string
	AuthToken     string
}

// RecordSummary is a lightweight representation of a directory record.
type RecordSummary struct {
	CID           string
	Name          string
	Version       string
	SchemaVersion string
	Authors       []string
	Skills        []string
	Domains       []string
	Modules       []string
	// Signed indicates the record carries a signature in the OASF payload.
	// Used as a best-effort proxy for "verified" without a server round-trip.
	Signed bool
}

// Client wraps the agntcy/dir gRPC client.
type Client struct {
	c      *client.Client
	Config Config
}

// Connect creates a new connected client.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	dirCfg := &client.Config{
		ServerAddress: cfg.ServerAddress,
		AuthMode:      cfg.AuthMode,
		TlsSkipVerify: cfg.TLSSkipVerify,
		TlsCAFile:     cfg.TLSCAFile,
		TlsCertFile:   cfg.TLSCertFile,
		TlsKeyFile:    cfg.TLSKeyFile,
		AuthToken:     cfg.AuthToken,
	}

	c, err := client.New(ctx, client.WithConfig(dirCfg))
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", cfg.ServerAddress, err)
	}

	return &Client{c: c, Config: cfg}, nil
}

// Close closes the underlying connection.
func (c *Client) Close() {
	if c.c != nil {
		_ = c.c.Close()
	}
}

// ListAll fetches all records from the directory as summaries.
func (c *Client) ListAll(ctx context.Context) ([]*RecordSummary, error) {
	limit := uint32(1000) //nolint:mnd
	offset := uint32(0)

	result, err := c.c.SearchRecords(ctx, &searchv1.SearchRecordsRequest{
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil {
		return nil, fmt.Errorf("searching records: %w", err)
	}

	var summaries []*RecordSummary

	for {
		select {
		case resp := <-result.ResCh():
			record := resp.GetRecord()
			if record == nil {
				continue
			}
			s := extractSummary(record)
			if s != nil {
				summaries = append(summaries, s)
			}
		case err := <-result.ErrCh():
			return nil, fmt.Errorf("receiving record: %w", err)
		case <-result.DoneCh():
			return summaries, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// PullJSON fetches a single record by CID and returns it as formatted JSON.
func (c *Client) PullJSON(ctx context.Context, cid string) (string, error) {
	record, err := c.c.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return "", fmt.Errorf("pulling record %s: %w", cid, err)
	}

	data := record.GetData()
	if data == nil {
		return "{}", nil
	}

	// Marshal using protojson for proper field names.
	b, err := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: false,
	}.Marshal(data)
	if err != nil {
		// Fallback to standard JSON if protojson fails.
		b2, err2 := json.MarshalIndent(data, "", "  ")
		if err2 != nil {
			return "", fmt.Errorf("marshaling record to JSON: %w", err)
		}
		return string(b2), nil
	}

	return string(b), nil
}

// FilterValues holds the unique values present in a record set for each
// filterable field. Used to populate the [2] Filters panel options lists.
type FilterValues struct {
	Skills         []string
	Domains        []string
	Modules        []string
	OASFVersions   []string
	Versions       []string
	Authors        []string
}

// ExtractClasses returns unique skill, domain, and module names from a list
// of summaries. Kept for backward compatibility with callers that only need
// the taxonomy values.
func ExtractClasses(summaries []*RecordSummary) (skills, domains, modules []string) {
	v := ExtractFilterValues(summaries)
	return v.Skills, v.Domains, v.Modules
}

// ExtractFilterValues collects the unique values for every supported filter
// category across all records.
func ExtractFilterValues(summaries []*RecordSummary) FilterValues {
	skills := map[string]bool{}
	domains := map[string]bool{}
	modules := map[string]bool{}
	oasfVersions := map[string]bool{}
	versions := map[string]bool{}
	authors := map[string]bool{}

	for _, s := range summaries {
		for _, v := range s.Skills {
			skills[v] = true
		}
		for _, v := range s.Domains {
			domains[v] = true
		}
		for _, v := range s.Modules {
			modules[v] = true
		}
		for _, v := range s.Authors {
			if v != "" {
				authors[v] = true
			}
		}
		if s.SchemaVersion != "" {
			oasfVersions[s.SchemaVersion] = true
		}
		if s.Version != "" {
			versions[s.Version] = true
		}
	}

	return FilterValues{
		Skills:       sortedKeys(skills),
		Domains:      sortedKeys(domains),
		Modules:      sortedKeys(modules),
		OASFVersions: sortedKeys(oasfVersions),
		Versions:     sortedKeys(versions),
		Authors:      sortedKeys(authors),
	}
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

// extractSummary pulls name/version/skills/domains/modules from a raw record.
func extractSummary(record *corev1.Record) *RecordSummary {
	cid := record.GetCid()
	data := record.GetData()
	if data == nil {
		return nil
	}

	decoded, err := decoder.DecodeRecord(data)
	if err != nil || decoded == nil {
		return nil
	}

	s := &RecordSummary{CID: cid}

	switch {
	case decoded.HasV1():
		r := decoded.GetV1()
		if r == nil {
			return nil
		}
		s.Name = r.GetName()
		s.Version = r.GetVersion()
		s.SchemaVersion = r.GetSchemaVersion()
		s.Authors = append(s.Authors, r.GetAuthors()...)
		for _, sk := range r.GetSkills() {
			if sk.GetName() != "" {
				s.Skills = append(s.Skills, sk.GetName())
			}
		}
		for _, d := range r.GetDomains() {
			if d.GetName() != "" {
				s.Domains = append(s.Domains, d.GetName())
			}
		}
		for _, m := range r.GetModules() {
			if m.GetName() != "" {
				s.Modules = append(s.Modules, m.GetName())
			}
		}
	case decoded.HasV1Alpha2():
		r := decoded.GetV1Alpha2()
		if r == nil {
			return nil
		}
		s.Name = r.GetName()
		s.Version = r.GetVersion()
		s.SchemaVersion = r.GetSchemaVersion()
		s.Authors = append(s.Authors, r.GetAuthors()...)
	case decoded.HasV1Alpha1():
		r := decoded.GetV1Alpha1()
		if r == nil {
			return nil
		}
		s.Name = r.GetName()
		s.Version = r.GetVersion()
		s.SchemaVersion = r.GetSchemaVersion()
		s.Authors = append(s.Authors, r.GetAuthors()...)
		if sig := r.GetSignature(); sig != nil && sig.GetSignature() != "" {
			s.Signed = true
		}
	default:
		return nil
	}

	if s.Name == "" && cid != "" {
		s.Name = cid[:min(20, len(cid))]
	}

	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
