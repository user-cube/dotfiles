package main

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/fatih/color"
)

const encryptedYAML = "password: ENC[AES256_GCM,data:abc,iv:x,tag:y,type:str]\n" +
	"sops:\n    mac: ENC[AES256_GCM,data:def,iv:x,tag:y,type:str]\n    version: 3.9.0\n"

// index fakes the staged files
type index map[string]string

func (ix index) read(p string) ([]byte, error) {
	s, ok := ix[p]
	if !ok {
		return nil, os.ErrNotExist
	}
	return []byte(s), nil
}

func TestYAMLScalar(t *testing.T) {
	for in, want := range map[string]string{
		`secrets/.*\.yaml$`:             `secrets/.*\.yaml$`,
		`'secrets/.*\.yaml$'`:           `secrets/.*\.yaml$`,
		`'it''s\.yaml$'  # a comment`:   `it's\.yaml$`,
		`"values-prod\\.yaml$"  # prod`: `values-prod\.yaml$`,
		`"values-prod\.yaml$"`:          `values-prod\.yaml$`, // not a YAML escape: kept
		`"say \"hi\""`:                  `say "hi"`,
		`prod\.yaml$ # comment`:         `prod\.yaml$`,
		`# only a comment`:              ``,
		`'a#b'`:                         `a#b`,
	} {
		if got := yamlScalar(in); got != want {
			t.Errorf("yamlScalar(%s) = %q, want %q", in, got, want)
		}
	}
}

func TestPathRegexes(t *testing.T) {
	config := "creation_rules:\n" +
		"  - path_regex: 'secrets/.*\\.yaml$'\n    age: age1x\n" +
		"  - age: age1y\n    path_regex: \"values-prod\\\\.yaml$\"  # prod only\r\n" +
		"  # - path_regex: commented\\.yaml$\n"
	want := []string{`secrets/.*\.yaml$`, `values-prod\.yaml$`}
	if got := pathRegexes(config); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name  string
		index index
		want  []string
	}{
		{"a cleartext sops file", index{"k8s/secrets.sops.yaml": "password: hunter2\n"}, []string{"k8s/secrets.sops.yaml"}},
		{"the encrypted formats", index{
			"secrets.sops.yaml": encryptedYAML,
			"app.enc.json":      `{"a": "ENC[x]", "sops": {"mac": "ENC[AES256_GCM,data:z]"}}`,
			"prod.sops.env":     "A=ENC[x]\nsops_mac=ENC[AES256_GCM,data:z]\n",
		}, nil},
		{"other files", index{"values.yaml": "password: x\n"}, nil},
		{"the path_regex of .sops.yaml", index{
			".sops.yaml":            "creation_rules:\n  - path_regex: 'secrets/.*\\.yaml$'\n",
			"secrets/db.yaml":       "password: hunter2\n",
			"secrets/ok.yaml":       encryptedYAML,
			"chart/values-dev.yaml": "token: x\n",
		}, []string{"secrets/db.yaml"}},
		// sops reads RE2: neither \d nor (?i) is POSIX ERE
		{"RE2 syntax", index{
			".sops.yaml": "creation_rules:\n  - path_regex: '(?i)env\\d\\.yaml$'\n",
			"ENV1.yaml":  "a: 1\n",
			"env.yaml":   "a: 1\n",
		}, []string{"ENV1.yaml"}},
		{"a .sops.yaml in a folder, relative to it", index{
			"apps/api/.sops.yaml":         "creation_rules:\n  - path_regex: ^values\\.yaml$\n",
			"apps/api/values.yaml":        "token: x\n",
			"apps/api/charts/values.yaml": "token: x\n", // ^ anchors to apps/api/
			"values.yaml":                 "token: x\n", // not under apps/api
		}, []string{"apps/api/values.yaml"}},
		{"the root's and a folder's", index{
			".sops.yaml":       "creation_rules:\n  - path_regex: prod/\n",
			"apps/.sops.yaml":  "creation_rules:\n  - path_regex: dev\\.yaml$\n",
			"apps/prod/x.yaml": "a: 1\n",
			"apps/dev.yaml":    "a: 1\n",
		}, []string{"apps/prod/x.yaml", "apps/dev.yaml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var files, configs []string
			for p := range tt.index {
				files = append(files, p)
				if strings.HasSuffix(p, configName) {
					configs = append(configs, p)
				}
			}
			bad, err := check(files, configs, tt.index.read, &bytes.Buffer{})
			if err != nil {
				t.Fatal(err)
			}
			if !sameSet(paths(bad), tt.want) {
				t.Errorf("got %q, want %q", bad, tt.want)
			}
		})
	}
}

