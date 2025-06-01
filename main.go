package main

import "github.com/Gagonlaire/mcgoserv/internal/server"

func main() {
	serv := server.New()

	serv.Serve()
}
