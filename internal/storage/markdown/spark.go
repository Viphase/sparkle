package markdown

import (
	"fmt"
	"time"

	"github.com/viphase/sparkle/internal/domain"
)

// SparkSchemaVersion is bumped whenever the on-disk shape of a spark file changes.
const SparkSchemaVersion = 1

// applySparkToDocument writes Spark fields onto the document's frontmatter
// without disturbing unknown keys, and replaces the body with the description.
func applySparkToDocument(doc Document, s domain.Spark) Document {
	if doc.Frontmatter == nil {
		doc.Frontmatter = map[string]any{}
	}
	fm := doc.Frontmatter
	fm["schema_version"] = SparkSchemaVersion
	fm["id"] = s.ID
	fm["title"] = s.Title
	fm["status"] = string(s.Status)
	fm["tags"] = s.Tags
	fm["created_at"] = formatTime(s.CreatedAt)
	fm["updated_at"] = formatTime(s.UpdatedAt)
	fm["promoted_project_id"] = s.PromotedProjectID
	doc.Body = s.Description
	return doc
}

func documentToSpark(doc Document) (domain.Spark, error) {
	fm := doc.Frontmatter
	s := domain.Spark{
		ID:                stringField(fm, "id"),
		Title:             stringField(fm, "title"),
		Description:       doc.Body,
		Status:            domain.SparkStatus(stringField(fm, "status")),
		Tags:              stringSliceField(fm, "tags"),
		PromotedProjectID: stringField(fm, "promoted_project_id"),
	}
	if !s.Status.Valid() {
		return domain.Spark{}, fmt.Errorf("invalid spark status %q", s.Status)
	}
	if t, ok, err := timeField(fm, "created_at"); err != nil {
		return domain.Spark{}, fmt.Errorf("created_at: %w", err)
	} else if ok {
		s.CreatedAt = t
	}
	if t, ok, err := timeField(fm, "updated_at"); err != nil {
		return domain.Spark{}, fmt.Errorf("updated_at: %w", err)
	} else if ok {
		s.UpdatedAt = t
	}
	return s, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func stringField(fm map[string]any, key string) string {
	v, ok := fm[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func stringSliceField(fm map[string]any, key string) []string {
	v, ok := fm[key]
	if !ok || v == nil {
		return nil
	}
	switch xs := v.(type) {
	case []any:
		out := make([]string, 0, len(xs))
		for _, x := range xs {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, len(xs))
		copy(out, xs)
		return out
	}
	return nil
}

func timeField(fm map[string]any, key string) (time.Time, bool, error) {
	v, ok := fm[key]
	if !ok || v == nil {
		return time.Time{}, false, nil
	}
	switch x := v.(type) {
	case time.Time:
		return x, true, nil
	case string:
		if x == "" {
			return time.Time{}, false, nil
		}
		t, err := time.Parse(time.RFC3339, x)
		if err != nil {
			return time.Time{}, false, err
		}
		return t, true, nil
	}
	return time.Time{}, false, fmt.Errorf("unexpected type %T", v)
}