func TestABadRegexIsReportedAndSkipped(t *testing.T) {
	ix := index{
		".sops.yaml": "creation_rules:\n  - path_regex: '(?<=x)y'\n  - path_regex: a\\.yaml$\n",
		"a.yaml":     "a: 1\n",
	}
	var warn bytes.Buffer
	bad, err := check([]string{"a.yaml"}, []string{".sops.yaml"}, ix.read, &warn)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths(bad), []string{"a.yaml"}) || !strings.Contains(warn.String(), "(?<=x)y") {
		t.Errorf("bad %q, warning %q", bad, warn.String())
	}
}

func TestWhyEncrypt(t *testing.T) {
	rules := loadRules([]string{"apps/.sops.yaml"},
		index{"apps/.sops.yaml": "creation_rules:\n  - path_regex: dev\\.yaml$\n"}.read, &bytes.Buffer{})
	for p, want := range map[string]string{
		"x.sops.yaml":   "named *.sops.*",
		"app.enc.env":   "named *.enc.*",
		"apps/dev.yaml": `path_regex dev\.yaml$ of apps/.sops.yaml`,
		"dev.yaml":      "",
	} {
		if got := whyEncrypt(p, rules); got != want {
			t.Errorf("whyEncrypt(%s) = %q, want %q", p, got, want)
		}
	}
}

// Without a terminal (git from lazygit, VS Code, CI): no escape codes, no emojis
func TestReportIsPlainWithoutATerminal(t *testing.T) {
	color.NoColor, emoji = true, false
	var out bytes.Buffer
	report(&out, []finding{{"k8s/db.yaml", "path_regex k8s/ of .sops.yaml"}, {"a.sops.env", "named *.sops.*"}})
	want := "sops-guard: 2 files sops should encrypt staged in cleartext\n" +
		"   ✗ k8s/db.yaml  path_regex k8s/ of .sops.yaml\n" +
		"   ✗ a.sops.env   named *.sops.*\n" +
		"sops encrypt -i FILE, then git add FILE · skip once: git commit --no-verify\n"
	if out.String() != want {
		t.Errorf("got\n%s\nwant\n%s", out.String(), want)
	}
}

func TestReportColorsOnATerminal(t *testing.T) {
	color.NoColor, emoji = false, true
	t.Cleanup(func() { color.NoColor, emoji = true, false })
	var out bytes.Buffer
	report(&out, []finding{{"a.sops.env", "named *.sops.*"}})
	if s := out.String(); !strings.Contains(s, "\x1b[") || !strings.HasPrefix(s, "🔒 ") || !strings.Contains(s, "💡 ") {
		t.Errorf("no colors or emojis: %q", s)
	}
}

func TestNoColorTurnsThemOff(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	setup()
	if !color.NoColor || emoji {
		t.Error("NO_COLOR set, colors or emojis still on")
	}
}

func paths(fs []finding) []string {
	var ps []string
	for _, f := range fs {
		ps = append(ps, f.path)
	}
	return ps
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := map[string]int{}
	for _, s := range a {
		seen[s]++
	}
	for _, s := range b {
		if seen[s]--; seen[s] < 0 {
			return false
		}
	}
	return true
}
