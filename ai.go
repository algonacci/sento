package main

import (
	"context"
	"time"

	"github.com/algonacci/sento/internal/ai"
)

type aiState struct {
	open, busy                                   bool
	baseURL, model, key, prompt, answer, message string
	cancel                                       context.CancelFunc
	generation                                   int
}

func (a *app) stopAI() {
	if a.ai.cancel != nil {
		a.ai.cancel()
		a.ai.cancel = nil
	}
	a.ai.generation++
	a.ai.busy = false
}

func (a *app) generate() {
	if a.ai.busy {
		return
	}
	cfg := ai.Config{BaseURL: a.ai.baseURL, Model: a.ai.model, APIKey: a.ai.key}
	prompt := a.ai.prompt
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	a.ai.cancel, a.ai.busy = cancel, true
	a.ai.generation++
	gen := a.ai.generation
	a.ai.message, a.ai.answer = "Generating .http…", ""
	go func() {
		defer cancel()
		answer, err := ai.Generate(ctx, cfg, prompt)
		a.host.Update(func() {
			if a.closed || gen != a.ai.generation {
				return
			}
			a.ai.busy, a.ai.cancel = false, nil
			if err != nil {
				a.ai.message = err.Error()
				return
			}
			a.ai.answer, a.ai.message = answer, "Valid .http · ready to open in editor"
		})
	}()
}

func (a *app) useGenerated() {
	if a.ai.answer == "" || a.fileBusy {
		return
	}
	a.stop()
	a.source, a.format, a.path = a.ai.answer, ".http", ""
	a.ai.open = false
	a.parse()
	a.sourceMode = false
}
