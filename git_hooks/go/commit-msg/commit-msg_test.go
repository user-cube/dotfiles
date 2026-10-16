package main

import "testing"

func TestIssueKey(t *testing.T) {
	cases := map[string]string{
		"features/PROJECT-123-foo": "PROJECT-123",
		"feature/proj-7":           "PROJ-7",
		"PROJ-42-thing":            "PROJ-42",
		"chore/cleanup":            "cleanup", // no key: last segment
	}
	for branch, want := range cases {
		if got := issueKey(branch); got != want {
			t.Errorf("issueKey(%q) = %q, want %q", branch, got, want)
		}
	}
}

func TestFormatHeader(t *testing.T) {
	cases := []struct{ header, action, want string }{
		// Prefixes
		{"fix something", "feat", "fix: PROJ-5 something"},
		{"fix: something", "feat", "fix: PROJ-5 something"},
		{"Fix: something", "feat", "Fix: PROJ-5 something"},
		{"fix(api): something", "feat", "fix(api): PROJ-5 something"},
		{"feat!: breaking", "feat", "feat!: PROJ-5 breaking"},
		{"something", "feat", "feat: PROJ-5 something"},
		{"something", "chore", "chore: PROJ-5 something"},
		{"fixup the thing", "feat", "feat: PROJ-5 fixup the thing"}, // not "fix"
		// Modes
		{"no-prefix: something", "feat", "PROJ-5 something"},
		{"no-track: something", "feat", "something"},
		{"something", "disabled", "something"},
		// Already tagged (amend, reword): unchanged
		{"fix: PROJ-5 something", "feat", "fix: PROJ-5 something"},
		{"fix(api): proj-5 something", "feat", "fix(api): proj-5 something"},
		{"PROJ-5 something", "feat", "PROJ-5 something"},
		{"no-prefix: PROJ-5 something", "feat", "PROJ-5 something"},
		// Messages git or autosquash rely on
		{"fixup! fix: PROJ-5 something", "feat", "fixup! fix: PROJ-5 something"},
		{"squash! fix: PROJ-5 something", "feat", "squash! fix: PROJ-5 something"},
		{"amend! fix: PROJ-5 something", "feat", "amend! fix: PROJ-5 something"},
		{"Merge branch 'main' into feature/PROJ-5", "feat", "Merge branch 'main' into feature/PROJ-5"},
		{`Revert "fix: PROJ-5 something"`, "feat", `Revert "fix: PROJ-5 something"`},
	}
	for _, c := range cases {
		if got := formatHeader(c.header, "PROJ-5", c.action); got != c.want {
			t.Errorf("formatHeader(%q, %q) = %q, want %q", c.header, c.action, got, c.want)
		}
	}
}
