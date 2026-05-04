package runner

import (
	"github.com/gosom/google-maps-scraper/geojsonfilter"
	"github.com/gosom/scrapemate"
)

// WrapWritersWithGeoJSON returns writers unchanged when no GeoJSON filter was loaded.
func WrapWritersWithGeoJSON(cfg *Config, writers []scrapemate.ResultWriter) []scrapemate.ResultWriter {
	if cfg == nil || cfg.GeoJSONFilter == nil {
		return writers
	}

	return geojsonfilter.WrapWriters(cfg.GeoJSONFilter, cfg.GeoJSONIncludeNoCoords, writers)
}
