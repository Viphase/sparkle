package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/viphase/sparkle/internal/domain"
)

// Store reads and writes Markdown files inside a workspace root.
type Store struct {
	Root string
}

// NewStore constructs a Store rooted at the given workspace directory.
func NewStore(root string) *Store { return &Store{Root: root} }

// SparksDir returns the directory where spark files live.
func (s *Store) SparksDir() string { return filepath.Join(s.Root, "sparks") }

// SparkPath returns the file path for a spark by id.
func (s *Store) SparkPath(id string) string {
	return filepath.Join(s.SparksDir(), id+".md")
}

// SaveSpark writes the spark to disk. If a file already exists at the spark's
// path, unknown frontmatter fields and any body that lives there are merged so
// nothing the user wrote by hand gets lost.
func (s *Store) SaveSpark(spark domain.Spark) error {
	if spark.ID == "" {
		return fmt.Errorf("spark id required")
	}
	if !spark.Status.Valid() {
		return fmt.Errorf("invalid spark status %q", spark.Status)
	}

	path := s.SparkPath(spark.ID)
	doc := Document{Frontmatter: map[string]any{}}
	if raw, err := os.ReadFile(path); err == nil {
		if existing, perr := Parse(raw); perr == nil {
			doc = existing
		}
	}
	doc = applySparkToDocument(doc, spark)

	out, err := Encode(doc)
	if err != nil {
		return fmt.Errorf("encode spark: %w", err)
	}
	return WriteAtomic(path, out, 0o644)
}

// LoadSpark reads a spark by id.
func (s *Store) LoadSpark(id string) (domain.Spark, error) {
	raw, err := os.ReadFile(s.SparkPath(id))
	if err != nil {
		return domain.Spark{}, err
	}
	doc, err := Parse(raw)
	if err != nil {
		return domain.Spark{}, fmt.Errorf("parse %s: %w", id, err)
	}
	return documentToSpark(doc)
}

// DeleteSpark removes a spark file from disk.
func (s *Store) DeleteSpark(id string) error {
	if id == "" {
		return fmt.Errorf("spark id required")
	}
	path := s.SparkPath(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete spark: %w", err)
	}
	return nil
}

// ListSparks loads every well-formed spark file in the workspace, sorted by
// CreatedAt descending. Unparseable files are skipped silently — a UI can
// surface these later via a separate scan.
func (s *Store) ListSparks() ([]domain.Spark, error) {
	dir := s.SparksDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]domain.Spark, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		doc, err := Parse(raw)
		if err != nil {
			continue
		}
		sp, err := documentToSpark(doc)
		if err != nil {
			continue
		}
		out = append(out, sp)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
