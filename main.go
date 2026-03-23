//go:generate go run ./cmd/gen-registries-data
//go:generate go run ./cmd/gen-packet-id
//go:generate go run ./cmd/gen-prismarine-js

//go:generate go run ./cmd/gen-field
//go:generate go run ./cmd/gen-version

package main

import (
	"github.com/Gagonlaire/mcgoserv/internal/server"
	"github.com/Gagonlaire/mcgoserv/internal/server/commands"
)

func main() {
	serv := server.NewServer()

	commands.RegisterAll(serv)
	serv.Start()
}
