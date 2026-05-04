package geojsonfilter

import (
	"context"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/scrapemate"
	"golang.org/x/sync/errgroup"
)

var _ scrapemate.ResultWriter = (*filterWriter)(nil)

// WrapWriters returns writers unchanged when filter is nil.
// Otherwise it returns a single ResultWriter that drops non-matching
// *gmaps.Entry / []*gmaps.Entry before forwarding to inner writers.
func WrapWriters(filter *Filter, includeMissingCoords bool, inner []scrapemate.ResultWriter) []scrapemate.ResultWriter {
	if filter == nil || len(filter.multi) == 0 {
		return inner
	}

	if len(inner) == 0 {
		return inner
	}

	fw := &filterWriter{
		filter:               filter,
		includeMissingCoords: includeMissingCoords,
		inner:                inner,
	}

	return []scrapemate.ResultWriter{fw}
}

type filterWriter struct {
	filter               *Filter
	includeMissingCoords bool
	inner                []scrapemate.ResultWriter
}

func (f *filterWriter) keepEntry(e *gmaps.Entry) bool {
	if e == nil {
		return false
	}

	lat, lon := e.Latitude, e.Longtitude

	if !validLatLon(lat, lon) {
		return f.includeMissingCoords
	}

	return f.filter.Contains(lat, lon)
}

func (f *filterWriter) filterResult(r scrapemate.Result) (scrapemate.Result, bool) {
	switch data := r.Data.(type) {
	case *gmaps.Entry:
		if !f.keepEntry(data) {
			return r, false
		}

		return r, true

	case []*gmaps.Entry:
		if len(data) == 0 {
			return r, false
		}

		out := make([]*gmaps.Entry, 0, len(data))

		for _, e := range data {
			if f.keepEntry(e) {
				out = append(out, e)
			}
		}

		if len(out) == 0 {
			return r, false
		}

		r.Data = out

		return r, true

	default:
		return r, true
	}
}

func (f *filterWriter) Run(ctx context.Context, in <-chan scrapemate.Result) error {
	switch len(f.inner) {
	case 0:
		for range in {
		}

		return nil

	case 1:
		pipe := make(chan scrapemate.Result, 64)

		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			defer close(pipe)

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case r, ok := <-in:
					if !ok {
						return nil
					}

					fr, keep := f.filterResult(r)
					if !keep {
						continue
					}

					select {
					case pipe <- fr:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
		})

		g.Go(func() error {
			return f.inner[0].Run(ctx, pipe)
		})

		return g.Wait()

	default:
		chans := make([]chan scrapemate.Result, len(f.inner))

		for i := range f.inner {
			chans[i] = make(chan scrapemate.Result, 64)
		}

		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			defer func() {
				for _, ch := range chans {
					close(ch)
				}
			}()

			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case r, ok := <-in:
					if !ok {
						return nil
					}

					fr, keep := f.filterResult(r)
					if !keep {
						continue
					}

					for _, ch := range chans {
						select {
						case ch <- fr:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
				}
			}
		})

		for i := range f.inner {
			i := i

			g.Go(func() error {
				return f.inner[i].Run(ctx, chans[i])
			})
		}

		return g.Wait()
	}
}
