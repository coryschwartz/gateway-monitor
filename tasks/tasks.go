package tasks

import (
	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

var All = []task.Task{
	NewNoopTask(10),
}
