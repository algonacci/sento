# Sento

API client native pakai Go + [MyGo](https://mygo.egoist.dev), terinspirasi dari
[Endpoint](https://github.com/yaffalhakim1/endpoint). Requests ditulis sebagai
kode yang bisa dibaca manusia, AI, dan automation, dengan request builder sebagai
tampilan utama: pilih method, isi URL, headers, dan body, lalu Send.

## Yang sudah jalan

- Request builder dengan method/URL, tab Params, Headers, Body, Auth, dan New request.
- Send/Cancel, response body dan headers; source editor opsional melalui **Source**.
- Open/Save `.http` atau workspace JSON; export `.http` menjadi workspace JSON.
- Variables `{{name}}`, secret dari `{{env.NAME}}`, dan environment pada JSON.
- CLI headless untuk validate, list, run, export, schema, dan AI generate.
- AI generator OpenAI-compatible: base URL, model, API key, prompt, preview
  `.http`, lalu **Open generated requests** untuk membukanya di editor.
- Request engine dari Endpoint: Bearer/Basic/API-key auth, query/headers,
  JSON/raw/multipart/file body, timeout, redirect, TLS, proxy.

## Mulai dari request builder

Aplikasi dibuka dengan request GET kosong. Isi URL, pilih method, lalu klik
**Send**. Untuk payload, buka tab **Body**, pilih `json` atau `raw`, lalu isi body.
Tab **Params**, **Headers**, dan **Auth** mengatur detail request tanpa menulis
source. **New request** menambah request; sidebar berpindah antar-request.

**Import** membuka `.http` atau workspace JSON ke builder. **Save as** dari
builder menyimpan workspace JSON yang mempertahankan seluruh pengaturan.
**Source** membuka representasi JSON lanjutan, **Builder** kembali ke form.
Hasil AI juga dibuka langsung di builder.

## Opsional: import `.http`

```http
@baseUrl = https://api.example.com

### List users
# @name list-users
GET {{baseUrl}}/users
Authorization: Bearer {{env.API_TOKEN}}
Accept: application/json

### Create user
# @name create-user
POST {{baseUrl}}/users
Content-Type: application/json

{"name":"Ada"}
```

Contoh siap dibuka: [`examples/requests.http`](examples/requests.http).
File yang diimpor bisa diedit lewat form. Di mode Source, klik **Validate /
refresh** untuk memperbarui daftar; **Send** mem-parse source terbaru terlebih dulu.

Format `.http` memakai subset eksplisit, bukan seluruh fitur VS Code REST Client
atau JetBrains HTTP Client. Detail grammar dan JSON ada di
[`docs/request-format.md`](docs/request-format.md).

## Development

Go **1.27.3**, MyGo **v0.7.4**, dan native build dependencies sesuai
[dokumentasi MyGo](https://mygo.egoist.dev).

```sh
go mod download
make dev             # GUI dengan live reload
make build           # package GUI ke build/
make cli             # headless binary: build/sento
make check           # tests, Go vet, MyGo vet
```

## CLI / automation

```sh
go run ./cmd/sento validate examples/requests.http
go run ./cmd/sento list examples/requests.http
go run ./cmd/sento run --request get-query examples/requests.http
go run ./cmd/sento export examples/requests.http > workspace.sento.json
go run ./cmd/sento run --request get-query --env staging workspace.sento.json
go run ./cmd/sento schema > sento.schema.json
```

`--env` membutuhkan environment dengan ID tersebut di workspace JSON; contoh
`.http` tidak mendefinisikan `staging`. Flags ditempatkan sebelum path.
Validate/list/run menghasilkan JSON. Run mencetak status, headers, body string,
dan durasi; error transport atau HTTP 4xx/5xx menghasilkan exit code 1.
Tidak ada request jaringan saat validate/list/export. Ctrl+C membatalkan run.

## AI

Konfigurasi dari panel **AI**, atau environment sebelum menjalankan aplikasi/CLI:

```sh
export SENTO_AI_BASE_URL=http://localhost:11434/v1
export SENTO_AI_MODEL=your-model
# export SENTO_AI_API_KEY=...  # jika provider membutuhkan key

printf 'Generate GET and POST requests for a users API at localhost:3000' |
  go run ./cmd/sento generate > users.http
go run ./cmd/sento validate users.http
```

Base URL boleh memakai path provider sendiri; Sento menambahkan
`/chat/completions`. AI menerima prompt yang diketik/paste, mengembalikan `.http`,
dan output diperiksa dengan parser yang sama dengan GUI/CLI. Tombol **Open
generated requests** mengganti isi editor; request dijalankan lewat Send.
Provider config dari panel hanya berlaku selama sesi; environment menyediakan
konfigurasi programmatic. Implementasi awal non-streaming dengan timeout 60 detik.

## Batas tahap ini

File disimpan eksplisit melalui Save as; belum ada autosave, recent files,
history, atau konfirmasi unsaved edits saat mengganti editor (Import, Open
generated requests). Menutup app (Cmd+W di macOS / Ctrl+W, tombol close
window, Cmd+Q) bertanya dulu bila request/AI masih berjalan atau workspace
berubah sejak terakhir dibuka/disimpan.
Response GUI dibatasi preview 256 KiB; engine membaca maksimal 64 MiB.
Cookie jar berlaku per eksekusi; belum ada sesi cookie antar-request.
JSON mendukung environment dan project options; `.http` belum membaca sidecar
config. AI belum chat multi-turn, tool calling, atau analisis response.

## Struktur

```text
main.go, app.go, view.go    Native window, state, request execution, UI
builder.go                 Request form, tabs, headers/params/auth/body
files.go, ai.go            File actions dan AI UI lifecycle
internal/httpfile/         Parser .http
internal/workspace/        Model, validation, variables, JSON schema, persistence
internal/project/          Loader bersama .http / JSON
internal/request/          HTTP engine dan response formatting dari Endpoint
internal/ai/               OpenAI-compatible .http generator
cmd/sento/                 CLI tanpa dependency MyGo
examples/                 Request files
```

Worker memakai snapshot input dan mengirim hasil melalui `host.Update`.
Cancellation memakai generation counter agar callback lama tidak menimpa state
baru. Element dan `*ui.Context` tidak disimpan di app state.

## License & credits

[MIT](LICENSE). Bagian engine/model diadaptasi dari Endpoint; atribusi, commit
asal, dan lisensi ada di [`third_party/endpoint`](third_party/endpoint).
Pola AI transport dan lifecycle mengikuti [Tomori](third_party/tomori/README.md).
