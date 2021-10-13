package task

import (
	"context"

	shell "github.com/ipfs/go-ipfs-api"
)

type TerminalTask struct {
	Done chan bool
}

func (t *TerminalTask) Run(context.Context, *shell.Shell, string) error {
	t.Done <- true
	return nil
}

func (t *TerminalTask) Registration() *Registration {
	return new(Registration)
}
