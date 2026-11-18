package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestHookCheck(t *testing.T) {
	for _, tt := range []struct {
		name, ctx, cmd string
		ask            bool
	}{
		// allowed: reads, or not production
		{"read on prod", "prod-eu", `kubectl get pods -A`, false},
		{"logs on prod", "prod-eu", `kubectl logs deploy/api -f`, false},
		{"helm template on prod", "prod-eu", `helm template x ./chart`, false},
		{"delete on dev", "dev", `kubectl delete pod x`, false},
		{"not kubectl", "prod-eu", `echo "kubectl delete pod x"`, false},
		{"empty", "prod-eu", ``, false},

		// asks: a change on production
		{"delete on prod", "prod-eu", `kubectl delete pod x`, true},
		{"apply on prod", "prod-eu", `kubectl apply -f x.yaml`, true},
		{"helm upgrade on prod", "prod-eu", `helm upgrade x ./chart`, true},
		{"rollout restart", "prod-eu", `kubectl rollout restart deploy/api`, true},
		{"rollout status is a read", "prod-eu", `kubectl rollout status deploy/api`, false},
		{"exec on prod", "prod-eu", `kubectl exec -it pod -- sh`, true},
		{"kubecolor", "prod-eu", `kubecolor delete pod x`, true},
		{"--context to prod", "dev", `kubectl --context prod-eu delete pod x`, true},
		{"--context to dev", "prod-eu", `kubectl --context dev delete pod x`, false},
		{"--kubeconfig to prod", "dev", `kubectl --kubeconfig prod.yaml delete pod x`, true},

		// hidden in the shell
		{"after &&", "prod-eu", `cd x && kubectl delete pod y`, true},
		{"pipe", "prod-eu", `kubectl get pods -o name | xargs kubectl delete`, true},
		{"rtk wrapper", "prod-eu", `rtk kubectl delete pod x`, true},
		{"env prefix", "prod-eu", `FOO=1 sudo -E kubectl delete pod x`, true},
		{"sh -c", "prod-eu", `sh -c "kubectl delete pod x"`, true},
		{"bash -lc", "prod-eu", `bash -lc 'kubectl apply -f x.yaml'`, true},
		{"eval", "prod-eu", `eval "kubectl delete pod x"`, true},
		{"substitution", "prod-eu", `echo $(kubectl delete pod x)`, true},
		{"backticks", "prod-eu", "echo `kubectl delete pod x`", true},
		{"full path", "prod-eu", `/usr/local/bin/kubectl delete pod x`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			guarded(t, tt.ctx)
			got := hookCheck(tt.cmd, 0)
			if (got != "") != tt.ask {
				t.Errorf("hookCheck(%q) = %q, ask want %v", tt.cmd, got, tt.ask)
			}
		})
	}
}

func TestHookIgnoresKubeGuard0(t *testing.T) {
	guarded(t, "prod-eu")
	t.Setenv("KUBE_GUARD", "0") // the user's switch, not Claude's
	if hookCheck(`kubectl delete pod x`, 0) == "" {
		t.Error("KUBE_GUARD=0 must not switch the hook off")
	}
}

func TestHookBadRegexAsks(t *testing.T) {
	guarded(t, "prod-eu")
	t.Setenv("KUBE_PROD_REGEX", "(")
	if hookCheck(`kubectl delete pod x`, 0) == "" {
		t.Error("an invalid production regex must ask, not allow")
	}
}

func TestClaudeHookOutput(t *testing.T) {
	guarded(t, "prod-eu")
	event := func(cmd string) *bytes.Reader {
		b, _ := json.Marshal(map[string]any{"tool_input": map[string]string{"command": cmd}})
		return bytes.NewReader(b)
	}

	var out bytes.Buffer
	if rc := claudeHook(event(`kubectl delete pod x`), &out); rc != 0 {
		t.Errorf("exit %d, want 0", rc)
	}
	var resp struct {
		Out map[string]string `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("not JSON: %q", out.String())
	}
	if resp.Out["permissionDecision"] != "ask" || !strings.Contains(resp.Out["permissionDecisionReason"], "prod-eu") {
		t.Errorf("decision = %v", resp.Out)
	}

	out.Reset()
	claudeHook(event(`kubectl get pods`), &out)
	if out.Len() != 0 {
		t.Errorf("a read printed %q, want nothing", out.String())
	}

	out.Reset()
	claudeHook(strings.NewReader("not json"), &out)
	if out.Len() != 0 {
		t.Errorf("bad input printed %q, want nothing", out.String())
	}
}
