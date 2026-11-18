package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type tc struct {
	name, cmd string
	deny      string // a part of the reason; "" = allowed
}

func table(t *testing.T, profileNames []string, cases []tc) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := check(tt.cmd, profileNames)
			switch {
			case tt.deny == "" && got != "":
				t.Errorf("%q denied: %s", tt.cmd, got)
			case tt.deny != "" && !strings.Contains(got, tt.deny):
				t.Errorf("%q: got %q, want a reason with %q", tt.cmd, got, tt.deny)
			}
		})
	}
}

func TestGit(t *testing.T) {
	table(t, []string{"git"}, []tc{
		// allowed: git that only reads
		{"diff", `git diff`, ""},
		{"diff against a base", `git diff main...HEAD`, ""},
		{"diff stat", `git diff --stat origin/main...HEAD`, ""},
		{"diff of a path", `git diff HEAD -- claude/settings.json`, ""},
		{"log", `git log --oneline -5`, ""},
		{"log with a format", `git log --format=%h -3`, ""},
		{"show", `git show HEAD~1`, ""},
		{"status", `git status --short`, ""},
		{"ls-files", `git ls-files claude`, ""},
		{"blame", `git blame -L 1,20 main.go`, ""},
		{"merge-base", `git merge-base main HEAD`, ""},
		{"rev-parse", `git rev-parse --abbrev-ref HEAD`, ""},
		{"rtk in front", `rtk git diff`, ""},
		{"a flag after -- is a path", `git diff -- --output`, ""},
		{"empty", ``, ""},

		// denied: not a read
		{"commit", `git commit -m x`, "is not one of the git commands"},
		{"push", `git push`, "is not one of the git commands"},
		{"reset", `git reset --hard`, "is not one of the git commands"},
		{"checkout", `git checkout x`, "is not one of the git commands"},
		{"config", `git config user.name x`, "is not one of the git commands"},
		{"clean", `git clean -fd`, "is not one of the git commands"},
		{"stash", `git stash`, "is not one of the git commands"},
		{"other programs", `cat README.md`, "may only run: git"},
		{"rm", `rm -rf x`, "may only run: git"},
		{"gh", `gh pr diff 3`, "may only run: git"},
		{"helm is not in this profile", `helm template x .`, "may only run: git"},
		{"git alone", `git`, "give it one of"},
		{"env prefix", `FOO=1 git diff`, "may only run: git"},
		{"absolute path to git", `/usr/bin/git diff`, "may only run: git"},

		// denied: an option that runs a program or moves the repository
		{"-c", `git -c diff.external=./x diff`, "No option before"},
		{"-C", `git -C /other diff`, "No option before"},
		{"--git-dir", `git --git-dir=/x diff`, "No option before"},
		{"--no-pager is still an option before", `git --no-pager diff`, "No option before"},

		// denied: a flag that writes or runs
		{"--output=", `git diff --output=out.patch`, "writes a file"},
		{"--output file", `git log --output out.txt`, "writes a file"},
		{"--ext-diff", `git diff --ext-diff`, "runs a program"},
		{"--textconv", `git show --textconv HEAD:x`, "runs a program"},
		{"-O", `git diff -Otool`, "runs a program"},
		{"--open-files-in-pager", `git grep --open-files-in-pager`, "is not one of the git commands"},
		{"--output on show", `git show --output=x HEAD`, "writes a file"},

		// denied: shell syntax
		{"semicolon", `git diff; rm x`, "one simple command"},
		{"and", `git diff && git push`, "one simple command"},
		{"pipe", `git diff | head`, "one simple command"},
		{"redirect", `git diff > out.patch`, "one simple command"},
		{"redirect in", `git diff < x`, "one simple command"},
		{"substitution", `git diff $(rm x)`, "one simple command"},
		{"backticks", "git diff `rm x`", "one simple command"},
		{"newline", "git diff\nrm x", "one simple command"},
		{"background", `git diff &`, "one simple command"},
	})
}

