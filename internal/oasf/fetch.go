package oasf

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://schema.oasf.outshift.com/1.0.0"

// ClassType identifies the OASF taxonomy class type.
type ClassType string

const (
	ClassTypeSkill  ClassType = "skills"
	ClassTypeDomain ClassType = "domains"
	ClassTypeModule ClassType = "modules"
)

// ClassInfo holds basic info about a taxonomy class.
type ClassInfo struct {
	Name        string
	Caption     string
	Description string
	Type        ClassType
}

var (
	mu    sync.Mutex
	cache = map[string]*ClassInfo{}
)

// Fetch retrieves the description for a class from the OASF API.
// Results are cached in memory.
func Fetch(classType ClassType, name string) (*ClassInfo, error) {
	key := string(classType) + "/" + name
	mu.Lock()
	if v, ok := cache[key]; ok {
		mu.Unlock()
		return v, nil
	}
	mu.Unlock()

	url := fmt.Sprintf("%s/%s/%s", baseURL, classType, name)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("OASF request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("class %q not found in OASF schema", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OASF returned status %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading OASF response: %w", err)
	}

	info := &ClassInfo{
		Name:        name,
		Type:        classType,
		Description: extractDescription(string(body)),
	}

	mu.Lock()
	cache[key] = info
	mu.Unlock()

	return info, nil
}

// extractDescription pulls the human-readable description from the OASF HTML page.
// The page structure contains descriptive paragraphs after the header.
func extractDescription(html string) string {
	var lines []string
	// Strip HTML tags and extract visible text lines.
	inTag := false
	var cur strings.Builder
	for _, ch := range html {
		switch {
		case ch == '<':
			inTag = true
			text := strings.TrimSpace(cur.String())
			if text != "" {
				lines = append(lines, text)
			}
			cur.Reset()
		case ch == '>':
			inTag = false
		case !inTag:
			cur.WriteRune(ch)
		}
	}
	if text := strings.TrimSpace(cur.String()); text != "" {
		lines = append(lines, text)
	}

	// Filter out boilerplate lines and collect meaningful description lines.
	var result []string
	skip := map[string]bool{
		"Open Agentic Schema Framework": true,
		"Card view Taxonomy view":       true,
		"JSON Schema Sample Validate":   true,
		"Base Attributes Optional Attributes": true,
		"Decline Accept":                true,
	}
	for _, line := range lines {
		if skip[line] {
			continue
		}
		// Skip lines that are just IDs in brackets like "[101]".
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}
		// Skip navigation artifacts.
		if strings.Contains(line, "OASF Server version") ||
			strings.Contains(line, "Terms &") ||
			strings.Contains(line, "Privacy Policy") ||
			strings.Contains(line, "We use cookies") {
			continue
		}
		if line != "" {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		return "No description available."
	}
	return strings.Join(result, "\n")
}
