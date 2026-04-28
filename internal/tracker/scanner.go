package tracker

import (
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/viphase/sparkle/internal/domain"
)

// ScanIndex tracks per-file state between scan runs.
type ScanIndex struct {
	// FileMtime maps file path → last-observed mtime.
	FileMtime map[string]time.Time
	// FileWords maps file path → last-observed word count.
	FileWords map[string]int
}

// NewScanIndex returns an empty index.
func NewScanIndex() ScanIndex {
	return ScanIndex{
		FileMtime: map[string]time.Time{},
		FileWords: map[string]int{},
	}
}

// ScanResult holds events produced by a scan and the updated index.
type ScanResult struct {
	Events []domain.TrackingEvent
	Index  ScanIndex
}

// ScanProjectDir walks projectDir (a single project's directory), detects word
// count changes, and returns events that exceed the meaningful threshold.
// wordsThreshold is the minimum absolute delta to emit a words_added/removed event.
// touchWindowSecs is the minimum seconds between file_touched events for the same file.
func ScanProjectDir(
	projectID string,
	projectDir string,
	idx ScanIndex,
	wordsThreshold int,
	touchWindowSecs int,
	now time.Time,
) ScanResult {
	if wordsThreshold <= 0 {
		wordsThreshold = 10
	}
	if touchWindowSecs <= 0 {
		touchWindowSecs = 300
	}
	out := ScanResult{Index: ScanIndex{
		FileMtime: copyMap(idx.FileMtime),
		FileWords: copyMap(idx.FileWords),
	}}

	_ = filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".txt" {
			return nil
		}

		mtime := info.ModTime()
		lastMtime, seen := idx.FileMtime[path]

		if seen && !mtime.After(lastMtime) {
			return nil // no change
		}
		out.Index.FileMtime[path] = mtime

		// file_touched — rate-limited per window
		if !seen || mtime.Sub(lastMtime) >= time.Duration(touchWindowSecs)*time.Second {
			out.Events = append(out.Events, domain.TrackingEvent{
				Timestamp: now,
				Type:      domain.EventFileTouched,
				Value:     1,
				Source:    "auto",
				Note:      filepath.Base(path),
			})
		}

		// word delta
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		newWords := countWords(string(raw))
		oldWords := idx.FileWords[path]
		out.Index.FileWords[path] = newWords

		delta := newWords - oldWords
		if delta > wordsThreshold {
			out.Events = append(out.Events, domain.TrackingEvent{
				Timestamp: now,
				Type:      domain.EventWordsAdded,
				Value:     delta,
				Source:    "auto",
				Note:      filepath.Base(path),
			})
		} else if -delta > wordsThreshold {
			out.Events = append(out.Events, domain.TrackingEvent{
				Timestamp: now,
				Type:      domain.EventWordsRemoved,
				Value:     -delta,
				Source:    "auto",
				Note:      filepath.Base(path),
			})
		}
		return nil
	})

	return out
}

func countWords(s string) int {
	inWord := false
	count := 0
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !inWord {
				count++
				inWord = true
			}
		} else {
			inWord = false
		}
	}
	return count
}

func copyMap[K comparable, V any](m map[K]V) map[K]V {
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