func TestHelm(t *testing.T) {
	table(t, []string{"helm"}, []tc{
		// allowed: render and lint what is on this machine
		{"lint", `helm lint ./chart`, ""},
		{"lint with values", `helm lint ./chart -f values/prod.yaml`, ""},
		{"lint strict", `helm lint ./chart --strict`, ""},
		{"template", `helm template my-release ./chart`, ""},
		{"template with values", `helm template r ./chart -f ./chart/values.yaml -f values/prod.yaml`, ""},
		{"template with --values=", `helm template r ./chart --values=values/prod.yaml`, ""},
		{"template one file", `helm template r ./chart --show-only templates/deployment.yaml`, ""},
		{"template with --set", `helm template r ./chart --set image.tag=1.2.3 --set replicas=2`, ""},
		{"template with a namespace", `helm template r ./chart -n api --include-crds`, ""},
		{"template debug", `helm template r ./chart --debug`, ""},
		{"encrypted values under secrets", `helm template r ./chart -f secrets/prod.enc.yaml`, ""},
		{"an env template", `helm template r ./chart -f .env.example`, ""},
		{"version", `helm version`, ""},
		{"dependency list", `helm dependency list ./chart`, ""},
		{"rtk in front", `rtk helm lint ./chart`, ""},

		// denied: asks the cluster, or reaches outside
		{"--validate", `helm template r ./chart --validate`, "asks the cluster"},
		{"--dry-run", `helm template r ./chart --dry-run`, "asks the cluster"},
		{"--dry-run=server", `helm template r ./chart --dry-run=server`, "asks the cluster"},
		{"--kube-context after", `helm template r ./chart --kube-context prod`, "points at a cluster"},
		{"--kubeconfig after", `helm template r ./chart --kubeconfig=/x`, "points at a cluster"},
		{"--kube-context before", `helm --kube-context prod template r ./chart`, "No option before"},
		{"--output-dir", `helm template r ./chart --output-dir /tmp/out`, "writes files"},
		{"--post-renderer", `helm template r ./chart --post-renderer ./x`, "runs a program"},
		{"--set-file", `helm template r ./chart --set-file a=/x`, "reads a file"},
		{"--repo", `helm template r nginx --repo https://charts.example.com`, "fetches from the network"},
		{"a remote chart", `helm template r oci://registry.example.com/chart`, "a remote chart or URL"},
		{"a chart URL", `helm lint https://example.com/chart.tgz`, "a remote chart or URL"},
		{"--key-file", `helm template r ./chart --key-file /x`, "reads a key"},

		// denied: it changes something
		{"install", `helm install r ./chart`, "is not one of the helm commands"},
		{"upgrade", `helm upgrade r ./chart`, "is not one of the helm commands"},
		{"uninstall", `helm uninstall r`, "is not one of the helm commands"},
		{"rollback", `helm rollback r 1`, "is not one of the helm commands"},
		{"repo update", `helm repo update`, "is not one of the helm commands"},
		{"pull", `helm pull nginx`, "is not one of the helm commands"},
		{"package", `helm package ./chart`, "is not one of the helm commands"},
		{"plugin", `helm plugin install x`, "is not one of the helm commands"},
		{"get values asks the cluster", `helm get values r`, "is not one of the helm commands"},
		{"dependency build", `helm dependency build ./chart`, "may only be followed by list"},
		{"dependency update", `helm dependency update ./chart`, "may only be followed by list"},
		{"dependency alone", `helm dependency`, "may only be followed by list"},
		{"helm alone", `helm`, "give it one of"},

		// denied: a path to a secret that helm would read by itself
		{"values under secrets, cleartext", `helm template r ./chart -f secrets/prod.yaml`, "not marked as encrypted"},
		{"values under credentials", `helm template r ./chart --values=credentials/db.yaml`, "not marked as encrypted"},
		{"an env file", `helm template r ./chart -f .env`, "an env file"},
		{"an env file local", `helm template r ./chart -f app/.env.local`, "an env file"},
		{"a key", `helm template r ./chart -f certs/tls.key`, "a key or certificate file"},
		{"a pem", `helm template r ./chart -f tls.pem`, "a key or certificate file"},
		{"an age key", `helm template r ./chart -f age-key.txt`, "an age key"},
		{"ssh", `helm template r ./chart -f ~/.ssh/config`, "an ssh path"},
		{"aws", `helm template r ./chart -f ~/.aws/credentials`, "an AWS path"},

		// denied: shell syntax
		{"pipe", `helm template r ./chart | head`, "one simple command"},
		{"redirect", `helm template r ./chart > out.yaml`, "one simple command"},
		{"semicolon", `helm lint ./chart; rm x`, "one simple command"},
		{"substitution", `helm lint $(cat x)`, "one simple command"},

		// other programs
		{"kubectl", `kubectl get pods`, "may only run: helm"},
		{"git is not in this profile", `git diff`, "may only run: helm"},
	})
}

func TestProfilesAddUp(t *testing.T) {
	table(t, []string{"git", "helm"}, []tc{
		{"git", `git diff main...HEAD`, ""},
		{"helm", `helm lint ./chart`, ""},
		{"a git write is still denied", `git commit -m x`, "is not one of the git commands"},
		{"a helm install is still denied", `helm install r ./chart`, "is not one of the helm commands"},
		{"kubectl is in neither", `kubectl get pods`, "may only run: git, helm"},
	})
}

func TestUnknownProfileDeniesEverything(t *testing.T) {
	for _, names := range [][]string{nil, {}, {"gti"}, {"git", "hlem"}} {
		got := check("git diff", names)
		if !strings.Contains(got, "denies everything") {
			t.Errorf("profiles %v: got %q, want everything denied", names, got)
		}
	}
	if got := check("", []string{"gti"}); got != "" {
		t.Errorf("an empty command is not a command: %q", got)
	}
}

func TestRun(t *testing.T) {
	event := func(cmd string) *bytes.Reader {
		b, _ := json.Marshal(map[string]any{"tool_input": map[string]string{"command": cmd}})
		return bytes.NewReader(b)
	}

	var out bytes.Buffer
	run(event("git commit -m x"), &out, []string{"git"})
	var resp struct {
		Out map[string]string `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("not JSON: %q", out.String())
	}
	if resp.Out["permissionDecision"] != "deny" || !strings.Contains(resp.Out["permissionDecisionReason"], "readonly") {
		t.Errorf("decision = %v", resp.Out)
	}

	out.Reset()
	run(event("git diff main...HEAD"), &out, []string{"git"})
	if out.Len() != 0 {
		t.Errorf("a read printed %q, want nothing", out.String())
	}

	out.Reset()
	run(strings.NewReader("not json"), &out, []string{"git"})
	if out.Len() != 0 {
		t.Errorf("bad input printed %q, want nothing", out.String())
	}
}
