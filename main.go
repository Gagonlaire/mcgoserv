package main

import "github.com/Gagonlaire/mcgoserv/internal/server"

func main() {
	serv := server.NewServer()

	serv.Start()
}
