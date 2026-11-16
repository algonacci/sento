package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIRoundTripAndRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("local response")) }))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "api.http")
	if err := os.WriteFile(path, []byte("# @name local\nGET "+srv.URL), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, args := range [][]string{{"validate", path}, {"list", path}, {"export", path}, {"run", "--request", "local", path}, {"schema"}} {
		out.Reset()
		if err := run(context.Background(), args, strings.NewReader(""), &out); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !json.Valid(out.Bytes()) {
			t.Fatalf("non-JSON output: %s", out.String())
		}
		if args[0] == "export" {
			jsonPath := filepath.Join(filepath.Dir(path), "workspace.json")
			if err := os.WriteFile(jsonPath, out.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			out.Reset()
			if err := run(context.Background(), []string{"run", "--request", "local", jsonPath}, nil, &out); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "local response") {
				t.Fatal("exported request changed")
			}
		}
	}
}
