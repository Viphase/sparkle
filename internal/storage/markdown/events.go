package markdown

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/viphase/sparkle/internal/domain"
)

// EventsDir returns the directory where per-project event logs live.
func (s *Store) EventsDir() string { return filepath.Join(s.Root, ".sparkle", "events") }

// EventsPath returns the JSONL file path for a project's event log.
func (s *Store) EventsPath(projectID string) string {
	return filepath.Join(s.EventsDir(), projectID+".jsonl")
}

// AppendEvent appends a single TrackingEvent to the project's JSONL log.
// The events directory is created on first write.
func (s *Store) AppendEvent(projectID string, ev domain.TrackingEvent) error {
	path := s.EventsPath(projectID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir events: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open events file: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// LoadEvents reads all events for a project from its JSONL log.
// Missing file returns an empty slice without error.
func (s *Store) LoadEvents(projectID string) ([]domain.TrackingEvent, error) {
	path := s.EventsPath(projectID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open events file: %w", err)
	}
	defer f.Close()

	var events []domain.TrackingEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev domain.TrackingEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue // skip malformed lines
		}
		events = append(events, ev)
	}
	return events, scanner.Err()
}

// LoadAllEvents reads events for every project in the workspace.
// Returns a map from project ID to its events.
func (s *Store) LoadAllEvents() (map[string][]domain.TrackingEvent, error) {
	dir := s.EventsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string][]domain.TrackingEvent{}, nil
		}
		return nil, fmt.Errorf("read events dir: %w", err)
	}

	result := make(map[string][]domain.TrackingEvent, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		id := e.Name()[:len(e.Name())-len(".jsonl")]
		evs, err := s.LoadEvents(id)
		if err != nil {
			continue
		}
		result[id] = evs
	}
	return result, nil
}
