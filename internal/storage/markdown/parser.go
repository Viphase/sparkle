// Package markdown reads and writes Sparkle's Markdown files (frontmatter +
// body) and exposes a per-entity Store on top.
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is a parsed Markdown file: its YAML frontmatter (as a map so
// unknown keys round-trip) and the body that follows.
type Document struct {
	Frontmatter map[string]any
	Body        string
}

// ErrUnterminatedFrontmatter is returned when a file opens with `---` but
// never closes it.
var ErrUnterminatedFrontmatter = errors.New("unterminated frontmatter")

// Parse splits a Markdown file into frontmatter and body. Files without a
// leading `---` line are treated as body-only.
func Parse(raw []byte) (Document, error) {
	s := string(raw)

	firstNL := strings.IndexByte(s, '\n')
	if firstNL < 0 || strings.TrimRight(s[:firstNL], "\r") != "---" {
		return Document{Frontmatter: map[string]any{}, Body: s}, nil
	}
	rest := s[firstNL+1:]

	closeIdx := -1
	cursor := 0
	for cursor <= len(rest) {
		nl := strings.IndexByte(rest[cursor:], '\n')
		var line string
		if nl < 0 {
			line = rest[cursor:]
		} else {
			line = rest[cursor : cursor+nl]
		}
		if strings.TrimRight(line, "\r") == "---" {
			closeIdx = cursor
			break
		}
		if nl < 0 {
			break
		}
		cursor += nl + 1
	}
	if closeIdx < 0 {
		return Document{}, ErrUnterminatedFrontmatter
	}

	fmYAML := rest[:closeIdx]
	afterDelim := rest[closeIdx:]
	closeNL := strings.IndexByte(afterDelim, '\n')
	body := ""
	if closeNL >= 0 {
		body = afterDelim[closeNL+1:]
	}

	fm := map[string]any{}
	if strings.TrimSpace(fmYAML) != "" {
		if err := yaml.Unmarshal([]byte(fmYAML), &fm); err != nil {
			return Document{}, fmt.Errorf("yaml: %w", err)
		}
	}
	if fm == nil {
		fm = map[string]any{}
	}
	return Document{Frontmatter: fm, Body: body}, nil
}

// Encode renders a Document back to Markdown bytes. Maps are emitted in YAML's
// stable order; ordering is not preserved across a Parse → Encode cycle.
func Encode(doc Document) ([]byte, error) {
	var buf bytes.Buffer
	if len(doc.Frontmatter) > 0 {
		buf.WriteString("---\n")
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(doc.Frontmatter); err != nil {
			return nil, fmt.Errorf("encode yaml: %w", err)
		}
		if err := enc.Close(); err != nil {
			return nil, fmt.Errorf("close yaml: %w", err)
		}
		buf.WriteString("---\n")
	}
	buf.WriteString(doc.Body)
	return buf.Bytes(), nil
}
