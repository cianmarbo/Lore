package server

import (
	"os"
	"path/filepath"
)

// findSessionFile searches common Claude Code session file locations for a given session ID.
func findSessionFile(sessionID string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	filename := sessionID + ".jsonl"

	// Search in Claude projects directories
	projectsDir := filepath.Join(home, ".claude", "projects")
	var found string

	filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == filename {
			found = path
			return filepath.SkipAll
		}
		return nil
	})

	if found != "" {
		return found
	}

	// Also check ~/.claude/sessions/ (older format)
	sessionsPath := filepath.Join(home, ".claude", "sessions", filename)
	if _, err := os.Stat(sessionsPath); err == nil {
		return sessionsPath
	}

	return ""
}
