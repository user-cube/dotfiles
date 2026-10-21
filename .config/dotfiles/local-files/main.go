// local-files: backup / restore of the per-machine files that never go to git,
// as one .tar.gz attached to a Bitwarden secure note "Local files (<machine>)".
//
//	local-files backup                      this machine -> Bitwarden (replaces the old copy)
//	local-files restore [machine] [--force] Bitwarden -> this machine; existing files
//	                                        are kept unless --force
//	local-files list                        machines with a backup
//
// Bitwarden session: BW_SESSION if exported, else the one kept by the dotfiles
// `bw` wrapper (/var/root/.bitwarden.session, read with sudo / Touch ID).
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Paths relative to $HOME, globs allowed; missing ones are skipped
var files = []string{
	".zshrc.local",
	".gitconfig-work",
	".gitconfig-hook",
	".config/tmux/sessions.local",
	".config/sketchybar/checks",
	".config/sketchybar/aws_profiles",
	".kube/config",
	".terraformrc",
	".terraform.d/credentials.tfrc.json",
	"workspaces/*.code-workspace", // VS Code workspaces, see `ws`
}

const (
	notePrefix = "Local files"
	attachment = "local-files.tar.gz"
	// Claude Code's MCP servers (tokens in their env) live in ~/.claude.json next
	// to the login and per-project state: only the mcpServers key is saved, under
	// mcpEntry in the archive, and merged back on restore (see mcp.go).
	claudeJSON = ".claude.json"
	mcpEntry   = ".claude/mcp-servers.json"
)

const usage = `usage:
  local-files backup                      this machine -> Bitwarden (replaces the old copy)
  local-files restore [machine] [--force] Bitwarden -> this machine; existing files
                                          are kept unless --force
  local-files list                        machines with a backup
`

// vault is what backup / restore need from Bitwarden; bwCLI (bw.go) is the
// real one, the tests use one in memory.
type vault interface {
	Sync() error
	FindItem(name string) (id string, err error) // "" when there is none
	ListNames(prefix string) ([]string, error)
	CreateNote(name, notes string) (id string, err error)
	ReplaceAttachment(itemID, fileName, path string) error
	DownloadAttachment(itemID, fileName, dest string) error
}

var errUsage = errors.New("usage")

func main() {
	err := run(os.Args[1:], os.Stdout)
	if errors.Is(err, errUsage) {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "local-files:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errUsage
	}
	cmd, args := args[0], args[1:]
	if cmd != "backup" && cmd != "restore" && cmd != "list" {
		return errUsage
	}
	machine, force, err := parseArgs(cmd, args)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	v, err := newBW()
	if err != nil {
		return err
	}

	switch cmd {
	case "backup":
		return backup(v, home, thisMachine(), out)
	case "restore":
		if machine == "" {
			machine = thisMachine()
		}
		return restore(v, home, machine, force, out)
	default:
		machines, err := listMachines(v)
		if err != nil {
			return err
		}
		for _, m := range machines {
			fmt.Fprintln(out, m)
		}
		return nil
	}
}

// parseArgs reads restore's [machine] [--force], in either order; backup and
// list take nothing.
func parseArgs(cmd string, args []string) (machine string, force bool, err error) {
	for _, a := range args {
		switch {
		case cmd == "restore" && a == "--force":
			force = true
		case cmd == "restore" && a != "" && machine == "" && !strings.HasPrefix(a, "-"):
			machine = a
		case a == "":
			// the playbook passes an empty machine when none is given
		default:
			return "", false, errUsage
		}
	}
	return machine, force, nil
}

// thisMachine is the Bonjour name (Ruis-MacBook-Pro), as in the note names.
func thisMachine() string {
	if out, err := exec.Command("scutil", "--get", "LocalHostName").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	host, _ := os.Hostname()
	return strings.Split(host, ".")[0]
}

func noteName(machine string) string { return notePrefix + " (" + machine + ")" }

func backup(v vault, home, machine string, out io.Writer) error {
	var present []string
	for _, pattern := range files {
		matches, _ := filepath.Glob(filepath.Join(home, pattern))
		for _, m := range matches {
			if fi, err := os.Stat(m); err == nil && fi.Mode().IsRegular() {
				rel, _ := filepath.Rel(home, m)
				present = append(present, rel)
			}
		}
	}
	if len(present) == 0 {
		return errors.New("none of the files exist on this machine")
	}
	servers, err := readMCPServers(filepath.Join(home, claudeJSON))
	if err != nil {
		return err
	}

	// The archive (MCP tokens included) only ever exists in this temp dir
	tmp, err := os.MkdirTemp("", "local-files")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archive := filepath.Join(tmp, attachment)
	if err := writeArchive(archive, home, present, servers); err != nil {
		return err
	}

	if err := v.Sync(); err != nil {
		return err
	}
	name := noteName(machine)
	id, err := v.FindItem(name)
	if err != nil {
		return err
	}
	if id == "" {
		if id, err = v.CreateNote(name, "Per-machine dotfiles, restore with: local-restore"); err != nil {
			return err
		}
		fmt.Fprintf(out, "created Bitwarden item: %s\n", name)
	}
	if err := v.ReplaceAttachment(id, attachment, archive); err != nil {
		return err
	}

	fmt.Fprintf(out, "backed up to %q:\n", name)
	for _, f := range present {
		fmt.Fprintf(out, "  ~/%s\n", f)
	}
	if len(servers) > 0 {
		fmt.Fprintf(out, "  ~/%s  (MCP servers only: %s)\n", claudeJSON, strings.Join(sortedKeys(servers), ", "))
	}
	return nil
}

// restore writes the files of machine's backup into home. Its output lines
// start with "  ~/", and say "exists, kept" when a file was left alone: the
// playbook's changed_when (roles/system/tasks/local-files.yml) reads them.
func restore(v vault, home, machine string, force bool, out io.Writer) error {
	if err := v.Sync(); err != nil {
		return err
	}
	name := noteName(machine)
	id, err := v.FindItem(name)
	if err != nil {
		return err
	}
	if id == "" {
		machines, _ := listMachines(v)
		return fmt.Errorf("no backup %q; machines with one: %s", name, strings.Join(machines, " "))
	}

	tmp, err := os.MkdirTemp("", "local-files")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archive := filepath.Join(tmp, attachment)
	if err := v.DownloadAttachment(id, attachment, archive); err != nil {
		return err
	}
	entries, err := readArchive(archive)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "restoring %q:\n", name)
	for _, e := range entries {
		if e.name == mcpEntry {
			if err := restoreMCP(filepath.Join(home, claudeJSON), e.data, force, out); err != nil {
				return err
			}
			continue
		}
		dest := filepath.Join(home, e.name)
		if _, err := os.Lstat(dest); err == nil && !force {
			fmt.Fprintf(out, "  ~/%s  (exists, kept; --force to overwrite)\n", e.name)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, e.data, e.mode); err != nil {
			return err
		}
		if err := os.Chmod(dest, e.mode); err != nil { // WriteFile keeps an existing file's mode
			return err
		}
		fmt.Fprintf(out, "  ~/%s\n", e.name)
	}
	return nil
}

func listMachines(v vault) ([]string, error) {
	if err := v.Sync(); err != nil {
		return nil, err
	}
	names, err := v.ListNames(notePrefix + " (")
	if err != nil {
		return nil, err
	}
	var machines []string
	for _, n := range names {
		machines = append(machines, strings.TrimSuffix(strings.TrimPrefix(n, notePrefix+" ("), ")"))
	}
	return machines, nil
}
