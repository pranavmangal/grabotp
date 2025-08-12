package main

import (
	"github.com/pranavmangal/grabotp/cmd"
)

var version = "dev"

func main() {
	cmd.Execute(version)
}
