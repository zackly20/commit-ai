## Deskripsi

Jelaskan apa yang diubah Pull Request ini dan mengapa perlu.

Closes #(nomor issue, jika ada)

## Jenis Perubahan

- [ ] 🐛 Bug fix (perbaikan yang tidak mengubah perilaku yang ada)
- [ ] ✨ Fitur baru (perubahan yang menambah fungsionalitas)
- [ ] 💥 Breaking change (perbaikan/fitur yang merusak perilaku yang ada)
- [ ] 📝 Dokumentasi
- [ ] ♻️ Refactor (tanpa perubahan perilaku)
- [ ] ✅ Test

## Cara Dites

Jelaskan tes yang kamu lakukan untuk memverifikasi perubahan:

- [ ] `go vet ./...` lulus
- [ ] `go test ./...` lulus (semua unit + integration test dengan mock)
- [ ] `go build ./...` lulus
- [ ] Dites manual dengan mock server (`go run ./tools/mock-ollama`) — jelaskan langkahnya di bawah

```
Langkah manual test (jika ada):
1. ...
2. ...
```

## Checklist

- [ ] Kode mengikuti konvensi project (lihat CONTRIBUTING.md)
- [ ] Fitur baru disertai unit test
- [ ] Commit message memakai Conventional Commits
- [ ] Dokumentasi (README/CONTRIBUTING) diperbarui jika relevan
- [ ] Tidak ada secret/token yang ter-commit
