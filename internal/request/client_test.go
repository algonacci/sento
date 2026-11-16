package request

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/algonacci/sento/internal/workspace"
)

func TestSendRawBodyRepeatedFieldsAndInheritedAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "virtual.example" {
			t.Errorf("host %q", r.Host)
		}
		if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"one", "two", "three"}) {
			t.Errorf("query: %s", r.URL.RawQuery)
		}
		if !reflect.DeepEqual(r.Header.Values("X-Tag"), []string{"one", "two"}) {
			t.Errorf("headers: %v", r.Header)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("auth: %s", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello\nworld" {
			t.Errorf("body: %q", body)
		}
		w.Header().Add("X-Reply", "one")
		w.Header().Add("X-Reply", "two")
		w.WriteHeader(201)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()
	r := workspace.Request{Method: "POST", URL: srv.URL + "?tag=one", Body: workspace.BodyRaw, Raw: "hello\nworld",
		Params:  []workspace.Pair{{Key: "tag", Value: "two"}, {Key: "tag", Value: "three"}},
		Headers: []workspace.Pair{{Key: "Host", Value: "virtual.example"}, {Key: "X-Tag", Value: "one"}, {Key: "X-Tag", Value: "two"}},
	}
	result := Send(context.Background(), r, workspace.Auth{Type: workspace.AuthBearer, Token: "{{token}}"}, []workspace.Pair{{Key: "token", Value: "secret"}}, srv.Client())
	if result.Err != "" || result.Code != 201 || string(result.Body) != `{"ok":true}` {
		t.Fatalf("result: %+v", result)
	}
}

func TestCancellationAndRedirectOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/next", http.StatusFound) }))
	defer srv.Close()
	client, err := Build(workspace.HTTPOptions{NoRedirects: true, TimeoutSeconds: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.CloseIdleConnections()
	r := workspace.Request{Method: "GET", URL: srv.URL}
	if got := Send(context.Background(), r, workspace.Auth{}, nil, client); got.Code != 302 {
		t.Fatalf("redirect: %+v", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := Send(ctx, r, workspace.Auth{}, nil, client); got.Err == "" {
		t.Fatal("canceled request succeeded")
	}
}
