package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// bwCLI is the vault, through the Bitwarden CLI (`bw`).
type bwCLI struct{ session string }

type bwItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Attachments []struct {
		ID       string `json:"id"`
		FileName string `json:"fileName"`
	} `json:"attachments"`
}

func newBW() (*bwCLI, error) {
	if _, err := exec.LookPath("bw"); err != nil {
		return nil, errors.New("bitwarden-cli (bw) is not installed")
	}
	session := os.Getenv("BW_SESSION")
	if session == "" {
		var err error
		if session, err = savedSession(); err != nil {
			return nil, err
		}
	}
	return &bwCLI{session: session}, nil
}

// savedSession reads the session the dotfiles `bw` wrapper keeps root-only
// (sudo asks for Touch ID), then drops sudo's cached credentials.
func savedSession() (string, error) {
	cmd := exec.Command("sudo", "cat", "/var/root/.bitwarden.session")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	_ = exec.Command("sudo", "-k").Run()
	session := strings.TrimSpace(string(out))
	if err != nil || session == "" {
		return "", errors.New("no Bitwarden session: run `bw --regenerate-session-key` or export BW_SESSION")
	}
	return session, nil
}

// run calls bw with the session in its environment, never on the command line.
func (b *bwCLI) run(args ...string) ([]byte, error) {
	cmd := exec.Command("bw", args...)
	cmd.Env = append(os.Environ(), "BW_SESSION="+b.session)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		what := strings.Join(args[:min(2, len(args))], " ")
		return nil, fmt.Errorf("bw %s: %v: %s", what, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (b *bwCLI) Sync() error {
	_, err := b.run("sync")
	return err
}

func (b *bwCLI) search(text string) ([]bwItem, error) {
	out, err := b.run("list", "items", "--search", text)
	if err != nil {
		return nil, err
	}
	var items []bwItem
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("bw list items: %w", err)
	}
	return items, nil
}

// FindItem matches the whole name: --search alone also finds "Local files (A)"
// when looking for "Local files (A-2)".
func (b *bwCLI) FindItem(name string) (string, error) {
	items, err := b.search(name)
	if err != nil {
		return "", err
	}
	for _, it := range items {
		if it.Name == name {
			return it.ID, nil
		}
	}
	return "", nil
}

func (b *bwCLI) ListNames(prefix string) ([]string, error) {
	items, err := b.search(prefix)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, it := range items {
		if strings.HasPrefix(it.Name, prefix) {
			names = append(names, it.Name)
		}
	}
	return names, nil
}

// CreateNote creates a secure note from bw's item template.
func (b *bwCLI) CreateNote(name, notes string) (string, error) {
	tpl, err := b.run("get", "template", "item")
	if err != nil {
		return "", err
	}
	var item map[string]any
	if err := json.Unmarshal(tpl, &item); err != nil {
		return "", fmt.Errorf("bw get template item: %w", err)
	}
	item["type"] = 2 // secure note
	item["secureNote"] = map[string]int{"type": 0}
	item["login"] = nil
	item["name"] = name
	item["notes"] = notes
	encoded, err := json.Marshal(item)
	if err != nil {
		return "", err
	}
	out, err := b.run("create", "item", base64.StdEncoding.EncodeToString(encoded))
	if err != nil {
		return "", err
	}
	var created bwItem
	if err := json.Unmarshal(out, &created); err != nil {
		return "", fmt.Errorf("bw create item: %w", err)
	}
	return created.ID, nil
}

// ReplaceAttachment drops the item's attachments named fileName, then uploads path.
func (b *bwCLI) ReplaceAttachment(itemID, fileName, path string) error {
	out, err := b.run("get", "item", itemID)
	if err != nil {
		return err
	}
	var item bwItem
	if err := json.Unmarshal(out, &item); err != nil {
		return fmt.Errorf("bw get item: %w", err)
	}
	for _, a := range item.Attachments {
		if a.FileName == fileName {
			if _, err := b.run("delete", "attachment", a.ID, "--itemid", itemID); err != nil {
				return err
			}
		}
	}
	_, err = b.run("create", "attachment", "--file", path, "--itemid", itemID)
	return err
}

func (b *bwCLI) DownloadAttachment(itemID, fileName, dest string) error {
	_, err := b.run("get", "attachment", fileName, "--itemid", itemID, "--output", dest)
	return err
}
