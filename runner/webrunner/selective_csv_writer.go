package webrunner

import (
	"context"
	"encoding/csv"
	"errors"
	"slices"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/gosom/scrapemate"
)

var _ scrapemate.ResultWriter = (*selectiveCSVWriter)(nil)

type selectiveCSVWriter struct {
	writer  *csv.Writer
	fields  []string
	headers []string
	indexes []int
	wroteH  bool
}

func newSelectiveCSVWriter(w *csv.Writer, fields []string) scrapemate.ResultWriter {
	return &selectiveCSVWriter{
		writer: w,
		fields: fields,
	}
}

func (s *selectiveCSVWriter) Run(_ context.Context, in <-chan scrapemate.Result) error {
	for result := range in {
		entry, ok := result.Data.(*gmaps.Entry)
		if !ok {
			return errors.New("invalid data type")
		}

		if err := s.initFromEntry(entry); err != nil {
			return err
		}

		row := entry.CsvRow()
		out := make([]string, 0, len(s.indexes))
		for _, idx := range s.indexes {
			out = append(out, row[idx])
		}

		if err := s.writer.Write(out); err != nil {
			return err
		}
	}

	s.writer.Flush()

	return s.writer.Error()
}

func (s *selectiveCSVWriter) initFromEntry(entry *gmaps.Entry) error {
	if s.wroteH {
		return nil
	}

	allHeaders := entry.CsvHeaders()
	s.headers = allHeaders

	// empty selection => all fields
	if len(s.fields) == 0 {
		s.indexes = make([]int, len(allHeaders))
		for i := range allHeaders {
			s.indexes[i] = i
		}
	} else {
		for _, f := range s.fields {
			idx := slices.Index(allHeaders, f)
			if idx == -1 {
				continue
			}
			s.indexes = append(s.indexes, idx)
		}

		if len(s.indexes) == 0 {
			return errors.New("no valid output fields selected")
		}
	}

	headerOut := make([]string, 0, len(s.indexes))
	for _, idx := range s.indexes {
		headerOut = append(headerOut, s.headers[idx])
	}

	if err := s.writer.Write(headerOut); err != nil {
		return err
	}

	s.wroteH = true

	return nil
}
