package main

import (
	"github.com/a13labs/m3uproxy/server/cmd"

	_ "github.com/a13labs/m3uproxy/server/cmd/server"
	_ "github.com/a13labs/m3uproxy/server/cmd/users"
)

func main() {
	cmd.Execute()
}
