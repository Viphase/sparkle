// Package markdown reads and writes Sparkle's Markdown files (frontmatter +
// body) and exposes a per-entity Store on top.
package markdown

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
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

// BodySection extracts the raw source text under a top-level H1 heading from a
// Markdown body string. It uses goldmark's AST parser so code fences, nested
// headings, list nesting, and blank lines are preserved correctly.
//
// Matching is case-insensitive on the heading text. The content runs from the
// line after the matched heading to the line before the next H1 (or EOF).
// Returns an empty string when no matching heading is found.
//
// L2: replaces the old line-by-line string-scan approach with a real AST.
func BodySection(body, heading string) string {
	src := []byte(body)
	parser := goldmark.DefaultParser()
	reader := text.NewReader(src)
	doc := parser.Parse(reader)

	want := strings.ToLower(strings.TrimSpace(heading))

	// Walk only the document's direct children (top-level blocks).
	capturing := false
	start := -1
	end := -1

	for node := doc.FirstChild(); node != nil; node = node.NextSibling() {
		if node.Kind() == ast.KindHeading {
			h := node.(*ast.Heading)
			if h.Level == 1 {
				// Extract heading text from its children.
				var hb strings.Builder
				for c := node.FirstChild(); c != nil; c = c.NextSibling() {
					if tc, ok := c.(*ast.Text); ok {
						hb.Write(tc.Segment.Value(src))
					}
				}
				title := strings.ToLower(strings.TrimSpace(hb.String()))
				if title == want {
					capturing = true
					// Start from the byte after the heading's last byte.
					if node.Lines().Len() > 0 {
						seg := node.Lines().At(node.Lines().Len() - 1)
						start = seg.Stop
					} else {
						start = 0
					}
					continue
				}
				if capturing {
					// Next H1 found — record where section content ends.
					if node.Lines().Len() > 0 {
						seg := node.Lines().At(0)
						end = seg.Start
					}
					break
				}
			}
			continue
		}
		if capturing && node.Lines().Len() > 0 {
			// Track the latest byte seen so we know section's end.
			seg := node.Lines().At(node.Lines().Len() - 1)
			if end < seg.Stop {
				end = seg.Stop
			}
		}
	}

	if start < 0 {
		return ""
	}
	if end < 0 || end > len(src) {
		end = len(src)
	}
	return strings.TrimSpace(string(src[start:end]))
}
