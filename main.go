//go:generate go run cmd/gen-registries-data/main.go
//go:generate go run cmd/gen-packet-id/main.go
//go:generate go run cmd/gen-fields/main.go
//go:generate go run cmd/gen-version/main.go

package main

import (
	"github.com/Gagonlaire/mcgoserv/internal/server"
)

func main() {
	serv := server.NewServer()

	serv.Start()
}
