# Panduan Berkontribusi

Terima kasih sudah mau berkontribusi ke **CommitAI**! 🎉 Dokumen ini menjelaskan cara melaporkan masalah, mengusulkan fitur, dan mengirim kode.

## Cara Berkontribusi

- 🐛 **Lapor bug** — buka [Issue](https://github.com/zackly20/commit-ai/issues) dengan template Bug Report
- 💡 **Usulkan fitur** — buka Issue dengan template Feature Request
- 💻 **Kirim kode** — fork → branch → commit → Pull Request
- 📖 **Perbaiki dokumentasi** — README, komentar kode, contoh konfigurasi

## Setup Development

### Requirements

| Tool | Keterangan |
|---|---|
| Go 1.27+ | Wajib untuk build & test |
| Git | Wajib (tool ini membungkus Git CLI) |
| Ollama | **Opsional** — tidak diperlukan untuk development/test |

### Clone & build

```bash
git clone https://github.com/zackly20/commit-ai.git
cd commit-ai
go build -o commit-ai .
```

### Testing tanpa LLM

Semua test memakai mock HTTP — laptop ber-speks rendah pun bisa menjalankan seluruh test suite:

```bash
go test ./...   # unit + integration test (tanpa Ollama)
go vet ./...    # static analysis
```

Untuk mencoba CLI secara manual tanpa model sungguhan, pakai mock server:

```bash
# Terminal 1
go run ./tools/mock-ollama

# Terminal 2 — di repository Git apa pun
git add .
./commit-ai
```

## Konvensi Kode

- Struktur mengikuti layout yang ada: `cmd/` (Cobra commands), `internal/` (git, ai, prompt, config, ui)
- Setiap layer terpisah lewat interface (`git.Runner`, `ai.Provider`) — pertahankan agar mudah di-test
- Komentar kode dalam bahasa Indonesia, identifier dalam bahasa Inggris
- Setiap fitur baru **wajib** disertai unit test; fungsi Git/Ollama di-test lewat mock, bukan proses nyata
- Output terminal tidak boleh bergantung hanya pada warna (aksesibilitas)

## Konvensi Commit Message

Projek ini memakai [Conventional Commits](https://www.conventionalcommits.org) — pratilik saja dengan tool ini sendiri 😉

```
feat: menambahkan opsi --dry-run
fix(git): perbaiki deteksi repository pada subdirektori
docs: perbarui contoh konfigurasi
```

Type yang valid: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.

## Alur Pull Request

1. **Fork** repo, lalu buat branch dari `main`:
   ```bash
   git checkout -b feat/nama-fitur
   ```
2. Pastikan semua lulus sebelum push:
   ```bash
   go vet ./...
   go test ./...
   go build ./...
   ```
3. Commit dengan pesan Conventional Commits.
4. Push ke fork kamu dan buka **Pull Request** ke `main`.
5. Isi template PR — jelaskan *apa* dan *kenapa*, bukan hanya *bagaimana*.

## Melapor Masalah Keamanan

CommitAI adalah tool privacy-first. Jika kamu menemukan kerentanan (misalnya kebocoran data ke jaringan eksternal), **jangan buka Issue publik** — hubungi pemilik repo melalui halaman profil [@zackly20](https://github.com/zackly20).

## Gagasan Fitur?

Lihat bagian [Roadmap di README](README.md#roadmap) dulu — fitur Phase 2+ (`review`, `explain`, `changelog`, integrasi IDE) terbuka untuk dikontribusikan. Diskusikan dulu di Issue sebelum implementasi besar agar tidak bentrok dengan arah produk.
