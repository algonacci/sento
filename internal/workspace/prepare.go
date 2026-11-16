package workspace

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var placeholder = regexp.MustCompile(`\{\{\s*([^{}]+?)\s*\}\}`)

func (d *Document) FindRequest(id string) (*Project, *Request) {
	for pi := range d.Projects {
		p := &d.Projects[pi]
		for fi := range p.Folders {
			for ri := range p.Folders[fi].Requests {
				r := &p.Folders[fi].Requests[ri]
				if r.ID == id {
					return p, r
				}
			}
		}
	}
	return nil, nil
}

// Prepare resolves one snapshot. Secrets are read at execution time, never
// written back into the workspace. Relative files use the workspace directory.
func Prepare(p Project, r Request, environment, dir string, lookup func(string) (string, bool)) (Request, HTTPOptions, error) {
	if environment != "" {
		p.ActiveEnv = environment
		if p.ActiveEnvironment() == nil {
			return Request{}, HTTPOptions{}, fmt.Errorf("unknown environment %q", environment)
		}
	}
	vars := p.Variables()
	var resolve func(string, map[string]bool) (string, error)
	resolve = func(s string, visiting map[string]bool) (string, error) {
		var firstErr error
		out := placeholder.ReplaceAllStringFunc(s, func(token string) string {
			name := strings.TrimSpace(token[2 : len(token)-2])
			if firstErr != nil {
				return token
			}
			if strings.HasPrefix(name, "env.") {
				value, ok := lookup(strings.TrimPrefix(name, "env."))
				if !ok {
					firstErr = fmt.Errorf("missing environment variable %q", name)
				}
				return value
			}
			if visiting[name] {
				firstErr = fmt.Errorf("cyclic variable %q", name)
				return token
			}
			value, ok := Lookup(vars, name)
			if !ok {
				firstErr = fmt.Errorf("unknown variable %q", name)
				return token
			}
			visiting[name] = true
			value, firstErr = resolve(value, visiting)
			delete(visiting, name)
			return value
		})
		return out, firstErr
	}
	if r.Auth.Type == "" || r.Auth.Type == AuthInherit {
		r.Auth = p.Auth
	}
	r = Clone(r)
	options := p.HTTP
	fields := []*string{&r.URL, &r.JSON, &r.Raw, &r.File, &r.Auth.Token, &r.Auth.Username, &r.Auth.Password, &r.Auth.Header, &r.Auth.Value, &options.Proxy}
	for _, pairs := range [][]Pair{r.Headers, r.Params, r.Form} {
		for i := range pairs {
			fields = append(fields, &pairs[i].Key, &pairs[i].Value)
		}
	}
	for _, field := range fields {
		value, err := resolve(*field, map[string]bool{})
		if err != nil {
			return Request{}, HTTPOptions{}, err
		}
		*field = value
	}
	if r.Body == BodyJSON && strings.TrimSpace(r.JSON) != "" && !json.Valid([]byte(r.JSON)) {
		return Request{}, HTTPOptions{}, fmt.Errorf("request %q: body is not valid JSON after variable substitution", r.ID)
	}
	if r.File != "" && !filepath.IsAbs(r.File) {
		r.File = filepath.Join(dir, r.File)
	}
	return r, options, nil
}
