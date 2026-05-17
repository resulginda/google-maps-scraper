// Build-time helper: download GADM Turkey boundaries into data-folder.
// Usage: go run ./scripts/prepare-turkey-geojson/main.go [/gmapsdata]
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gosom/google-maps-scraper/web"
)

func main() {
	folder := "/gmapsdata"
	if len(os.Args) > 1 {
		folder = os.Args[1]
	}

	if err := os.MkdirAll(folder, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	svc := web.NewService(nil, folder)
	if err := svc.EnsureGeoJSONData(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "prepare geojson: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Turkey geojson ready under %s/geojson/tr", folder)
}
