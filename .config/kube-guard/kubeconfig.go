package main

import (
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// currentContext is the current-context kubectl would use: of --kubeconfig
// when given, else of the KUBECONFIG files (the first that sets one wins, as
// kubectl merges them), else of ~/.kube/config. "" when there is none. Read
// from the files rather than asking kubectl, which takes ~30 ms
func currentContext(flag, env, home string) string {
	var files []string
	switch {
	case flag != "":
		files = []string{flag}
	case env != "":
		files = filepath.SplitList(env)
	default:
		files = []string{filepath.Join(home, ".kube", "config")}
	}
	for _, f := range files {
		if f == "" {
			continue
		}
		if strings.HasPrefix(f, "~/") {
			f = filepath.Join(home, f[2:])
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue // kubectl skips a missing file too
		}
		var kc struct {
			CurrentContext string `yaml:"current-context"`
		}
		if yaml.Unmarshal(data, &kc) == nil && kc.CurrentContext != "" {
			return kc.CurrentContext
		}
	}
	return ""
}
