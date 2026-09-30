# CommitAI

[![Build Status](https://github.com/zackly20/commit-ai/actions/workflows/ci.yml/badge.svg)](https://github.com/zackly20/commit-ai/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/zackly20/commit-ai)](https://github.com/zackly20/commit-ai/releases)
[![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

CLI commit message generator berbasis **Local AI** (Ollama). CommitAI menganalisis staged Git diff dan menghasilkan commit message [Conventional Commits](https://www.conventionalcommits.org) yang ringkas dan konsisten — **tanpa mengirim source code ke layanan AI cloud** secara default.

```
git add .
commit-ai
✓ Checking repository...
✓ Reading staged changes... (2 file)
✓ Generating commit message...

Generated commit:
  feat(auth): add password reset functionality

? What do you want to do?
  1) Commit
  2) Edit
  3) Regenerate
  4) Cancel
```

## Requirements

- **Git** — dipasang dan tersedia di PATH
- **Ollama** — [unduh di sini](https://ollama.com/download), lalu ambil model:

```bash
ollama pull qwen2.5-coder
```

## Installation

### Build dari source

```bash
git clone https://github.com/zackly20/commit-ai.git
cd commit-ai
go build -o commit-ai .
```

Binary hasil build bisa dipindahkan ke folder PATH kamu.

### Release binary

Binary cross-platform tersedia di halaman [Releases](https://github.com/zackly20/commit-ai/releases) — build otomatis oleh CI saat tag `v*` di-push.

| OS | File |
|---|---|
| Windows (amd64/arm64) | `commit-ai-windows-amd64.zip` / `commit-ai-windows-arm64.zip` |
| macOS (Intel/Apple Silicon) | `commit-ai-darwin-amd64.tar.gz` / `commit-ai-darwin-arm64.tar.gz` |
| Linux (amd64/arm64) | `commit-ai-linux-amd64.tar.gz` / `commit-ai-linux-arm64.tar.gz` |

Verifikasi integritas unduhan dengan `checksums.txt` (SHA-256) yang disertakan di setiap rilis:

```bash
sha256sum -c checksums.txt --ignore-missing
```

## Usage

| Command | Fungsi |
|---|---|
| `commit-ai` | Workflow utama: analyze → generate → review → commit |
| `commit-ai generate` | Generate message saja, tanpa commit |
| `commit-ai init` | Buat `.commit-ai.yaml` di repository |
| `commit-ai config show` | Tampilkan konfigurasi efektif |
| `commit-ai config get <key>` | Tampilkan satu nilai |
| `commit-ai config set <key> <value>` | Ubah & simpan konfigurasi |
| `commit-ai config reset` | Kembalikan project config ke default |
| `commit-ai --version` | Versi |

Flag opsional: `--endpoint`, `--model`, `--lang`, `--yes`.

## Configuration

Konfigurasi dibaca dengan precedence: **CLI flags > `.commit-ai.yaml` (project) > global config > default**.

Global config: `~/.config/commit-ai/config.yaml` (Windows: `%AppData%\commit-ai\config.yaml`).

Contoh `.commit-ai.yaml` (dibuat oleh `commit-ai init`):

```yaml
provider: ollama
endpoint: http://localhost:11434
model: qwen2.5-coder
format: conventional
language: en
max_diff_chars: 20000
subject_max_length: 72
confirm_before_commit: true
```

## Privacy

- Source code **tidak dikirim ke cloud** — inference berjalan penuh di komputer kamu via Ollama.
- Tidak ada telemetry dan tidak ada API key cloud yang dibutuhkan.
- Hati-hati: diff bisa mengandung secret yang tidak sengaja ter-stage (API key, password). Jangan stage file secret, gunakan `.gitignore`.

## Troubleshooting

| Masalah | Solusi |
|---|---|
| `Bukan Git repository` | Jalankan dari dalam folder repository Git |
| `Tidak ada staged changes` | Jalankan `git add` dulu, contoh `git add .` |
| `Git tidak ditemukan` | Install Git dari [git-scm.com](https://git-scm.com) |
| `ollama tidak dapat dijangkau` | Jalankan `ollama serve` (atau buka app Ollama) |
| `model belum tersedia` | Jalankan `ollama pull <model>` sesuai nama model di error |
| Commit gagal | Staged changes tetap utuh; perbaiki penyebabnya lalu coba lagi |

## Development

```bash
go test ./...      # unit tests
go vet ./...       # static analysis
go build -o commit-ai .
```

### Testing tanpa LLM (laptop ber-speks rendah)

Pipeline penuh bisa diuji tanpa menjalankan model AI: server tiruan Ollama
disertakan di `tools/mock-ollama`.

```bash
# Terminal 1: mock server (port 11434, seperti Ollama asli)
go run ./tools/mock-ollama

# Terminal 2: jalankan CommitAI seperti biasa
commit-ai          # akan menghasilkan: feat(mock): commit message from mock server
```

Mock ini juga menerima `--message "..."` untuk menguji output tertentu.
Integration test otomatis (`go test ./...`) juga memakai mock HTTP,
sehingga CI tidak membutuhkan Ollama sama sekali.

## Roadmap

- **v0.1.0 (MVP)** — core CLI + Ollama + Git: generate/review/commit
- **Phase 2 (DX)** — `review`, `explain`, `changelog`
- **Phase 3** — IDE integration (VS Code / JetBrains / Neovim)
- **Phase 4** — shared team config & conventions
- **Phase 5** — cloud provider sebagai opsi eksplisit (opt-in)

## Contributing

Kontribusi sangat diterima! Lihat [CONTRIBUTING.md](CONTRIBUTING.md) untuk panduan setup development, konvensi kode, dan alur Pull Request.

## License

MIT — lihat [LICENSE](LICENSE).
