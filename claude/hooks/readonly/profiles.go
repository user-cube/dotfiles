package main

import (
	"regexp"
	"strings"
)

// program is what one profile allows of one program
type program struct {
	subs   map[string]bool            // the subcommands that read; nil: the program takes no subcommand
	deeper map[string]map[string]bool // a subcommand that needs one more word: helm dependency list
	// the options allowed before the subcommand, by their start (terraform -chdir=dir); any other is denied
	leading []string
	needs   map[string]need          // a subcommand that only reads in a certain mode: terraform fmt -check
	flags   []flag                   // denied after the subcommand (up to a bare --), by their start
	arg     func(word string) string // why an argument may not be passed, "" to let it through
}

type flag struct{ prefix, why string }

// need: at least one of the flags (by their start) must be there, else why
type need struct {
	anyOf []string
	why   string
}

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

	// ansible that parses and lints a playbook, role or inventory on this machine: nothing
	// that connects to a host, runs a task, decrypts a vault or installs from the network
	"ansible": ansibleProfile,

	// terraform and OpenTofu that check the code: format in check mode, validate, and
	// the static linters. Nothing that plans, applies, initialises (downloads
	// providers and modules), reads state or runs infrastructure tests
	"terraform": {
		"terraform": terraformProgram,
		"tofu":      terraformProgram,
		"tflint": {
			flags: []flag{
				{"--init", "downloads plugins and writes them"},
				{"--fix", "rewrites the files"},
				{"--langserver", "starts a server"},
			},
			arg: sensitiveArg,
		},
		"trivy": trivyProgram,
	},

	// Kubernetes manifests and kustomize overlays, checked on this machine: schema
	// validation and the static linters, and kustomize build to see the final
	// output. No kubectl, not even --dry-run=client: it can ask the API server
	"k8s": {
		"kubeconform": {
			// it fetches the schemas of the Kubernetes API over the network (a read);
			// the cache is a directory it writes
			flags: []flag{{"-cache", "writes a cache directory"}},
			arg:   sensitiveArg,
		},
		"kustomize": {
			subs: set("build version"),
			flags: []flag{
				{"--enable-", "turns on plugins or helm, which run programs"},
				{"--load-restrictor", "reads outside the kustomization root"},
				{"--output", "writes files"},
				{"-o", "writes files"},
				{"--helm-command", "runs a program"},
				{"--network", "runs containers"},
				{"--mount", "runs containers"},
				{"--env", "passes the environment to a plugin"},
				{"-e", "passes the environment to a plugin"},
			},
			arg: sensitiveArg,
		},
		"kube-linter": {subs: set("lint version"), arg: sensitiveArg},
		"kube-score":  {subs: set("score version"), arg: sensitiveArg},
		"trivy":       trivyProgram,
	},
}

var ansibleProfile = map[string]program{
	// the playbook is parsed, never run: one of the modes that stop before any task
	// does, so nothing connects to a host. -i is denied too: an inventory can be a
	// script, and it is run to be read
	"ansible-playbook": {
		needs: map[string]need{
			"": {anyOf: []string{"--syntax-check", "--list-tasks", "--list-tags", "--list-hosts", "--version"},
				why: "only runs in a mode that stops before any task: --syntax-check, --list-tasks, --list-tags or --list-hosts"},
		},
		flags: []flag{
			{"--check", "connects to the hosts"},
			{"-C", "connects to the hosts"},
			{"-i", "can run a script (an inventory can be one)"},
			{"--inventory", "can run a script (an inventory can be one)"},
			{"--vault", "decrypts secrets"},
			{"--ask-vault", "decrypts secrets"},
			{"--ask-", "asks for a password"},
			{"-k", "asks for a password"},
			{"-K", "asks for a password"},
			{"--private-key", "reads a key"},
			{"--key-file", "reads a key"},
			{"-u", "sets a connection user"},
			{"--user", "sets a connection user"},
			{"-c", "sets a connection"},
			{"--connection", "sets a connection"},
			{"-M", "loads code from a path"},
			{"--module-path", "loads code from a path"},
		},
		arg: sensitiveArg,
	},
	// static analysis. It installs the roles and collections a project asks for
	// unless it is told it is offline, so offline is required; --fix rewrites the
	// files; a rules directory is code that it runs
	"ansible-lint": {
		needs: map[string]need{
			"": {anyOf: []string{"--offline", "--version", "--list-rules", "-L", "--list-tags", "-T"},
				why: "installs roles and collections from the network unless it is run with --offline"},
		},
		flags: []flag{
			{"--fix", "rewrites the files"},
			{"--write", "rewrites the files"},
			{"-r", "runs the code in a rules directory"},
			{"--rules-dir", "runs the code in a rules directory"},
			{"--rulesdir", "runs the code in a rules directory"},
		},
		arg: sensitiveArg,
	},
	"yamllint":    {arg: sensitiveArg},
	"ansible-doc": {arg: sensitiveArg},
}

// trivy config: static scan of IaC and manifests. It may fetch its checks from the
// network the first time and keeps them in its cache; nothing else is written
var trivyProgram = program{
	subs: set("config"),
	flags: []flag{
		{"--output", "writes a file"},
		{"-o", "writes a file"},
		{"--server", "sends the code to a server"},
		{"--token", "is for a server"},
		{"--custom-headers", "is for a server"},
	},
	arg: sensitiveArg,
}

// What terraform and tofu share. fmt must be told not to write; -chdir is the
// only way to point validate at a directory, so it is the one option allowed
// before the subcommand (and its value is judged like any argument).
var terraformProgram = program{
	subs:    set("fmt validate version"),
	leading: []string{"-chdir="},
	needs: map[string]need{
		"fmt": {anyOf: []string{"-check", "-write=false"}, why: "rewrites the files unless it is run with -check (or -write=false)"},
	},
	flags: []flag{
		{"-write=true", "rewrites the files"}, // not -write=false, which is what is asked for
	},
	arg: sensitiveArg,
}

func set(words string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

// How a remote target is written without a scheme: kustomize's github.com/org/repo//path, git@host:org/repo
var remoteShorthand = []string{"github.com/", "gitlab.com/", "bitbucket.org/", "git@", "git::"}

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
	w = strings.TrimPrefix(w, "@") // ansible -e @vars.yml, curl -d @file: a file to read
	if strings.Contains(w, "://") || strings.Contains(w, "?ref=") || startsWithAny(w, remoteShorthand) {
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
