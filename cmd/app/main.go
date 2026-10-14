package main

import (
	"log"
	"os"

	"job-aggregator/internal/ui"

	"gioui.org/app"
)

func main() {
	go func() {
		if err := ui.Run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
