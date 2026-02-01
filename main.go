package main

import (
	"os"

	"github.com/chris-regnier/diaryctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
