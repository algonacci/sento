// Package project is the format-neutral boundary shared by GUI and automation.
package project

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/algonacci/sento/internal/httpfile"
	"github.com/algonacci/sento/internal/workspace"
)

func Load(path string) (*workspace.Document, error) {
	if strings.EqualFold(filepath.Ext(path), ".http") {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, workspace.MaxDocumentSize+1))
		if err != nil {
			return nil, err
		}
		return httpfile.Parse(string(data))
	}
	return workspace.Load(path)
}
