//go:generate go run ./cmd/gen-registries-data
//go:generate go run ./cmd/gen-packet-id
//go:generate go run ./cmd/gen-prismarine-js
//go:generate go run ./cmd/gen-version

package main

import (
	"os"

	"github.com/Gagonlaire/mcgoserv/internal/mc/block"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	"github.com/Gagonlaire/mcgoserv/internal/server/commands"
)

func main() {
	serv := server.NewServer()

	// TODO: check if not breaking, as world load happens before block registration
	block.RegisterAll()
	commands.RegisterAll(serv)
	serv.Start(os.Args[1:])
}
