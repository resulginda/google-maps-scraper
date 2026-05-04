package geojsonfilter

import "math"

// pointInPolygon2D tests inclusion using ray casting on lon/lat treated as planar
// (adequate for administrative polygons at city/district scale).
func pointInPolygon2D(lon, lat float64, ring [][]float64) bool {
	if len(ring) < 3 {
		return false
	}

	inside := false

	n := len(ring)
	j := n - 1

	for i := 0; i < n; i++ {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]

		if yi == yj {
			j = i

			continue
		}

		if ((yi > lat) != (yj > lat)) && (lon < (xj-xi)*(lat-yi)/(yj-yi)+xi) {
			inside = !inside
		}

		j = i
	}

	return inside
}

func pointInRingWithHoles(lon, lat float64, rings [][][]float64) bool {
	if len(rings) == 0 {
		return false
	}

	if !pointInPolygon2D(lon, lat, rings[0]) {
		return false
	}

	for h := 1; h < len(rings); h++ {
		if pointInPolygon2D(lon, lat, rings[h]) {
			return false
		}
	}

	return true
}

func pointInMultiPolygon(lon, lat float64, polys [][][][]float64) bool {
	for _, poly := range polys {
		if pointInRingWithHoles(lon, lat, poly) {
			return true
		}
	}

	return false
}

func validLatLon(lat, lon float64) bool {
	if math.Abs(lat) > 90 || math.Abs(lon) > 180 {
		return false
	}

	if lat == 0 && lon == 0 {
		return false
	}

	return true
}
