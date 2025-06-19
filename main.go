//go:generate go run internal/generators/registries_data/main.go
//go:generate go run internal/generators/fields/main.go

package main

import (
	"github.com/Gagonlaire/mcgoserv/internal/server"
)

func main() {
	serv := server.NewServer()

	serv.Start()
}
