package markdown

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/viphase/sparkle/internal/domain"
	"github.com/viphase/sparkle/internal/workspace"
)

const sessionsSubDir = "sessions"
const sessionMaxLoad = 40 // max turns loaded on project open

// SessionsDir returns the path to the sessions directory.
func SessionsDir(root string) string {
	return filepath.Join(root, workspace.MetaDirName, sessionsSubDir)
}

// sessionPath returns the JSONL file path for a project's conversation.
func sessionPath(root, projectID string) string {
	return filepath.Join(SessionsDir(root), projectID+".jsonl")
}

// sessionRecord is one line in the JSONL file.
type sessionRecord struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AppendSessionTurn appends a single message turn to the project's session log.
// The file is created if it does not exist; the directory is created as needed.
func AppendSessionTurn(root, projectID string, msg domain.Message) error {
	if err := os.MkdirAll(SessionsDir(root), 0o755); err != nil {
		return fmt.Errorf("mkdir sessions: %w", err)
	}
	path := sessionPath(root, projectID)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	rec := sessionRecord{Role: string(msg.Role), Content: msg.Content}
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal turn: %w", err)
	}
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// LoadSession reads up to sessionMaxLoad recent turns from the project's session
// log. Returns an empty slice (not an error) if the file does not exist.
func LoadSession(root, projectID string) ([]domain.Message, error) {
	path := sessionPath(root, projectID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open session: %w", err)
	}
	defer f.Close()

	// Collect all lines, then return the last N.
	var recs []sessionRecord
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec sessionRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue // skip malformed lines
		}
		recs = append(recs, rec)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}

	start := 0
	if len(recs) > sessionMaxLoad {
		start = len(recs) - sessionMaxLoad
	}
	recs = recs[start:]

	msgs := make([]domain.Message, 0, len(recs))
	for _, r := range recs {
		msgs = append(msgs, domain.Message{
			Role:    domain.MessageRole(r.Role),
			Content: r.Content,
		})
	}
	return msgs, nil
}
