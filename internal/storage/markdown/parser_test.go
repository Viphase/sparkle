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
