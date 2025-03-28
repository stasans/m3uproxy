package main

import (
	"github.com/a13labs/m3uproxy/server/cmd"

	_ "github.com/a13labs/m3uproxy/cli/cmd/playlist"
	_ "github.com/a13labs/m3uproxy/cli/cmd/server"
	_ "github.com/a13labs/m3uproxy/cli/cmd/users"
)

func main() {
	cmd.Execute()
}
