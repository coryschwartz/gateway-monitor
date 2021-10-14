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
	NewIpnsBench("0 * * * *", 16*miB),
	NewIpnsBench("0 * * * *", 256*miB),
	NewKnownGoodCheck("0 * * * *", map[string][]byte{
		"/ipfs/Qmc5gCcjYypU7y28oCALwfSvxCBskLuPKWpK4qpterKC7z": []byte("Hello World!\r\n"),
	}),
	NewNonExistCheck("0 * * * *"),
}
