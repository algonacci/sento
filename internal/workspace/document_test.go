package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validDocument = `{"version":1,"projects":[{"id":"api","name":"API","folders":[{"id":"items","name":"Items","requests":[{"id":"get","name":"Get","method":"GET","url":"{{baseUrl}}/items"}]}]}]}`

func TestDocumentValidationAndAtomicWrite(t *testing.T) {
	d, err := Decode([]byte(validDocument))
	if err != nil {
		t.Fatal(err)
	}
	data, err := Encode(d)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "workspace.json")
	if err := Write(path, data); err != nil {
		t.Fatal(err)
	}
	d.Projects[0].Name = "Updated"
	data, _ = Encode(d)
	if err := Write(path, data); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got.Projects[0].Name != "Updated" {
		t.Fatalf("load: %v %+v", err, got)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Fatalf("temporary file leaked: %v", entries)
	}
	if !json.Valid(Schema) {
		t.Fatal("invalid schema JSON")
	}
	for _, source := range []string{
		strings.Replace(validDocument, `"version":1`, `"version":2`, 1),
		strings.Replace(validDocument, `"method":"GET"`, `"method":"GET","typo":true`, 1),
		strings.Replace(validDocument, `"id":"get"`, `"id":"api"`, 1),
		strings.Replace(validDocument, `"version":1`, `"version":1,"active":"absent"`, 1),
		validDocument + "{}",
	} {
		if _, err := Decode([]byte(source)); err == nil {
			t.Fatalf("accepted %s", source)
		}
	}
}

func TestPrepareEnvironmentsSecretsAndCycles(t *testing.T) {
	p := Project{Vars: []Pair{{Key: "host", Value: "https://default.example"}, {Key: "baseUrl", Value: "{{host}}/v1"}},
		Auth:         Auth{Type: AuthBearer, Token: "{{env.API_TOKEN}}"},
		Environments: []Environment{{ID: "staging", Vars: []Pair{{Key: "host", Value: "https://staging.example"}}}},
	}
	r := Request{Method: "POST", URL: "{{baseUrl}}/items", Body: BodyFile, File: "payload.txt", Headers: []Pair{{Key: "X-Host", Value: "{{host}}"}}}
	lookup := func(key string) (string, bool) { return "token", key == "API_TOKEN" }
	got, _, err := Prepare(p, r, "staging", "/project", lookup)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "https://staging.example/v1/items" || got.Auth.Token != "token" || got.File != "/project/payload.txt" {
		t.Fatalf("unexpected resolved request: %+v", got)
	}
	if r.Headers[0].Value != "{{host}}" || p.Auth.Token != "{{env.API_TOKEN}}" {
		t.Fatal("mutated source while resolving")
	}
	if _, _, err := Prepare(p, r, "missing", "", lookup); err == nil {
		t.Fatal("accepted unknown environment")
	}
	if _, _, err := Prepare(p, r, "", "", func(string) (string, bool) { return "", false }); err == nil {
		t.Fatal("accepted missing secret")
	}
	p.Vars = []Pair{{Key: "baseUrl", Value: "{{host}}"}, {Key: "host", Value: "{{baseUrl}}"}}
	if _, _, err := Prepare(p, r, "", "", lookup); err == nil || !strings.Contains(err.Error(), "cyclic") {
		t.Fatalf("cycle: %v", err)
	}
}
