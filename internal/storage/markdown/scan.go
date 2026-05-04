package markdown

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// ScanIndexData is the on-disk representation of the file-scan index.
// It mirrors tracker.ScanIndex without creating a dependency on that package.
type ScanIndexData struct {
	FileMtime map[string]time.Time `json:"file_mtime"`
	FileWords map[string]int       `json:"file_words"`
}

// ScanIndexPath returns the path to the persisted scan index file.
func (s *Store) ScanIndexPath() string {
	return filepath.Join(s.Root, ".sparkle", "scan-index.json")
}

// LoadScanIndex reads the persisted file-scan index.
// Returns an empty index without error when the file does not exist.
func (s *Store) LoadScanIndex() (ScanIndexData, error) {
	path := s.ScanIndexPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newScanIndexData(), nil
		}
		return newScanIndexData(), err
	}
	var idx ScanIndexData
	if err := json.Unmarshal(raw, &idx); err != nil {
		return newScanIndexData(), err
	}
	if idx.FileMtime == nil {
		idx.FileMtime = make(map[string]time.Time)
	}
	if idx.FileWords == nil {
		idx.FileWords = make(map[string]int)
	}
	return idx, nil
}

// SaveScanIndex writes the scan index atomically.
func (s *Store) SaveScanIndex(idx ScanIndexData) error {
	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return WriteAtomic(s.ScanIndexPath(), data, 0o644)
}

func newScanIndexData() ScanIndexData {
	return ScanIndexData{
		FileMtime: make(map[string]time.Time),
		FileWords: make(map[string]int),
	}
}
