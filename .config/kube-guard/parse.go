package main

import "strings"

// Every subcommand of kubectl and helm (and helm diff, a read): the subcommand
// is the first word that is one, so the value of a flag before it (--as admin,
// --request-timeout 5s, -v 6) isn't taken for it
var commands = map[string]map[string]bool{
	"kubectl": set("alpha annotate api-resources api-versions apply attach auth autoscale certificate cluster-info completion config cordon cp create debug delete describe diff drain edit events exec explain expose get kustomize label logs options patch plugin port-forward proxy replace rollout run scale set taint top uncordon version wait"),
	"helm":    set("completion create del delete dependency diff env get help history install lint list ls package plugin pull push registry repo rollback search show status template test un uninstall upgrade verify version"),
}

// The subcommands that change the cluster
var mutating = map[string]map[string]bool{
	"kubectl": set("apply create delete edit patch replace scale set label annotate taint drain cordon uncordon autoscale expose run exec debug cp" +
		" rollout:restart rollout:undo rollout:pause rollout:resume"),
	"helm": set("install upgrade uninstall un delete del rollback"),
}

// Flags whose value is the next word, when not given as --flag=value
var valueFlags = set("-n --namespace --cluster --user -s --server -f --filename -l --selector -o --output -c --container")

func set(words string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(words) {
		m[w] = true
	}
	return m
}

// line is what kube-guard needs of a command line
type line struct {
	sub        string // subcommand; "rollout:restart" for rollout's
	context    string // --context / --kube-context
	kubeconfig string // --kubeconfig
}

// parse reads tool's arguments. isPlugin says whether a word is a kubectl
// plugin (kubectl-<name> on the PATH, e.g. who-can): a subcommand too, whose
// arguments (who-can delete pods) are not kubectl's
func parse(tool string, args []string, isPlugin func(string) bool) line {
	var l line
	for i := 0; i < len(args); i++ {
		arg := args[i]
		next := func() string {
			if i+1 < len(args) {
				i++
				return args[i]
			}
			return ""
		}
		switch {
		case arg == "--":
			return l
		case strings.HasPrefix(arg, "--context="), strings.HasPrefix(arg, "--kube-context="):
			l.context = arg[strings.IndexByte(arg, '=')+1:]
		case arg == "--context", arg == "--kube-context":
			l.context = next()
		case strings.HasPrefix(arg, "--kubeconfig="):
			l.kubeconfig = arg[len("--kubeconfig="):]
		case arg == "--kubeconfig":
			l.kubeconfig = next()
		case valueFlags[arg]:
			next()
		case strings.HasPrefix(arg, "-"):
		case l.sub == "":
			if commands[tool][arg] || (tool == "kubectl" && isPlugin != nil && isPlugin(arg)) {
				l.sub = arg
			}
		case l.sub == "rollout":
			l.sub += ":" + arg
		}
	}
	return l
}

// changes says whether the line changes the cluster
func (l line) changes(tool string) bool { return mutating[tool][l.sub] }
