package main

import (
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// kubeconfig is what kube-guard reads of a kubeconfig file
type kubeconfig struct {
	CurrentContext string `yaml:"current-context"`
	Contexts       []struct {
		Name    string `yaml:"name"`
		Context struct {
			Namespace string `yaml:"namespace"`
		} `yaml:"context"`
	} `yaml:"contexts"`
}

// kubeconfigs reads the files kubectl would: --kubeconfig when given, else the
// KUBECONFIG files in order, else ~/.kube/config. A missing or broken file is
// skipped, as kubectl skips a missing one. Read from the files rather than
// asking kubectl, which takes ~30 ms
func kubeconfigs(flag, env, home string) []kubeconfig {
	var files []string
	switch {
	case flag != "":
		files = []string{flag}
	case env != "":
		files = filepath.SplitList(env)
	default:
		files = []string{filepath.Join(home, ".kube", "config")}
	}
	var kcs []kubeconfig
	for _, f := range files {
		if f == "" {
			continue
		}
		if strings.HasPrefix(f, "~/") {
			f = filepath.Join(home, f[2:])
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var kc kubeconfig
		if yaml.Unmarshal(data, &kc) == nil {
			kcs = append(kcs, kc)
		}
	}
	return kcs
}

// currentContext is the current-context kubectl would use: the first file that
// sets one wins, as kubectl merges them. "" when there is none
func currentContext(flag, env, home string) string {
	for _, kc := range kubeconfigs(flag, env, home) {
		if kc.CurrentContext != "" {
			return kc.CurrentContext
		}
	}
	return ""
}

// namespace is the namespace of the context ctx: the first file that defines
// that context wins. "" when it sets none (kubectl's "default")
func namespace(ctx, flag, env, home string) string {
	for _, kc := range kubeconfigs(flag, env, home) {
		for _, c := range kc.Contexts {
			if c.Name == ctx {
				return c.Context.Namespace
			}
		}
	}
	return ""
}
