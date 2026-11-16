// Package httpfile implements Sento's documented, portable .http subset.
package httpfile

import (
	"fmt"
	"strings"

	"github.com/algonacci/sento/internal/workspace"
)

// Parse accepts ### separators, # @name IDs, @variable = value declarations,
// HTTP request lines, repeated headers and raw bodies. It never executes scripts.
func Parse(source string) (*workspace.Document, error) {
	if len(source) > workspace.MaxDocumentSize {
		return nil, fmt.Errorf("HTTP file exceeds 8 MiB")
	}
	source = strings.TrimPrefix(source, "\ufeff")
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	p := workspace.Project{ID: "http-project", Name: "HTTP requests", HTTP: workspace.HTTPOptions{TimeoutSeconds: 30}}
	f := workspace.Folder{ID: "http-requests", Name: "Requests"}
	var req *workspace.Request
	var body []string
	name, id := "", ""
	inBody := false
	flush := func() {
		if req == nil {
			return
		}
		if len(body) > 0 {
			req.Body = workspace.BodyRaw
			req.Raw = strings.Join(body, "\n")
		}
		f.Requests = append(f.Requests, *req)
		req, body, inBody = nil, nil, false
	}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		fail := func(message string) (*workspace.Document, error) {
			return nil, fmt.Errorf("line %d: %s", i+1, message)
		}
		if strings.HasPrefix(trim, "###") {
			flush()
			name, id = strings.TrimSpace(strings.TrimPrefix(trim, "###")), ""
			continue
		}
		if strings.HasPrefix(trim, "< {%") || strings.HasPrefix(trim, "> {%") || strings.HasPrefix(trim, "> ") || strings.HasPrefix(trim, "< ") {
			return fail("script hooks and file directives are not supported; use a literal body")
		}
		if inBody {
			// The final split item represents the file's terminating newline.
			if i != len(lines)-1 || line != "" {
				body = append(body, line)
			}
			continue
		}
		if req == nil {
			if trim == "" {
				continue
			}
			if strings.HasPrefix(trim, "# @name ") {
				id = strings.TrimSpace(strings.TrimPrefix(trim, "# @name "))
				continue
			}
			if strings.HasPrefix(trim, "# @") {
				return fail("unsupported request directive (only # @name is supported)")
			}
			if strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "//") {
				continue
			}
			if strings.HasPrefix(trim, "@") {
				key, value, ok := strings.Cut(trim[1:], "=")
				if !ok {
					return fail("expected @variable = value")
				}
				p.Vars = append(p.Vars, workspace.Pair{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
				continue
			}
			parts := strings.Fields(trim)
			if len(parts) < 2 || len(parts) > 3 || (len(parts) == 3 && parts[2] != "HTTP/1.1" && parts[2] != "HTTP/2") {
				return fail("expected METHOD URL [HTTP/1.1]")
			}
			if id == "" {
				id = fmt.Sprintf("request-%d", len(f.Requests)+1)
			}
			if name == "" {
				name = id
			}
			req = &workspace.Request{ID: id, Name: name, Method: parts[0], URL: parts[1], Body: workspace.BodyNone}
			if err := workspace.ValidateRequest(*req); err != nil {
				return fail(err.Error())
			}
			continue
		}
		if trim == "" {
			inBody = true
			continue
		}
		if strings.HasPrefix(trim, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) == "" || strings.ContainsAny(strings.TrimSpace(key), " \t") {
			return fail("expected Header: value, or a blank line before the body")
		}
		req.Headers = append(req.Headers, workspace.Pair{Key: strings.TrimSpace(key), Value: strings.TrimSpace(value)})
	}
	flush()
	if len(f.Requests) == 0 {
		return nil, fmt.Errorf("HTTP file contains no requests")
	}
	p.Folders = []workspace.Folder{f}
	d := &workspace.Document{Version: workspace.Version, Active: p.ID, Projects: []workspace.Project{p}}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return d, nil
}
