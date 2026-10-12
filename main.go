package main

import (
	"log"
	"os"

	"job-aggregator/internal/db"
	"job-aggregator/internal/ui"

	"gioui.org/app"
)

func main() {
	// Initialize local SQLite database
	db.InitDB()

	// Run Gio UI Application loop in a separate goroutine or main thread depending on OS requirements
	go func() {
		if err := ui.Run(); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
