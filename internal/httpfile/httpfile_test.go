package httpfile

import (
	"strings"
	"testing"

	"github.com/algonacci/sento/internal/workspace"
)

func TestParsePreservesRequestData(t *testing.T) {
	source := "@host = https://example.com\r\n### Create item\r\n# @name create\r\nPOST {{host}}/items HTTP/1.1\r\nX-Tag: one\r\nX-Tag: two\r\nContent-Type: application/json\r\n\r\n{\r\n  \"name\": \"Sento\"\r\n}\r\n### Read\r\n# @name read\r\nGET {{host}}/items\r\n"
	d, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	p, r := d.FindRequest("create")
	if p == nil || r == nil {
		t.Fatal("missing request")
	}
	if r.Name != "Create item" || r.Method != "POST" || len(r.Headers) != 3 || r.Headers[1].Value != "two" || r.Raw != "{\n  \"name\": \"Sento\"\n}" {
		t.Fatalf("lost request data: %+v", r)
	}
	encoded, err := workspace.Encode(d)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := workspace.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	_, got := roundTrip.FindRequest("create")
	if got.Raw != r.Raw {
		t.Fatal("JSON round trip changed body")
	}
}

func TestParseRejectsUnsupportedAndMalformedInput(t *testing.T) {
	for _, source := range []string{
		"", "nonsense", "GET https://example.com\nBroken header",
		"# @no-redirect\nGET https://example.com",
		"GET https://example.com\n\n> {% client.test('x') %}",
		"GET https://example.com\n\n< ./payload.json",
		"# @name duplicate\nGET https://example.com\n###\n# @name duplicate\nGET https://example.com",
		"@host = one\n@host = two\nGET {{host}}",
		"BREW https://example.com",
	} {
		t.Run(strings.ReplaceAll(source, "\n", " "), func(t *testing.T) {
			if _, err := Parse(source); err == nil {
				t.Fatal("accepted invalid source")
			}
		})
	}
}
