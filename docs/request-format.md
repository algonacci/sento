# Sento request format v1

## `.http`: definition files for humans and AI

Each request consists of an optional title (`### Title`), an optional stable ID
(`# @name request-id`), a request line (`METHOD URL`), headers, and an optional
body after a blank line. Separate requests with a line starting with `###`.
Optional `HTTP/1.1` or `HTTP/2` on the request line is accepted as notation;
the Go transport negotiates the actual HTTP version.

Supported methods: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS.
Headers and query parameters can repeat. Bodies are raw text, so JSON, XML,
GraphQL and form-urlencoded payloads work by setting the appropriate
Content-Type. CRLF is normalized to LF; the final file newline is not included
in the body. Use JSON file-body mode for byte-exact payloads.

`#` and `//` comment lines are accepted before the request line; `#` comments
are accepted among headers. Inside a body, these are literal payload text.
Lines beginning `###` always separate requests.

Declare file-global variables once, outside a request:

```http
@host = https://api.example.com
@baseUrl = {{host}}/v1

### Read users
# @name list-users
GET {{baseUrl}}/users
Authorization: Bearer {{env.API_TOKEN}}
```

Missing IDs become `request-1`, `request-2`, etc. Prefer explicit stable IDs for
automation. IDs are unique across a document and use letters, digits, dots,
underscores and hyphens, beginning with a letter or digit. The `.http` importer
reserves `http-project` and `http-requests` for its project and folder IDs.

Variables are recursively resolved; unknown names, missing OS environment
variables and cycles fail before sending. `env.` is reserved for OS variables.
Substitution is literal text, not JSON escaping or URL encoding. File variables
are available to every request regardless of declaration order. A request's
variable references are resolved only when it is run; validation does not need
runtime secrets. `.http` requests use a 30-second timeout.

Unsupported: script hooks, `< file` body directives, response handlers/chaining,
dynamic REST Client/JetBrains variables, and other `# @...` metadata. Recognized
unsupported directives produce errors rather than running scripts.

## JSON: projects, environments, options, portable export

The machine-readable schema is [`internal/workspace/schema.json`](../internal/workspace/schema.json)
and is also available with `sento schema`. Runtime validation additionally checks
global ID uniqueness, active references, variable names and duplicate variables.
Unknown fields and unsupported versions are rejected. Maximum input size is
8 MiB for both formats.

```json
{
  "version": 1,
  "active": "users-api",
  "projects": [{
    "id": "users-api",
    "name": "Users API",
    "auth": {"type": "bearer", "token": "{{env.API_TOKEN}}"},
    "http": {"timeoutSeconds": 30, "noRedirects": false},
    "vars": [{"key": "baseUrl", "value": "http://localhost:3000"}],
    "environments": [{
      "id": "staging",
      "name": "Staging",
      "vars": [{"key": "baseUrl", "value": "https://staging.example.com"}]
    }],
    "folders": [{
      "id": "users",
      "name": "Users",
      "requests": [{
        "id": "list-users",
        "name": "List users",
        "method": "GET",
        "url": "{{baseUrl}}/users",
        "auth": {"type": "inherit"},
        "headers": [{"key": "Accept", "value": "application/json"}]
      }]
    }]
  }]
}
```

Environment variables override project variables. `activeEnv` chooses an
environment per project; `sento run --env ID` overrides it for that execution.
Request auth defaults to project auth; `{"type":"none"}` disables inheritance.
Other auth types: `bearer` (`token`), `basic` (`username`, `password`), `key`
(`header`, `value`; header defaults to X-API-Key).

Body modes:
- `none` or omitted: no body.
- `json`: JSON text in the `json` field, checked after substitution.
- `raw`: literal text in `raw`; Content-Type comes from headers.
- `form`: multipart fields from `form` pairs, optional `file` attachment.
- `file`: bytes from `file`, relative to the workspace file's directory.

HTTP options: `timeoutSeconds` (0 means no overall timeout), `noRedirects`,
`insecureTLS`, `noCookies`, and `proxy`. URL without a scheme defaults to HTTPS.
HTTP and HTTPS are the supported network schemes.

Export preserves placeholders and definitions; runtime-resolved values are not
written back. Files are plain text and literal values remain literal values on
export. Use `{{env.NAME}}` for credentials that should be supplied at run time.
JSON → `.http` serialization is not implemented; opening a JSON workspace keeps
the full JSON source in the editor.

## AI integration contract

External agents can read this grammar or the schema, write a file, call
`sento validate`, and run a stable request ID using `sento run --request ID`.
The built-in AI generator follows the same `.http` grammar and parser.
No proprietary tool protocol is required for this workflow.
