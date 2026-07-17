// Command contix syncs AI coding-agent state to a user-owned Git repository and
// restores it on another machine.
package main

import (
	"os"

	"contix/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
