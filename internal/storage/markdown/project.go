package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/viphase/sparkle/internal/domain"
)

const ProjectSchemaVersion = 1

// ProjectsDir returns the directory where project subdirectories live.
func (s *Store) ProjectsDir() string { return filepath.Join(s.Root, "projects") }

// ProjectDir returns the subdirectory for a single project.
func (s *Store) ProjectDir(id string) string { return filepath.Join(s.ProjectsDir(), id) }

// ProjectPath returns the main project.md path for a project.
func (s *Store) ProjectPath(id string) string {
	return filepath.Join(s.ProjectDir(id), "project.md")
}

// SaveProject writes a project to disk, preserving any existing body the user
// may have written by hand. Notes.md is created alongside project.md if absent.
func (s *Store) SaveProject(p domain.Project) error {
	if p.ID == "" {
		return fmt.Errorf("project id required")
	}
	if !p.Status.Valid() {
		return fmt.Errorf("invalid project status %q", p.Status)
	}
	dir := s.ProjectDir(p.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir project dir: %w", err)
	}
	path := s.ProjectPath(p.ID)
	doc := Document{Frontmatter: map[string]any{}}
	if raw, err := os.ReadFile(path); err == nil {
		if existing, perr := Parse(raw); perr == nil {
			doc = existing
		}
	}
	doc = applyProjectToDocument(doc, p)
	out, err := Encode(doc)
	if err != nil {
		return fmt.Errorf("encode project: %w", err)
	}
	if err := WriteAtomic(path, out, 0o644); err != nil {
		return err
	}
	// Bootstrap notes.md the first time only — never overwrite user content.
	notesPath := filepath.Join(dir, "notes.md")
	if _, err := os.Stat(notesPath); os.IsNotExist(err) {
		title := p.Title
		if title == "" {
			title = p.ID
		}
		notesContent := fmt.Sprintf("# Notes — %s\n\nFreeform notes for this project.\n", title)
		_ = os.WriteFile(notesPath, []byte(notesContent), 0o644)
	}
	return nil
}

// LoadProject reads a project by id.
func (s *Store) LoadProject(id string) (domain.Project, error) {
	raw, err := os.ReadFile(s.ProjectPath(id))
	if err != nil {
		return domain.Project{}, err
	}
	doc, err := Parse(raw)
	if err != nil {
		return domain.Project{}, fmt.Errorf("parse project %s: %w", id, err)
	}
	return documentToProject(doc)
}

// ListProjects loads every project from subdirectories, sorted by CreatedAt
// descending. Unreadable or unparseable project.md files are skipped silently.
func (s *Store) ListProjects() ([]domain.Project, error) {
	dir := s.ProjectsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]domain.Project, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name(), "project.md"))
		if err != nil {
			continue
		}
		doc, err := Parse(raw)
		if err != nil {
			continue
		}
		p, err := documentToProject(doc)
		if err != nil {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func applyProjectToDocument(doc Document, p domain.Project) Document {
	if doc.Frontmatter == nil {
		doc.Frontmatter = map[string]any{}
	}
	fm := doc.Frontmatter
	fm["schema_version"] = ProjectSchemaVersion
	fm["id"] = p.ID
	fm["title"] = p.Title
	fm["status"] = string(p.Status)
	fm["github_url"] = p.GitHubURL
	fm["target_audience"] = p.TargetAudience
	fm["tags"] = p.Tags
	fm["created_at"] = formatTime(p.CreatedAt)
	fm["updated_at"] = formatTime(p.UpdatedAt)
	if doc.Body == "" {
		doc.Body = defaultProjectBody()
	}
	return doc
}

func documentToProject(doc Document) (domain.Project, error) {
	fm := doc.Frontmatter
	p := domain.Project{
		ID:             stringField(fm, "id"),
		Title:          stringField(fm, "title"),
		Status:         domain.ProjectStatus(stringField(fm, "status")),
		GitHubURL:      stringField(fm, "github_url"),
		TargetAudience: stringField(fm, "target_audience"),
		Tags:           stringSliceField(fm, "tags"),
	}
	if !p.Status.Valid() {
		return domain.Project{}, fmt.Errorf("invalid project status %q", p.Status)
	}
	if t, ok, err := timeField(fm, "created_at"); err != nil {
		return domain.Project{}, err
	} else if ok {
		p.CreatedAt = t
	}
	if t, ok, err := timeField(fm, "updated_at"); err != nil {
		return domain.Project{}, err
	} else if ok {
		p.UpdatedAt = t
	}
	return p, nil
}

func defaultProjectBody() string {
	return "# Description\n\n\n# Architecture\n\n\n# Target Audience\n\n\n# Roadmap\n\n\n# Open Questions\n\n"
}
