package commands

import (
	"github.com/urfave/cli/v2"

	shell "github.com/ipfs/go-ipfs-api"
)

var All = []*cli.Command{
	singleCommand,
	daemonCommand,
}

// utility functions
func GetIPFS(cctx *cli.Context) *shell.Shell {
	sh := new(shell.Shell)
	if cctx.IsSet("ipfs") {
		sh = shell.NewShell(cctx.String("ipfs"))
	} else {
		sh = shell.NewLocalShell()
	}

	// TODO not implemented
	if cctx.IsSet("pinning-service") {
	}
	return sh
}

func GetGW(cctx *cli.Context) string {
	args := cctx.Args()
	if len(args.Slice()) > 0 {
		return args.First()
	}
	return "https://ipfs.io"
}
