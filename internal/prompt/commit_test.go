package prompt

import (
	"strings"
	"testing"
)

func TestNormalizeValid(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"plain", "feat(auth): add password reset", "feat(auth): add password reset"},
		{"with fence", "```text\nfix(api): handle nil pointer\n```", "fix(api): handle nil pointer"},
		{"with explanation", "Here is the commit message:\nfeat: add login flow", "feat: add login flow"},
		{"with quotes", "\"chore: bump deps\"", "chore: bump deps"},
		{"trailing period", "fix: fix bug.", "fix: fix bug"},
		{"extra prose first line", "Sure!\nfeat(core): add caching", "feat(core): add caching"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.raw, Options{SubjectMaxLength: 72})
			if got != tt.want {
				t.Fatalf("Normalize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTruncatesLongSubject(t *testing.T) {
	raw := "feat: " + strings.Repeat("a", 100)
	got := Normalize(raw, Options{SubjectMaxLength: 72})
	if len(got) != 72 {
		t.Fatalf("len = %d, want 72", len(got))
	}
}

func TestValidate(t *testing.T) {
	opts := Options{SubjectMaxLength: 72}

	valid := []string{
		"feat: add x",
		"feat(auth): add y",
		"fix(api)!: breaking change",
	}
	for _, m := range valid {
		if err := Validate(m, opts); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", m, err)
		}
	}

	invalid := []string{
		"",
		"update stuff",          // tanpa type
		"feature: add x",        // type tidak dikenal
		"feat: ",                // subject kosong — regex menolak karena butuh "type: " + isi
	}
	for _, m := range invalid {
		if err := Validate(m, opts); err == nil {
			t.Errorf("Validate(%q) = nil, want error", m)
		}
	}
}

func TestValidateSubjectTooLong(t *testing.T) {
	opts := Options{SubjectMaxLength: 20}
	msg := "feat: " + strings.Repeat("x", 30)
	if err := Validate(msg, opts); err == nil {
		t.Fatal("want error for long subject, got nil")
	}
}

func TestBuildUserPrompt(t *testing.T) {
	opts := Options{Language: "id", SubjectMaxLength: 72}
	got := BuildUserPrompt("diff --git a/x b/x", opts)
	if !strings.Contains(got, `language code "id"`) {
		t.Errorf("user prompt should mention language, got: %s", got)
	}
	if !strings.Contains(got, "Git diff:") {
		t.Errorf("user prompt should contain diff header")
	}
}

func TestSystemPromptMentionsConventional(t *testing.T) {
	if !strings.Contains(SystemPrompt(), "Conventional Commits") {
		t.Error("system prompt should mention Conventional Commits")
	}
}
