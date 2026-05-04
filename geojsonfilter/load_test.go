package geojsonfilter

import (
	"testing"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/scrapemate"
)

func TestParsePolygonContains(t *testing.T) {
	t.Parallel()

	// Small square around a point in WGS84 (lon, lat order in GeoJSON).
	geo := []byte(`{
		"type": "Polygon",
		"coordinates": [[
			[26.9, 38.2],
			[27.0, 38.2],
			[27.0, 38.3],
			[26.9, 38.3],
			[26.9, 38.2]
		]]
	}`)

	f, err := Parse(geo)
	if err != nil {
		t.Fatal(err)
	}

	// Inside (lat, lon) — Contains(lat, lon)
	if !f.Contains(38.25, 26.95) {
		t.Fatal("expected inside")
	}

	if f.Contains(38.5, 26.95) {
		t.Fatal("expected outside (lat)")
	}

	if f.Contains(38.25, 27.5) {
		t.Fatal("expected outside (lon)")
	}
}

func TestWrapWritersNilFilter(t *testing.T) {
	t.Parallel()

	inner := []scrapemate.ResultWriter(nil)
	out := WrapWriters(nil, false, inner)
	if out != nil {
		t.Fatalf("expected nil slice passthrough, got %d", len(out))
	}

	f := &Filter{multi: nil}
	out2 := WrapWriters(f, false, []scrapemate.ResultWriter{})
	if len(out2) != 0 {
		t.Fatalf("expected empty inner unchanged")
	}
}

func TestKeepEntryMissingCoords(t *testing.T) {
	t.Parallel()

	geo := []byte(`{
		"type": "Polygon",
		"coordinates": [[[0, 0], [1, 0], [1, 1], [0, 1], [0, 0]]]
	}`)

	f, err := Parse(geo)
	if err != nil {
		t.Fatal(err)
	}

	fw := &filterWriter{
		filter:               f,
		includeMissingCoords: false,
	}

	e := &gmaps.Entry{Latitude: 0, Longtitude: 0}
	if fw.keepEntry(e) {
		t.Fatal("expected drop when coords missing")
	}

	fw.includeMissingCoords = true
	if !fw.keepEntry(e) {
		t.Fatal("expected keep when includeMissingCoords")
	}
}
