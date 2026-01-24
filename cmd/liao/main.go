package main

import (
	"os"
)

var osExit = os.Exit
var runFn = run

func main() {
	if err := runFn(); err != nil {
		osExit(1)
	}
}
