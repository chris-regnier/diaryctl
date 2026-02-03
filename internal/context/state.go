package context

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const activeContextsFile = "active-contexts.json"

// LoadManualContexts reads the active manual context names from the state file.
func LoadManualContexts(dataDir string) ([]string, error) {
	path := filepath.Join(dataDir, activeContextsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, err
	}
	return names, nil
}

// SetManualContext adds a context name to the active list (idempotent).
func SetManualContext(dataDir string, name string) error {
	names, err := LoadManualContexts(dataDir)
	if err != nil {
		return err
	}
	for _, n := range names {
		if n == name {
			return nil // already set
		}
	}
	names = append(names, name)
	return writeManualContexts(dataDir, names)
}

// UnsetManualContext removes a context name from the active list.
func UnsetManualContext(dataDir string, name string) error {
	names, err := LoadManualContexts(dataDir)
	if err != nil {
		return err
	}
	var filtered []string
	for _, n := range names {
		if n != name {
			filtered = append(filtered, n)
		}
	}
	return writeManualContexts(dataDir, filtered)
}

func writeManualContexts(dataDir string, names []string) error {
	if names == nil {
		names = []string{}
	}
	data, err := json.MarshalIndent(names, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dataDir, activeContextsFile)
	return os.WriteFile(path, data, 0644)
}
