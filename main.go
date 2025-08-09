package main

import (
	"golte/cmd"
)

//go:generate go run tools/generate.go

func main() {
	cmd.Execute()
}
