package main

import (
	"os"

	"github.com/meimolihan/fan-files/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
