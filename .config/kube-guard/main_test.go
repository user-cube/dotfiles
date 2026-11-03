package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func init() { color.NoColor = true }

func TestParse(t *testing.T) {
	plugins := func(w string) bool { return w == "who-can" }
	for _, tt := range []struct {
		tool, args string
		want       line
	}{
		{"kubectl", "get pods -A", line{sub: "get"}},
		{"kubectl", "-n api delete pod x", line{sub: "delete"}},
		// a flag's value before the subcommand is not taken for it
		{"kubectl", "--as admin delete pod x", line{sub: "delete"}},
		{"kubectl", "--request-timeout 5s delete pod x", line{sub: "delete"}},
		{"kubectl", "-v 6 delete pod x", line{sub: "delete"}},
		{"kubectl", "--context PRD-us -n a apply -f x.yaml", line{sub: "apply", context: "PRD-us"}},
		{"kubectl", "delete pod x --context=dev", line{sub: "delete", context: "dev"}},
		{"kubectl", "--kubeconfig prod.yaml delete pod x", line{sub: "delete", kubeconfig: "prod.yaml"}},
		{"kubectl", "--kubeconfig=prod.yaml delete pod x", line{sub: "delete", kubeconfig: "prod.yaml"}},
		{"kubectl", "rollout restart deploy/api", line{sub: "rollout:restart"}},
		{"kubectl", "rollout status deploy/api", line{sub: "rollout:status"}},
		{"kubectl", "who-can delete pods", line{sub: "who-can"}},
		{"kubectl", "exec pod -- kubectl --context prod delete x", line{sub: "exec"}}, // after -- is the pod's
		{"helm", "--kube-context dev upgrade api ./chart", line{sub: "upgrade", context: "dev"}},
		{"helm", "upgrade api ./chart --kube-context=prod-eu", line{sub: "upgrade", context: "prod-eu"}},
		{"helm", "--kube-as-user admin uninstall api", line{sub: "uninstall"}},
		{"helm", "diff upgrade api ./chart", line{sub: "diff"}},
	} {
		if got := parse(tt.tool, strings.Fields(tt.args), plugins); got != tt.want {
			t.Errorf("parse(%s %s) = %+v, want %+v", tt.tool, tt.args, got, tt.want)
		}
	}
}

func TestChanges(t *testing.T) {
	for tool, subs := range map[string]map[string]bool{
		"kubectl": {"delete": true, "apply": true, "rollout:restart": true, "get": false, "rollout:status": false, "diff": false, "who-can": false},
		"helm":    {"upgrade": true, "uninstall": true, "template": false, "diff": false},
	} {
		for sub, want := range subs {
			if got := (line{sub: sub}).changes(tool); got != want {
				t.Errorf("%s %s changes = %v, want %v", tool, sub, got, want)
			}
		}
	}
}

func writeConfig(t *testing.T, dir, name, ctx string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	body := "apiVersion: v1\nkind: Config\n"
	if ctx != "" {
		body += "current-context: " + ctx + "\n"
	}
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCurrentContext(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".kube"), 0o700)
	writeConfig(t, filepath.Join(home, ".kube"), "config", "default-ctx")
	a := writeConfig(t, home, "a.yaml", "")
	b := writeConfig(t, home, "b.yaml", "prod-b")
	c := writeConfig(t, home, "c.yaml", "dev-c")
	for _, tt := range []struct{ flag, env, want string }{
		{"", "", "default-ctx"},
		{c, b, "dev-c"},                       // --kubeconfig wins over KUBECONFIG
		{"", a + ":" + b + ":" + c, "prod-b"}, // the first file that sets one
		{"", "/missing.yaml:" + c, "dev-c"},   // a missing file is skipped
		{"~/b.yaml", "", "prod-b"},            // ~ is home
		{"/missing.yaml", "", ""},
	} {
		if got := currentContext(tt.flag, tt.env, home); got != tt.want {
			t.Errorf("currentContext(%q, %q) = %q, want %q", tt.flag, tt.env, got, tt.want)
		}
	}
}

// guarded sets up a home with ~/.kube/config on ctx
func guarded(t *testing.T, ctx string) {
	t.Helper()
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".kube"), 0o700)
	writeConfig(t, filepath.Join(home, ".kube"), "config", ctx)
	writeConfig(t, home, "prod.yaml", "prod-eu")
	writeConfig(t, home, "dev.yaml", "dev")
	t.Setenv("HOME", home)
	t.Setenv("KUBECONFIG", "")
	t.Setenv("KUBE_GUARD", "")
	t.Setenv("KUBE_PROD_REGEX", "")
	t.Chdir(home)
}

