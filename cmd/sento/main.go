// Command sento is the headless companion to the native desktop app.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/algonacci/sento/internal/ai"
	"github.com/algonacci/sento/internal/project"
	"github.com/algonacci/sento/internal/request"
	"github.com/algonacci/sento/internal/workspace"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: sento <validate|list|run|export|generate|schema> [options] file")
	}
	command := args[0]
	if command == "schema" {
		_, err := out.Write(workspace.Schema)
		return err
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	id := fs.String("request", "", "request ID to run")
	env := fs.String("env", "", "environment ID")
	base := fs.String("base-url", os.Getenv("SENTO_AI_BASE_URL"), "OpenAI-compatible base URL, including /v1 if needed")
	model := fs.String("model", os.Getenv("SENTO_AI_MODEL"), "AI model")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if command == "generate" {
		if fs.NArg() != 0 {
			return fmt.Errorf("generate reads an API description from stdin")
		}
		prompt, err := io.ReadAll(io.LimitReader(in, (32<<10)+1))
		if err != nil {
			return err
		}
		text, err := ai.Generate(ctx, ai.Config{BaseURL: *base, Model: *model, APIKey: os.Getenv("SENTO_AI_API_KEY")}, string(prompt))
		if err != nil {
			return err
		}
		_, err = io.WriteString(out, text)
		return err
	}
	switch command {
	case "validate", "list", "run", "export":
	default:
		return fmt.Errorf("unknown command %q", command)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%s expects one .http or .json file; flags go before the path", command)
	}
	path := fs.Arg(0)
	d, err := project.Load(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	switch command {
	case "validate":
		return enc.Encode(map[string]any{"valid": true, "version": d.Version, "projects": len(d.Projects)})
	case "list":
		type entry struct {
			ID      string `json:"id"`
			Project string `json:"project"`
			Name    string `json:"name"`
			Method  string `json:"method"`
			URL     string `json:"url"`
		}
		entries := []entry{}
		for _, p := range d.Projects {
			for _, f := range p.Folders {
				for _, r := range f.Requests {
					entries = append(entries, entry{r.ID, p.ID, r.Name, r.Method, r.URL})
				}
			}
		}
		return enc.Encode(entries)
	case "export":
		data, err := workspace.Encode(d)
		if err != nil {
			return err
		}
		_, err = out.Write(data)
		return err
	case "run":
		p, r := d.FindRequest(*id)
		if r == nil {
			return fmt.Errorf("request %q not found; pass --request with an ID from list", *id)
		}
		req, opts, err := workspace.Prepare(*p, *r, *env, filepath.Dir(path), os.LookupEnv)
		if err != nil {
			return err
		}
		client, err := request.Build(opts, nil)
		if err != nil {
			return err
		}
		defer client.CloseIdleConnections()
		result := request.Send(ctx, req, workspace.Auth{}, nil, client)
		if err := enc.Encode(struct {
			Code       int              `json:"code"`
			Status     string           `json:"status"`
			DurationMS float64          `json:"durationMs"`
			Headers    []workspace.Pair `json:"headers"`
			Body       string           `json:"body"`
			Error      string           `json:"error,omitempty"`
		}{result.Code, result.Status, float64(result.Duration.Microseconds()) / 1000, result.Headers, string(result.Body), result.Err}); err != nil {
			return err
		}
		if result.Err != "" {
			return fmt.Errorf("request failed: %s", result.Err)
		}
		if result.Code >= 400 {
			return fmt.Errorf("request returned HTTP %d", result.Code)
		}
	}
	return nil
}
