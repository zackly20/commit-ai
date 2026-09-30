package prompt

import (
	"strings"
	"testing"
)

func TestExplainSystemPromptContent(t *testing.T) {
	sp := ExplainSystemPrompt()
	if !strings.Contains(sp, "Explain what the provided Git diff changes") {
		t.Error("system prompt harus meminta penjelasan diff")
	}
	if !strings.Contains(sp, "Do not invent changes not present in the diff") {
		t.Error("system prompt harus melarang mengarang perubahan")
	}
	if strings.Contains(sp, "commit message") {
		t.Error("explain tidak boleh diminta membuat commit message")
	}
}

func TestBuildExplainUserPrompt(t *testing.T) {
	// Bahasa selain en harus disebut.
	got := BuildExplainUserPrompt("DIFF", Options{Language: "id"})
	if !strings.Contains(got, `Respond in language code "id"`) {
		t.Errorf("harus menyebut bahasa, dapat: %s", got)
	}
	if !strings.HasSuffix(got, "DIFF") {
		t.Errorf("diff harus di akhir prompt, dapat: %s", got)
	}

	// Bahasa en (default) tidak menambah instruksi bahasa.
	got = BuildExplainUserPrompt("DIFF", Options{Language: "en"})
	if strings.Contains(got, "Respond in language") {
		t.Error("en tidak perlu instruksi bahasa")
	}
	if !strings.Contains(got, "Git diff:\nDIFF") {
		t.Errorf("format header+diff salah: %q", got)
	}
}

func TestNormalizeExplanation(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			"plain",
			"A.\nB.",
			"A.\nB.",
		},
		{
			"buang fence",
			"```text\nA.\nB.\n```",
			"A.\nB.",
		},
		{
			"rapikan blank lines",
			"\n\nA.\n\n\n\nB.\n\n",
			"A.\n\nB.",
		},
		{
			"trim trailing spaces",
			"A.   \nB.\t",
			"A.\nB.",
		},
		{
			"indentasi dipertahankan",
			"A.\n  - poin satu\n  - poin dua",
			"A.\n  - poin satu\n  - poin dua",
		},
		{
			"kosong",
			"   \n\n  ",
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NormalizeExplanation(c.raw); got != c.want {
				t.Errorf("NormalizeExplanation() = %q, want %q", got, c.want)
			}
		})
	}
}