func TestCheck(t *testing.T) {
	for _, tt := range []struct {
		ctx, tool, args string
		asks, runs      bool // answering no
	}{
		{"dev-cluster", "kubectl", "delete pod x", false, true},
		{"prod-eu", "kubectl", "get pods -A", false, true},
		{"prod-eu", "kubectl", "-n api delete pod x", true, false},
		{"PROD-EU", "kubectl", "delete pod x", true, false}, // any case
		{"dev", "kubectl", "--context PRD-us -n a apply -f x.yaml", true, false},
		{"prod-eu", "kubectl", "--context dev delete pod x", false, true},
		{"prod-eu", "kubectl", "--as admin delete pod x", true, false},
		{"dev", "kubectl", "--kubeconfig prod.yaml delete pod x", true, false},
		{"prod-eu", "kubectl", "--kubeconfig dev.yaml delete pod x", false, true},
		{"prod-eu", "kubectl", "rollout restart deploy/api", true, false},
		{"prod-eu", "helm", "template ./chart", false, true},
		{"dev", "helm", "upgrade api ./chart --kube-context=prod-eu", true, false},
		{"prod-eu", "helm", "diff upgrade api ./chart", false, true},
	} {
		t.Run(tt.ctx+" "+tt.tool+" "+tt.args, func(t *testing.T) {
			guarded(t, tt.ctx)
			asked := false
			var out bytes.Buffer
			rc := check(tt.tool, strings.Fields(tt.args), func(string) bool { asked = true; return false }, &out)
			if asked != tt.asks || (rc == 0) != tt.runs {
				t.Errorf("asked %v, rc %d (%q)", asked, rc, out.String())
			}
		})
	}
}

func TestCheckYesRunsIt(t *testing.T) {
	guarded(t, "prod-eu")
	var label string
	rc := check("kubectl", []string{"delete", "pod", "x"}, func(l string) bool { label = l; return true }, &bytes.Buffer{})
	if rc != 0 || label != "kube-guard: prod-eu is production. kubectl delete pod x" {
		t.Errorf("rc %d, label %q", rc, label)
	}
}

func TestCheckKubeGuard0SkipsIt(t *testing.T) {
	guarded(t, "prod-eu")
	t.Setenv("KUBE_GUARD", "0")
	if rc := check("kubectl", []string{"delete", "pod", "x"}, func(string) bool { return false }, &bytes.Buffer{}); rc != 0 {
		t.Errorf("rc %d", rc)
	}
}

func TestKubeProdRegex(t *testing.T) {
	guarded(t, "live-eu")
	t.Setenv("KUBE_PROD_REGEX", "live")
	if rc := check("kubectl", []string{"delete", "pod", "x"}, func(string) bool { return false }, &bytes.Buffer{}); rc != 1 {
		t.Errorf("rc %d, want 1 (live-eu is production)", rc)
	}
	t.Setenv("KUBE_PROD_REGEX", "(")
	var out bytes.Buffer
	if rc := check("kubectl", []string{"delete", "pod", "x"}, func(string) bool { return true }, &out); rc != 2 || !strings.Contains(out.String(), "KUBE_PROD_REGEX") {
		t.Errorf("a bad regex: rc %d, %q", rc, out.String())
	}
}

func TestTUI(t *testing.T) {
	for _, tt := range []struct {
		ctx, args string
		readonly  bool
	}{
		{"prod-eu", "", true},
		{"prod-eu", "--write", false},
		{"prod-eu", "--readonly", false}, // already
		{"dev", "--context prod-us", true},
		{"dev", "--kubeconfig prod.yaml", true},
		{"dev", "", false},
	} {
		t.Run(tt.ctx+" "+tt.args, func(t *testing.T) {
			guarded(t, tt.ctx)
			var out bytes.Buffer
			rc := tui("k9s", strings.Fields(tt.args), &out)
			if (rc == 0) != tt.readonly {
				t.Errorf("rc %d (%q)", rc, out.String())
			}
			if tt.readonly && !strings.Contains(out.String(), "k9s read-only (k9s --write to change things)") {
				t.Errorf("not said: %q", out.String())
			}
		})
	}
}

func TestNamespace(t *testing.T) {
	home := t.TempDir()
	a := filepath.Join(home, "a.yaml")
	b := filepath.Join(home, "b.yaml")
	os.WriteFile(a, []byte("current-context: prod-eu\ncontexts:\n- name: dev\n  context:\n    namespace: dev-ns\n"), 0o600)
	os.WriteFile(b, []byte("contexts:\n- name: prod-eu\n  context:\n    cluster: c\n    namespace: api\n- name: dev\n  context:\n    namespace: other\n"), 0o600)
	env := a + ":" + b
	for ctx, want := range map[string]string{
		"prod-eu": "api",    // defined in the second file
		"dev":     "dev-ns", // the first file that defines it wins
		"nope":    "",
	} {
		if got := namespace(ctx, "", env, home); got != want {
			t.Errorf("namespace(%s) = %q, want %q", ctx, got, want)
		}
	}
}

func TestKubeContext(t *testing.T) {
	for _, tt := range []struct{ config, regex, want string }{
		{"current-context: prod-eu\ncontexts:\n- name: prod-eu\n  context:\n    namespace: api\n", "", "prod-eu\napi\nprod\n"},
		{"current-context: dev\n", "", "dev\n\nother\n"},
		{"current-context: live-1\n", "live", "live-1\n\nprod\n"}, // KUBE_PROD_REGEX
		{"apiVersion: v1\n", "", ""},                              // no context: nothing
	} {
		guarded(t, "")
		home, _ := os.UserHomeDir()
		os.WriteFile(filepath.Join(home, ".kube", "config"), []byte(tt.config), 0o600)
		t.Setenv("KUBE_PROD_REGEX", tt.regex)
		var out bytes.Buffer
		if rc := kubeContext(&out); rc != 0 || out.String() != tt.want {
			t.Errorf("%q: rc %d, got %q, want %q", tt.config, rc, out.String(), tt.want)
		}
	}
}
