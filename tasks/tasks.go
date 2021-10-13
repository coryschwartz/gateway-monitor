package tasks

import (
	logging "github.com/ipfs/go-log"

	"github.com/coryschwartz/gateway-monitor/pkg/task"
)

var (
	log = logging.Logger("tasks")
)

const (
	kiB = 1024
	miB = 1024 * kiB
	giB = 1024 * miB
)

var All = []task.Task{
	NewRandomLocalBench("0 * * * *", 16*miB),
	NewRandomLocalBench("0 * * * *", 256*miB),
	NewRandomLocalBench("0 * * * *", giB),
}
