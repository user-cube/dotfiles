package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// entry is one file of the archive; files are small, so they are read whole.
type entry struct {
	name string // relative to $HOME
	mode fs.FileMode
	data []byte
}

// writeArchive packs home's files (paths relative to it) into a .tar.gz at
// path, plus the MCP servers as mcpEntry when there are any. Same layout as the
// old `tar czf -C ~`, so backups made by local-files.sh still restore.
func writeArchive(path, home string, files []string, servers map[string]json.RawMessage) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	for _, name := range files {
		data, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			return err
		}
		fi, err := os.Stat(filepath.Join(home, name))
		if err != nil {
			return err
		}
		if err := addFile(tw, name, fi.Mode().Perm(), fi.ModTime(), data); err != nil {
			return err
		}
	}
	if len(servers) > 0 {
		data, err := json.MarshalIndent(map[string]any{"mcpServers": servers}, "", "  ")
		if err != nil {
			return err
		}
		if err := addFile(tw, mcpEntry, 0o600, time.Now(), data); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return f.Close()
}

func addFile(tw *tar.Writer, name string, mode fs.FileMode, mtime time.Time, data []byte) error {
	hdr := &tar.Header{
		Name:     name,
		Mode:     int64(mode),
		Size:     int64(len(data)),
		ModTime:  mtime,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

// readArchive returns the regular files of the .tar.gz at path. Directories,
// pax headers and macOS "._" metadata files are skipped; a path that would
// land outside $HOME is an error.
func readArchive(path string) ([]entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	tr := tar.NewReader(gz)

	var entries []entry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if hdr.Typeflag != tar.TypeReg || strings.HasPrefix(filepath.Base(hdr.Name), "._") {
			continue
		}
		name := filepath.Clean(hdr.Name)
		if !filepath.IsLocal(name) {
			return nil, fmt.Errorf("%s: unsafe path %q in the archive", path, hdr.Name)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry{name: name, mode: fs.FileMode(hdr.Mode).Perm(), data: data})
	}
}
