package oasf

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	sdkschema "github.com/agntcy/oasf-sdk/pkg/schema"
)

// DefaultServerAddress is used when the user does not override it.
const DefaultServerAddress = "https://schema.oasf.outshift.com"

// ClassType identifies the OASF taxonomy class type.
type ClassType string

const (
	ClassTypeSkill  ClassType = "skills"
	ClassTypeDomain ClassType = "domains"
	ClassTypeModule ClassType = "modules"
)

// ClassInfo holds user-visible info about a taxonomy class.
type ClassInfo struct {
	Name        string
	Caption     string
	Description string
	Type        ClassType
}

// Config configures the OASF schema client.
type Config struct {
	// ServerAddress is the base URL of the OASF schema server, e.g.
	// "https://schema.oasf.outshift.com". Protocol is optional.
	ServerAddress string
}

// Client wraps the OASF SDK schema client and caches class info lookups.
type Client struct {
	cfg    Config
	sdk    *sdkschema.Schema
	cacheM sync.Mutex
	cache  map[string]*ClassInfo
}

// NewClient constructs a new OASF client backed by the oasf-sdk/pkg/schema.
// The SDK performs URL normalization, so http/https schemes are handled.
func NewClient(cfg Config) (*Client, error) {
	if cfg.ServerAddress == "" {
		return nil, errors.New("OASF server address is required")
	}

	sdk, err := sdkschema.New(cfg.ServerAddress, sdkschema.WithCache(true))
	if err != nil {
		return nil, fmt.Errorf("creating OASF schema client: %w", err)
	}

	return &Client{
		cfg:   cfg,
		sdk:   sdk,
		cache: map[string]*ClassInfo{},
	}, nil
}

// ServerAddress returns the URL the client is currently pointed at.
func (c *Client) ServerAddress() string {
	return c.cfg.ServerAddress
}

// Fetch retrieves the description for a taxonomy class by its fully-qualified
// name (e.g. "natural_language_processing/text_completion"). The lookup walks
// the nested category tree returned by the SDK until it finds a matching node.
func (c *Client) Fetch(ctx context.Context, classType ClassType, name string) (*ClassInfo, error) {
	key := string(classType) + "|" + name
	c.cacheM.Lock()
	if v, ok := c.cache[key]; ok {
		c.cacheM.Unlock()
		return v, nil
	}
	c.cacheM.Unlock()

	// Budget the lookup so the UI does not hang on slow networks.
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	taxonomy, err := c.getTaxonomy(reqCtx, classType)
	if err != nil {
		return nil, err
	}

	item, ok := findItem(taxonomy, name)
	if !ok {
		return nil, fmt.Errorf("class %q not found in OASF %s taxonomy", name, classType)
	}

	info := &ClassInfo{
		Name:        item.Name,
		Caption:     item.Caption,
		Description: item.Description,
		Type:        classType,
	}
	if info.Description == "" {
		info.Description = "No description available."
	}

	c.cacheM.Lock()
	c.cache[key] = info
	c.cacheM.Unlock()

	return info, nil
}

func (c *Client) getTaxonomy(ctx context.Context, classType ClassType) (sdkschema.Taxonomy, error) {
	switch classType {
	case ClassTypeSkill:
		return c.sdk.GetSchemaSkills(ctx)
	case ClassTypeDomain:
		return c.sdk.GetSchemaDomains(ctx)
	case ClassTypeModule:
		return c.sdk.GetSchemaModules(ctx)
	default:
		return nil, fmt.Errorf("unknown OASF class type %q", classType)
	}
}

// findItem walks the nested taxonomy tree and returns the TaxonomyItem whose
// Name matches the supplied identifier. The SDK Taxonomy is a map of top-level
// categories, each containing Classes which may themselves be categories.
func findItem(root sdkschema.Taxonomy, name string) (sdkschema.TaxonomyItem, bool) {
	return findInItems(map[string]sdkschema.TaxonomyItem(root), name)
}

func findInItems(items map[string]sdkschema.TaxonomyItem, name string) (sdkschema.TaxonomyItem, bool) {
	for _, item := range items {
		if item.Name == name {
			return item, true
		}
		if len(item.Classes) > 0 {
			if child, ok := findInItems(item.Classes, name); ok {
				return child, true
			}
		}
	}
	return sdkschema.TaxonomyItem{}, false
}
