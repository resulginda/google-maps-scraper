package geojsonfilter

import (
	"encoding/json"
	"fmt"
	"os"
)

// Filter holds one or more polygon groups (MultiPolygon union).
type Filter struct {
	// Multi format: each element is one polygon's rings (outer + holes).
	multi [][][][]float64
}

// Contains reports whether (lat, lon) lies inside any loaded polygon.
func (f *Filter) Contains(lat, lon float64) bool {
	if f == nil || len(f.multi) == 0 {
		return true
	}

	return pointInMultiPolygon(lon, lat, f.multi)
}

// LoadFile reads a GeoJSON file and builds a Filter.
// Supported root types: FeatureCollection, Feature, Polygon, MultiPolygon.
func LoadFile(path string) (*Filter, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return Parse(raw)
}

// Parse decodes GeoJSON bytes into a Filter.
func Parse(raw []byte) (*Filter, error) {
	var head struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(raw, &head); err != nil {
		return nil, fmt.Errorf("geojson: %w", err)
	}

	switch head.Type {
	case "FeatureCollection":
		return parseFeatureCollection(raw)
	case "Feature":
		return parseFeature(raw)
	case "Polygon", "MultiPolygon":
		return parseGeometryObject(raw)
	default:
		return nil, fmt.Errorf("geojson: unsupported type %q", head.Type)
	}
}

func parseFeatureCollection(raw []byte) (*Filter, error) {
	var doc struct {
		Features []json.RawMessage `json:"features"`
	}

	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	ans := &Filter{}

	for i, feat := range doc.Features {
		f, err := Parse(feat)
		if err != nil {
			return nil, fmt.Errorf("feature %d: %w", i, err)
		}

		ans.multi = append(ans.multi, f.multi...)
	}

	if len(ans.multi) == 0 {
		return nil, fmt.Errorf("geojson: no polygon geometries in FeatureCollection")
	}

	return ans, nil
}

func parseFeature(raw []byte) (*Filter, error) {
	var doc struct {
		Geometry json.RawMessage `json:"geometry"`
	}

	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	if len(doc.Geometry) == 0 {
		return nil, fmt.Errorf("geojson: Feature has no geometry")
	}

	return Parse(doc.Geometry)
}

func parseGeometryObject(raw []byte) (*Filter, error) {
	var doc struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	}

	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	switch doc.Type {
	case "Polygon":
		var coords [][][]float64

		if err := json.Unmarshal(doc.Coordinates, &coords); err != nil {
			return nil, err
		}

		if len(coords) == 0 {
			return nil, fmt.Errorf("geojson: empty Polygon")
		}

		return &Filter{multi: [][][][]float64{coords}}, nil

	case "MultiPolygon":
		var coords [][][][]float64

		if err := json.Unmarshal(doc.Coordinates, &coords); err != nil {
			return nil, err
		}

		if len(coords) == 0 {
			return nil, fmt.Errorf("geojson: empty MultiPolygon")
		}

		return &Filter{multi: coords}, nil

	default:
		return nil, fmt.Errorf("geojson: unsupported geometry type %q", doc.Type)
	}
}
