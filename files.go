package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/algonacci/sento/internal/httpfile"
	"github.com/algonacci/sento/internal/workspace"
)

func (a *app) openFile() {
	if a.fileBusy {
		return
	}
	a.fileBusy = true
	go func() {
		path, err := a.host.OpenFile()
		var data []byte
		format := strings.ToLower(filepath.Ext(path))
		if err == nil && path != "" {
			var f *os.File
			f, err = os.Open(path)
			if err == nil {
				data, err = io.ReadAll(io.LimitReader(f, workspace.MaxDocumentSize+1))
				f.Close()
			}
			if err == nil {
				switch format {
				case ".http":
					_, err = httpfile.Parse(string(data))
				case ".json":
					_, err = workspace.Decode(data)
				default:
					err = fmt.Errorf("choose a .http or .json file")
				}
			}
		}
		a.host.Update(func() {
			if a.closed {
				return
			}
			a.fileBusy = false
			if err != nil {
				a.message = err.Error()
				return
			}
			if path == "" {
				return
			}
			a.stop()
			a.path, a.format, a.source = path, format, string(data)
			a.parse()
			a.sourceMode = false
		})
	}()
}

func (a *app) saveFile(export bool) {
	if a.fileBusy || (a.sourceMode && !a.parse()) {
		return
	}
	data, name := []byte(a.source), a.path
	if name == "" {
		name = "requests" + a.format
	}
	if export || !a.sourceMode {
		var err error
		data, err = workspace.Encode(a.doc)
		if err != nil {
			a.message = err.Error()
			return
		}
		name = "workspace.sento.json"
	}
	a.fileBusy = true
	go func() {
		path, err := a.host.SaveFile(name)
		if err == nil && path != "" {
			err = workspace.Write(path, data)
		}
		a.host.Update(func() {
			if a.closed {
				return
			}
			a.fileBusy = false
			if err != nil {
				a.message = err.Error()
			} else if path != "" {
				a.message = "Saved " + path
				if !export {
					a.path = path
				}
			}
		})
	}()
}
