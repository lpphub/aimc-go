package main

import (
	"aimc-go/server"
)

func main() {
	app := server.New()
	app.Run()
}