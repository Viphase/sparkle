package markdown

import (
	"errors"
	"strings"
	"testing"
)

func TestParseNoFrontmatter(t *testing.T) {
	doc, err := Parse([]byte("just a body\nwith two lines"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Body != "just a body\nwith two lines" {
		t.Errorf("body: %q", doc.Body)
	}
	if len(doc.Frontmatter) != 0 {
		t.Errorf("expected empty frontmatter, got %v", doc.Frontmatter)
	}
}

func TestParseEmptyFrontmatter(t *testing.T) {
	doc, err := Parse([]byte("---\n---\nbody"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Body != "body" {
		t.Errorf("body: %q", doc.Body)
	}
}

func TestParseUnterminatedFrontmatter(t *testing.T) {
	_, err := Parse([]byte("---\ntitle: oops\nno close here"))
	if !errors.Is(err, ErrUnterminatedFrontmatter) {
		t.Errorf("got %v, want ErrUnterminatedFrontmatter", err)
	}
}

func TestParseFrontmatterAndBody(t *testing.T) {
	in := "---\ntitle: Hello\ncount: 3\n---\nBody line one.\nBody line two.\n"
	doc, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Frontmatter["title"] != "Hello" {
		t.Errorf("title: %v", doc.Frontmatter["title"])
	}
	if doc.Frontmatter["count"] != 3 {
		t.Errorf("count: %v (%T)", doc.Frontmatter["count"], doc.Frontmatter["count"])
	}
	if doc.Body != "Body line one.\nBody line two.\n" {
		t.Errorf("body: %q", doc.Body)
	}
}

func TestRoundtripPreservesUnknownFields(t *testing.T) {
	in := `---
schema_version: 1
id: spark_x
title: Hello
custom_field: keep-me
nested:
  unknown: yes
---
Body content here.
`
	doc, err := Parse([]byte(in))
	if err != nil {
		t.Fatal(err)
	}
	out, err := Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if doc2.Frontmatter["custom_field"] != "keep-me" {
		t.Errorf("custom_field lost: %v", doc2.Frontmatter["custom_field"])
	}
	nested, ok := doc2.Frontmatter["nested"].(map[string]any)
	if !ok || nested["unknown"] != "yes" {
		t.Errorf("nested unknown lost: %v", doc2.Frontmatter["nested"])
	}
	if doc2.Body != "Body content here.\n" {
		t.Errorf("body changed: %q", doc2.Body)
	}
}

func TestEncodeBodyOnly(t *testing.T) {
	out, err := Encode(Document{Body: "hello\n"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "hello\n" {
		t.Errorf("got %q", out)
	}
}

func TestEncodeFrontmatterOnly(t *testing.T) {
	out, err := Encode(Document{
		Frontmatter: map[string]any{"title": "Hi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), "---\n") || !strings.Contains(string(out), "title: Hi") {
		t.Errorf("unexpected output: %q", out)
	}
}

// ── BodySection (L2) ──────────────────────────────────────────────────────────

const sampleProjectBody = `# Description

A local-first TUI for managing sparks and projects.

# Architecture

Clean Go with Bubble Tea at the edge.

## Sub-heading

Sub-heading content preserved by goldmark.

# Roadmap

- M11: settings
- M12: responsive layout
`

func TestBodySectionDescription(t *testing.T) {
	got := BodySection(sampleProjectBody, "Description")
	if !strings.Contains(got, "local-first TUI") {
		t.Errorf("expected description content, got %q", got)
	}
	// Must NOT bleed into Architecture.
	if strings.Contains(got, "Architecture") {
		t.Errorf("section bled into next heading: %q", got)
	}
}

func TestBodySectionArchitecture(t *testing.T) {
	got := BodySection(sampleProjectBody, "Architecture")
	if !strings.Contains(got, "Bubble Tea") {
		t.Errorf("expected architecture content, got %q", got)
	}
	// Sub-headings inside the section should be included.
	if !strings.Contains(got, "Sub-heading content preserved") {
		t.Errorf("sub-heading content dropped: %q", got)
	}
}

func TestBodySectionRoadmap(t *testing.T) {
	got := BodySection(sampleProjectBody, "Roadmap")
	if !strings.Contains(got, "M11") {
		t.Errorf("expected roadmap items, got %q", got)
	}
}

func TestBodySectionMissing(t *testing.T) {
	got := BodySection(sampleProjectBody, "Nonexistent")
	if got != "" {
		t.Errorf("expected empty string for missing section, got %q", got)
	}
}

func TestBodySectionCaseInsensitive(t *testing.T) {
	got := BodySection(sampleProjectBody, "DESCRIPTION")
	if !strings.Contains(got, "local-first TUI") {
		t.Errorf("case-insensitive match failed, got %q", got)
	}
}

func TestBodySectionEmpty(t *testing.T) {
	got := BodySection("", "Description")
	if got != "" {
		t.Errorf("expected empty on empty body, got %q", got)
	}
}
