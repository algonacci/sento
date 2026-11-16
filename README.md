# Sento

API client ala-ala, desktop native pakai Go + [MyGo](https://mygo.egoist.dev).

## Status

Scaffold awal: window native mengikuti tema sistem, sidebar workspace, draft
request (method, URL, body), reset lewat **New request**, dan empty state response.
Draft masih in-memory. Tombol **Send** belum aktif; HTTP execution, collections,
history, dan persistence jadi iterasi berikutnya.

## Development

Butuh Go **1.27.3** dan native build dependencies untuk OS yang dipakai sesuai
[dokumentasi MyGo](https://mygo.egoist.dev). MyGo dipin ke **v0.7.4**;
CLI tersedia lewat Go tool dependency, tanpa install global.

```sh
go mod download
go tool mygo dev       # jalankan dengan live reload
go tool mygo build     # package aplikasi ke build/
```

Shortcut: `make dev`, `make build`, `make check`, dan `make fmt`.

```sh
go test ./...
go vet ./...
go tool mygo vet .
```

## Struktur

```text
main.go     Bootstrap MyGo, window native, dan adapter desktop
app.go      State draft dan actions; host interface untuk UI updates
view.go     Layout API client dengan native UI
theme.go    Penyesuaian tema sistem
mygo.json   Metadata aplikasi dan output packaging
```

Side effect berjalan dari action, bukan render. I/O nanti dijalankan di
goroutine, lalu hasil diterapkan melalui `host.Update` (asynchronous).
Simpan data atau `ui.Handle` di state, bukan element atau `*ui.Context`.

## License

[MIT](LICENSE)
