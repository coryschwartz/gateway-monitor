package commands

import (
	"github.com/urfave/cli/v2"

	"github.com/coryschwartz/gateway-monitor/pkg/engine"
	"github.com/coryschwartz/gateway-monitor/tasks"
)

var singleCommand = &cli.Command{
	Name:  "single",
	Usage: "run a single test",
	Action: func(cctx *cli.Context) error {
		ipfs := GetIPFS(cctx)
		ps := GetPinningService(cctx)
		gw := GetGW(cctx)
		eng := engine.NewSingle(ipfs, ps, gw, tasks.All...)
		return <-eng.Start(cctx.Context)
	},
}
