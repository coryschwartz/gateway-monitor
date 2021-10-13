package task

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	shell "github.com/ipfs/go-ipfs-api"
)

type Task interface {
	Run(context.Context, *shell.Shell, string) error
	Registration() *Registration
}

type Registration struct {
	Collectors []prometheus.Collector
	Schedule   string
}
