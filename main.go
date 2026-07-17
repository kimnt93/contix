// Command contix syncs coding-agent state and portable machine configuration to
// a single GitHub repo and restores it on another machine.
package main

import (
	"os"

	"contix/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
