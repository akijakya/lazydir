package dirclient

import (
	"context"
	"encoding/json"
	"fmt"

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

// RecordSummary is a lightweight representation of a directory record. It
// exposes only the fields the TUI renders or filters on; everything else from
// the wire record is discarded by extractSummary.
type RecordSummary struct {
	CID           string
	Name          string
	Version       string
	SchemaVersion string
	Authors       []string
	Skills        []string
	Domains       []string
	Modules       []string
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

// FilterCategory identifies a server-side filter predicate. Each value maps
// 1:1 to a RecordQueryType in the agntcy.dir.search.v1 protobuf API.
type FilterCategory int

const (
	FilterSkill FilterCategory = iota
	FilterDomain
	FilterModule
	FilterSchemaVersion
	FilterVersion
	FilterAuthor
	FilterTrusted
	FilterVerified
)

// Query is one server-side predicate. Multiple Query values combine on the
// server with the semantics defined by the directory implementation.
type Query struct {
	Category FilterCategory
	Value    string
}

func (q Query) toRPC() *searchv1.RecordQuery {
	var t searchv1.RecordQueryType
	switch q.Category {
	case FilterSkill:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME
	case FilterDomain:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME
	case FilterModule:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME
	case FilterSchemaVersion:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION
	case FilterVersion:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION
	case FilterAuthor:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR
	case FilterTrusted:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_TRUSTED
	case FilterVerified:
		t = searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERIFIED
	}
	return &searchv1.RecordQuery{Type: t, Value: q.Value}
}

// FirstPageSize controls how many records are pulled in the initial batch
// before the loader signals "first page ready" to the caller. After that,
// pagination continues transparently in the background.
const FirstPageSize = 100

// streamBatchSize is the size of the batches delivered after the first page.
// It exists so the GUI can re-render once per batch instead of once per
// record, which is the difference between O(N) and O(N²) on large
// directories. Tuned to be small enough to feel "live" but big enough to
// keep redraws under control.
const streamBatchSize = 50

// rpcPageSize is how many records we ask the server for in a single
// SearchRecords RPC. The server enforces its own cap (currently 1000 in
// dirctl); we pick a value at that ceiling so we minimise round trips on
// large directories. If the server returns fewer than rpcPageSize records,
// we treat it as the last page and stop paginating.
const rpcPageSize uint32 = 1000

// StreamCallbacks bundle the optional notification hooks for Stream. Any of
// the callbacks may be nil. They are invoked from the goroutine driving the
// stream — callers must not block inside them.
type StreamCallbacks struct {
	// OnFirstPage fires once after the first FirstPageSize records have been
	// received (or after the stream ends, whichever comes first).
	OnFirstPage func(summaries []*RecordSummary)
	// OnBatch fires for every subsequent batch of streamBatchSize records
	// (and once at the end with whatever's left over). Batching exists so
	// callers can amortize per-update work like UI redraws.
	OnBatch func(summaries []*RecordSummary)
	// OnDone fires exactly once when the stream finishes — either cleanly,
	// because of an error, or because ctx was cancelled. err is nil on a
	// clean finish.
	OnDone func(err error)
}

// Stream issues SearchRecords RPCs with the supplied queries and drives the
// returned streams until the server has nothing more to send or ctx is
// cancelled. The first FirstPageSize records are delivered via OnFirstPage
// (so the UI can paint quickly); remaining records arrive in OnBatch chunks.
// Pagination across the server's per-RPC cap is handled internally — the
// caller sees a single logical stream.
//
// Callbacks fire on this goroutine; cancel ctx to stop reading at any time.
//
// Stream is single-shot: callers should re-invoke it (typically with a new
// ctx) whenever the active filter set changes. A previous in-flight call
// should be cancelled by its own ctx before starting a new one.
func (c *Client) Stream(ctx context.Context, queries []Query, cb StreamCallbacks) {
	rpcQueries := make([]*searchv1.RecordQuery, 0, len(queries))
	for _, q := range queries {
		rpcQueries = append(rpcQueries, q.toRPC())
	}

	buf := make([]*RecordSummary, 0, FirstPageSize)
	firstPageSent := false

	// handOff returns buf and starts a fresh backing array. Callbacks run
	// on this goroutine but the caller may stash the slice for later
	// rendering on a different goroutine, so we must not reuse the storage.
	handOff := func(capHint int) []*RecordSummary {
		out := buf
		buf = make([]*RecordSummary, 0, capHint)
		return out
	}

	flushFirstPage := func() {
		if firstPageSent {
			return
		}
		firstPageSent = true
		batch := handOff(streamBatchSize)
		if cb.OnFirstPage != nil {
			cb.OnFirstPage(batch)
		}
	}
	flushBatch := func() {
		if !firstPageSent {
			flushFirstPage()
			return
		}
		if len(buf) == 0 {
			return
		}
		batch := handOff(streamBatchSize)
		if cb.OnBatch != nil {
			cb.OnBatch(batch)
		}
	}
	finish := func(err error) {
		flushBatch()
		if cb.OnDone != nil {
			cb.OnDone(err)
		}
	}

	limit := rpcPageSize
	var offset uint32

	for {
		req := &searchv1.SearchRecordsRequest{
			Queries: rpcQueries,
			Limit:   &limit,
			Offset:  &offset,
		}
		result, err := c.c.SearchRecords(ctx, req)
		if err != nil {
			finish(fmt.Errorf("searching records (offset=%d): %w", offset, err))
			return
		}

		pageCount := uint32(0)
	pageLoop:
		for {
			select {
			case resp, ok := <-result.ResCh():
				if !ok {
					break pageLoop
				}
				record := resp.GetRecord()
				if record == nil {
					continue
				}
				s := extractSummary(record)
				if s == nil {
					continue
				}
				pageCount++
				buf = append(buf, s)
				if !firstPageSent {
					if len(buf) >= FirstPageSize {
						flushFirstPage()
					}
					continue
				}
				if len(buf) >= streamBatchSize {
					flushBatch()
				}
			case streamErr := <-result.ErrCh():
				if streamErr != nil {
					finish(fmt.Errorf("receiving record (offset=%d): %w", offset, streamErr))
					return
				}
				break pageLoop
			case <-result.DoneCh():
				break pageLoop
			case <-ctx.Done():
				finish(ctx.Err())
				return
			}
		}

		// Short page → server has nothing more for us; stop paginating.
		if pageCount < limit {
			finish(nil)
			return
		}
		offset += pageCount
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
