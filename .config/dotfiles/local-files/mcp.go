package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The servers are kept as raw JSON: their config (command, args, env) is
// copied as is, never looked into.
type mcpServers = map[string]json.RawMessage

// readMCPServers returns the mcpServers of ~/.claude.json; nil when the file
// or the key is missing.
func readMCPServers(path string) (mcpServers, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var doc struct {
		MCPServers mcpServers `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return doc.MCPServers, nil
}

// mergeServers adds the saved servers to the ones here. A server already here
// wins, unless force; kept lists those, sorted.
func mergeServers(here, saved mcpServers, force bool) (merged mcpServers, kept []string) {
	merged = maps.Clone(here)
	if merged == nil {
		merged = mcpServers{}
	}
	for name, cfg := range saved {
		if _, ok := here[name]; ok && !force {
			kept = append(kept, name)
			continue
		}
		merged[name] = cfg
	}
	slices.Sort(kept)
	return merged, kept
}

// restoreMCP merges the archive's mcpEntry (data) into ~/.claude.json at path.
// Every other key of the file (login, projects, ...) is left as it was.
func restoreMCP(path string, data []byte, force bool, out io.Writer) error {
	var saved struct {
		MCPServers mcpServers `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return fmt.Errorf("%s in the archive: %w", mcpEntry, err)
	}
	names := strings.Join(sortedKeys(saved.MCPServers), ", ")

	doc := map[string]json.RawMessage{}
	if raw, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var here mcpServers
	if raw, ok := doc["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &here); err != nil {
			return fmt.Errorf("%s: mcpServers: %w", path, err)
		}
	}

	if len(saved.MCPServers) == 0 {
		return nil
	}
	merged, kept := mergeServers(here, saved.MCPServers, force)
	if len(kept) == len(saved.MCPServers) { // nothing new: the file isn't rewritten
		fmt.Fprintf(out, "  ~/%s  (MCP servers %s: exists, kept; --force to overwrite)\n", claudeJSON, names)
		return nil
	}
	mergedJSON, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	doc["mcpServers"] = mergedJSON
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(path, b, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(out, "  ~/%s  (MCP servers: %s)\n", claudeJSON, names)
	if len(kept) > 0 {
		fmt.Fprintf(out, "    already here, kept: %s (--force to overwrite)\n", strings.Join(kept, ", "))
	}
	return nil
}

// writeAtomic replaces path in one rename, so a running Claude Code never
// reads a half-written ~/.claude.json.
func writeAtomic(path string, data []byte, mode fs.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func sortedKeys(m mcpServers) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
