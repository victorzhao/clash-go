// icongen writes the app's icon, resources/icon.png, at the size a bundle
// needs.
//
//	go run ./cmd/icongen
package main

import (
	"log"
	"os"

	"github.com/victorzhao/clash-go/internal/app"
)

func main() {
	data := app.IconPNG(1024)
	if data == nil {
		log.Fatal("drawing the icon failed")
	}
	if err := os.MkdirAll("resources", 0o755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("resources/icon.png", data, 0o644); err != nil {
		log.Fatal(err)
	}
}
