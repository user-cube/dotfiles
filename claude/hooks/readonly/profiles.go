package main

import (
	"regexp"
	"strings"
)

// program is what one profile allows of one program
type program struct {
	subs   map[string]bool            // the subcommands that read; nil: the program takes no subcommand
	deeper map[string]map[string]bool // a subcommand that needs one more word: helm dependency list
	flags  []flag                     // denied after the subcommand (up to a bare --), by their start
	arg    func(word string) string   // why an argument may not be passed, "" to let it through
}

type flag struct{ prefix, why string }

// profiles are added together by name: `readonly git helm`
var profiles = map[string]map[string]program{
	// git that only reads
	"git": {
		"git": {
			subs: set("diff log show status ls-files ls-tree blame rev-parse merge-base shortlog describe"),
			flags: []flag{
				{"--output", "writes a file"},
				{"--ext-diff", "runs a program"},
				{"--textconv", "runs a program"},
				{"-O", "runs a program"},
				{"--open-files-in-pager", "runs a program"},
				{"--exec", "runs a program"},
			},
		},
	},

	// helm that renders and lints a chart on this machine: nothing that talks to a
	// cluster, fetches from the network, writes a file or runs a program
	"helm": {
		"helm": {
			subs:   set("lint template version dependency"),
			deeper: map[string]map[string]bool{"dependency": set("list")},
			flags: []flag{
				{"--validate", "asks the cluster"},
				{"--dry-run", "asks the cluster"},
				{"--kube", "points at a cluster (--kube-context, --kubeconfig, --kube-apiserver, --kube-token...)"},
				{"--output-dir", "writes files"},
				{"--post-renderer", "runs a program"},
				{"--set-file", "reads a file the Read tool's guards never see"},
				{"--repo", "fetches from the network"},
				{"--username", "is for a remote repository"},
				{"--password", "is for a remote repository"},
				{"--key-file", "reads a key"},
				{"--keyring", "reads a key"},
			},
			arg: sensitiveArg,
		},
	},
}

func set(words string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

// An encrypted file under secrets/ is fine to name: what it holds is ENC[...]
var encryptedName = regexp.MustCompile(`\.(enc|encrypted|sops)\.(ya?ml|json|hcl|env|ini)`)

var (
	envTemplate = map[string]bool{"example": true, "sample": true, "template": true, "dist": true}
	ageKeyName  = regexp.MustCompile(`^age-key[^/]*\.txt$|\.agekey$`)
)

// sensitiveArg says why a word may not be an argument of a command that reads
// files by itself (helm -f, a chart path): a remote URL, or a path to what
// deny-sensitive-reads would not let Claude read with its own tools. "" when
// it may. A --flag=value is judged by its value.
func sensitiveArg(w string) string {
	if strings.HasPrefix(w, "-") {
		i := strings.IndexByte(w, '=')
		if i < 0 {
			return ""
		}
		w = w[i+1:]
	}
	if strings.Contains(w, "://") {
		return "a remote chart or URL: this agent works on what is on this machine"
	}
	p := strings.ToLower(strings.TrimRight(w, "."))
	segs := strings.Split(p, "/")
	base := segs[len(segs)-1]
	for _, s := range segs {
		switch s {
		case ".ssh":
			return "an ssh path"
		case ".aws":
			return "an AWS path"
		}
	}
	switch {
	case isEnvFile(base):
		return "an env file"
	case strings.HasSuffix(base, ".pem"), strings.HasSuffix(base, ".key"):
		return "a key or certificate file"
	case ageKeyName.MatchString(base), strings.HasSuffix(p, "sops/age/keys.txt"):
		return "an age key"
	}
	for _, s := range segs[:len(segs)-1] {
		if (s == "secrets" || s == "credentials") && !encryptedName.MatchString(base) {
			return "a file under " + s + "/ that is not marked as encrypted"
		}
	}
	return ""
}

// .env, .envrc, .env.local, .env-prod: env files; .env.example and other templates are not
func isEnvFile(base string) bool {
	rest, ok := strings.CutPrefix(base, ".env")
	switch {
	case !ok:
		return false
	case rest == "" || rest == "rc":
		return true
	case strings.ContainsAny(rest[:1], ".-_"):
		for _, part := range strings.FieldsFunc(rest, func(r rune) bool { return r == '.' || r == '-' || r == '_' }) {
			if envTemplate[part] {
				return false
			}
		}
		return true
	}
	return false // .environment
}
