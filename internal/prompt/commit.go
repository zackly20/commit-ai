// Package prompt membangun prompt untuk model dan menormalisasi/memvalidasi
// output model menjadi commit message yang valid sesuai Conventional Commits.
package prompt

import (
	"fmt"
	"regexp"
	"strings"
)

// AllowedTypes adalah daftar type Conventional Commits yang didukung (FR-05).
var AllowedTypes = []string{
	"feat", "fix", "docs", "style", "refactor",
	"perf", "test", "build", "ci", "chore", "revert",
}

// Options mengonfigurasi pembuatan prompt dan validasi output.
type Options struct {
	Language         string
	SubjectMaxLength int
}

// commitRe mencocokkan format `type(scope): subject`, `type: subject`,
// termasuk marker breaking change `!` sebelum titik dua.
var commitRe = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?!?: (.+)$`)

// SystemPrompt mengembalikan system prompt untuk model (bagian 9 PRD).
func SystemPrompt() string {
	return `You are an expert software developer.
Analyze the provided Git diff and generate one concise Git commit message.

Requirements:
- Follow Conventional Commits.
- Choose the type based on the actual change.
- Use a scope only when it is clear.
- Describe the primary user/developer-visible change.
- Do not invent changes not present in the diff.
- Do not include markdown fences.
- Return only the commit message.`
}

// ExplainSystemPrompt mengarahkan model menjelaskan perubahan dalam diff,
// bukan menulis ulang kode atau membuat commit message.
func ExplainSystemPrompt() string {
	return `You are an expert software developer reviewing Git changes.
Explain what the provided Git diff changes and why it matters.

Requirements:
- Summarize the primary change first.
- Mention the key files or components involved.
- Note anything risky or incomplete if visible.
- Do not invent changes not present in the diff.
- Do not include markdown fences.
- Keep it under 150 words.`
}

// BuildExplainUserPrompt membangun user prompt untuk explain:
// instruksi bahasa (opsional) + diff yang akan dijelaskan.
func BuildExplainUserPrompt(diff string, opts Options) string {
	var b strings.Builder
	if opts.Language != "" && opts.Language != "en" {
		fmt.Fprintf(&b, "Respond in language code %q.\n", opts.Language)
	}
	b.WriteString("\nGit diff:\n")
	b.WriteString(diff)
	return b.String()
}

// BuildUserPrompt membangun user prompt berisi instruksi bahasa + diff.
func BuildUserPrompt(diff string, opts Options) string {
	var b strings.Builder
	if opts.Language != "" && opts.Language != "en" {
		fmt.Fprintf(&b, "Write the commit message in language code %q.\n", opts.Language)
	}
	if opts.SubjectMaxLength > 0 {
		fmt.Fprintf(&b, "Keep the subject line under %d characters.\n", opts.SubjectMaxLength)
	}
	b.WriteString("\nGit diff:\n")
	b.WriteString(diff)
	return b.String()
}

// Normalize membersihkan output model: buang code fence, ambil baris
// message valid pertama, potong whitespace (FR-04 & bagian 9 PRD).
func Normalize(raw string, opts Options) string {
	s := strings.TrimSpace(raw)
	s = stripFences(s)

	var candidate string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Buang prefix retrorik seperti "Commit message:" dari model.
		cleaned := trimExplanationPrefix(line)
		if isValidFormat(cleaned) {
			candidate = cleaned
			break
		}
		if candidate == "" {
			candidate = cleaned
		}
	}
	if candidate == "" {
		candidate = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	}
	candidate = strings.Trim(candidate, "\"'` ")
	candidate = strings.TrimRight(candidate, ".")

	if opts.SubjectMaxLength > 0 && len(candidate) > opts.SubjectMaxLength {
		candidate = candidate[:opts.SubjectMaxLength]
		candidate = strings.TrimRight(candidate, " ,;:-")
	}
	return candidate
}

// stripFences membuang pembungkus ```...``` jika model mengembalikannya;
// isi di dalam fence tetap dipertahankan.
func stripFences(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// trimExplanationPrefix membuang label penjelasan yang umum dari model.
func trimExplanationPrefix(line string) string {
	lowered := strings.ToLower(line)
	for _, prefix := range []string{"commit message:", "message:", "commit:"} {
		if strings.HasPrefix(lowered, prefix) {
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return line
}

// isValidFormat memeriksa format Conventional Commits dasar.
func isValidFormat(s string) bool {
	m := commitRe.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	for _, t := range AllowedTypes {
		if strings.EqualFold(m[1], t) {
			return true
		}
	}
	return false
}

// NormalizeExplanation membersihkan output model untuk explain:
// buang code fence dan baris kosong berlebih. Penjelasan boleh multi-baris,
// jadi tidak dipangkas jadi satu baris seperti commit message.
func NormalizeExplanation(raw string) string {
	s := strings.TrimSpace(stripFences(raw))
	lines := strings.Split(s, "\n")
	var out []string
	blank := true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			if blank {
				continue // buang baris kosong di awal & beruntun
			}
			out = append(out, "")
			blank = true
			continue
		}
		out = append(out, strings.TrimRight(l, " \t"))
		blank = false
	}
	// Buang trailing blank lines.
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// Validate memvalidasi message final sebelum dipakai commit (bagian 9 PRD).
func Validate(message string, opts Options) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("commit message kosong")
	}
	m := commitRe.FindStringSubmatch(message)
	if m == nil {
		return fmt.Errorf("format tidak sesuai Conventional Commits (type(scope): description atau type: description)")
	}
	if !isValidFormat(message) {
		return fmt.Errorf("type %q tidak dikenal; gunakan salah satu dari: %s", strings.ToLower(m[1]), strings.Join(AllowedTypes, ", "))
	}
	if opts.SubjectMaxLength > 0 && len(message) > opts.SubjectMaxLength {
		return fmt.Errorf("subject melebihi %d karakter (%d)", opts.SubjectMaxLength, len(message))
	}
	return nil
}
