package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const Version = 1
const MaxDocumentSize = 8 << 20

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Decode is the common import boundary for GUI, CLI, and generated documents.
// Unknown fields and versions are errors, never silently discarded.
func Decode(data []byte) (*Document, error) {
	if len(data) > MaxDocumentSize {
		return nil, errors.New("workspace exceeds 8 MiB")
	}
	var d Document
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, fmt.Errorf("workspace JSON: %w", err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return nil, errors.New("expected exactly one workspace JSON document")
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return &d, nil
}

func Load(path string) (*Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxDocumentSize+1))
	if err != nil {
		return nil, err
	}
	return Decode(data)
}

func Encode(d *Document) ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if len(data) > MaxDocumentSize {
		return nil, errors.New("workspace exceeds 8 MiB")
	}
	return data, nil
}

// Write replaces a document atomically using a temporary file in the same folder.
func Write(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".sento-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func (d *Document) Validate() error {
	if d.Version != Version {
		return fmt.Errorf("version: expected %d, got %d", Version, d.Version)
	}
	if len(d.Projects) == 0 {
		return errors.New("projects: at least one project is required")
	}
	ids := map[string]bool{}
	claim := func(id, path string) error {
		if !identifier.MatchString(id) {
			return fmt.Errorf("%s.id: use letters, digits, dots, underscores or hyphens", path)
		}
		if ids[id] {
			return fmt.Errorf("%s.id: duplicate ID %q", path, id)
		}
		ids[id] = true
		return nil
	}
	activeFound := d.Active == ""
	for pi, p := range d.Projects {
		path := fmt.Sprintf("projects[%d]", pi)
		if err := claim(p.ID, path); err != nil {
			return err
		}
		if p.ID == d.Active {
			activeFound = true
		}
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("%s.name: required", path)
		}
		if err := validateAuth(p.Auth, false); err != nil {
			return fmt.Errorf("%s.auth: %w", path, err)
		}
		if p.HTTP.TimeoutSeconds < 0 || p.HTTP.TimeoutSeconds > 86400 {
			return fmt.Errorf("%s.http.timeoutSeconds: must be between 0 and 86400", path)
		}
		if err := validateVars(p.Vars); err != nil {
			return fmt.Errorf("%s.vars: %w", path, err)
		}
		envFound := p.ActiveEnv == ""
		for ei, env := range p.Environments {
			ep := fmt.Sprintf("%s.environments[%d]", path, ei)
			if err := claim(env.ID, ep); err != nil {
				return err
			}
			if strings.TrimSpace(env.Name) == "" {
				return fmt.Errorf("%s.name: required", ep)
			}
			if env.ID == p.ActiveEnv {
				envFound = true
			}
			if err := validateVars(env.Vars); err != nil {
				return fmt.Errorf("%s.vars: %w", ep, err)
			}
		}
		if !envFound {
			return fmt.Errorf("%s.activeEnv: unknown environment %q", path, p.ActiveEnv)
		}
		for fi, f := range p.Folders {
			fp := fmt.Sprintf("%s.folders[%d]", path, fi)
			if err := claim(f.ID, fp); err != nil {
				return err
			}
			if strings.TrimSpace(f.Name) == "" {
				return fmt.Errorf("%s.name: required", fp)
			}
			for ri, r := range f.Requests {
				rp := fmt.Sprintf("%s.requests[%d]", fp, ri)
				if err := claim(r.ID, rp); err != nil {
					return err
				}
				if err := ValidateRequest(r); err != nil {
					return fmt.Errorf("%s: %w", rp, err)
				}
			}
		}
	}
	if !activeFound {
		return fmt.Errorf("active: unknown project %q", d.Active)
	}
	return nil
}

func ValidateRequest(r Request) error {
	if strings.TrimSpace(r.Name) == "" || strings.TrimSpace(r.URL) == "" {
		return errors.New("name and url are required")
	}
	switch r.Method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
	default:
		return fmt.Errorf("method: unsupported value %q", r.Method)
	}
	switch r.Body {
	case "", BodyNone, BodyJSON, BodyRaw, BodyForm, BodyFile:
	default:
		return fmt.Errorf("body: unsupported value %q", r.Body)
	}
	return validateAuth(r.Auth, true)
}

func validateAuth(a Auth, inherit bool) error {
	switch a.Type {
	case "", AuthNone, AuthBearer, AuthBasic, AuthKey:
		return nil
	case AuthInherit:
		if inherit {
			return nil
		}
	}
	return fmt.Errorf("unsupported type %q", a.Type)
}

func validateVars(vars []Pair) error {
	seen := map[string]bool{}
	for _, v := range vars {
		if !identifier.MatchString(v.Key) || strings.HasPrefix(v.Key, "env.") {
			return fmt.Errorf("invalid variable name %q (env. is reserved)", v.Key)
		}
		if seen[v.Key] {
			return fmt.Errorf("duplicate variable %q", v.Key)
		}
		seen[v.Key] = true
	}
	return nil
}
