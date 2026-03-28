package main

import (
	"aimc-go/app/server"
)

func main() {
	app := server.New()
	app.Run()
}
