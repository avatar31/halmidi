package main

import (
	"os"

	app "github.com/avatar31/halmidi/cmd"
)

func main() {
	app.Init(os.Args[1:])
}
