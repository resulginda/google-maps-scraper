package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	repo       JobRepository
	dataFolder string
}

const (
	turkeyADM1URL = "https://geodata.ucdavis.edu/gadm/gadm4.1/json/gadm41_TUR_1.json"
	turkeyADM2URL = "https://geodata.ucdavis.edu/gadm/gadm4.1/json/gadm41_TUR_2.json"
)

func NewService(repo JobRepository, dataFolder string) *Service {
	return &Service{
		repo:       repo,
		dataFolder: dataFolder,
	}
}

func (s *Service) Create(ctx context.Context, job *Job) error {
	return s.repo.Create(ctx, job)
}

func (s *Service) All(ctx context.Context) ([]Job, error) {
	return s.repo.Select(ctx, SelectParams{})
}

func (s *Service) Get(ctx context.Context, id string) (Job, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return fmt.Errorf("invalid file name")
	}

	datapath := filepath.Join(s.dataFolder, id+".csv")

	if _, err := os.Stat(datapath); err == nil {
		if err := os.Remove(datapath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) Update(ctx context.Context, job *Job) error {
	return s.repo.Update(ctx, job)
}

func (s *Service) SelectPending(ctx context.Context) ([]Job, error) {
	return s.repo.Select(ctx, SelectParams{Status: StatusPending, Limit: 1})
}

func (s *Service) GetCSV(_ context.Context, id string) (string, error) {
	if strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return "", fmt.Errorf("invalid file name")
	}

	datapath := filepath.Join(s.dataFolder, id+".csv")

	if _, err := os.Stat(datapath); os.IsNotExist(err) {
		return "", fmt.Errorf("csv file not found for job %s", id)
	}

	return datapath, nil
}

func (s *Service) EnsureGeoJSONData(ctx context.Context) error {
	ilDir := filepath.Join(s.dataFolder, "geojson", "tr", "il")
	ilceDir := filepath.Join(s.dataFolder, "geojson", "tr", "ilce")

	if hasGeoJSONFiles(ilDir) && hasGeoJSONFiles(ilceDir) {
		return nil
	}

	if err := os.MkdirAll(ilDir, os.ModePerm); err != nil {
		return err
	}

	if err := os.MkdirAll(ilceDir, os.ModePerm); err != nil {
		return err
	}

	client := &http.Client{Timeout: 2 * time.Minute}

	if err := s.downloadAndSplitADM1(ctx, client, ilDir); err != nil {
		return fmt.Errorf("prepare ADM1 geojson failed: %w", err)
	}

	if err := s.downloadAndSplitADM2(ctx, client, ilceDir); err != nil {
		return fmt.Errorf("prepare ADM2 geojson failed: %w", err)
	}

	return nil
}

type geoFeatureCollection struct {
	Type     string       `json:"type"`
	Name     string       `json:"name,omitempty"`
	Features []geoFeature `json:"features"`
}

type geoFeature struct {
	Type       string          `json:"type"`
	Properties map[string]any  `json:"properties"`
	Geometry   json.RawMessage `json:"geometry"`
}

func (s *Service) downloadAndSplitADM1(ctx context.Context, client *http.Client, ilDir string) error {
	var fc geoFeatureCollection

	if err := fetchJSON(ctx, client, turkeyADM1URL, &fc); err != nil {
		return err
	}

	for _, f := range fc.Features {
		name, _ := f.Properties["NAME_1"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}

		slug := slugifyTR(name)
		if slug == "" {
			continue
		}

		outPath := filepath.Join(ilDir, slug+".geojson")
		if err := writeSingleFeatureCollection(outPath, f, "TR-ADM1-"+slug); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) downloadAndSplitADM2(ctx context.Context, client *http.Client, ilceDir string) error {
	var fc geoFeatureCollection

	if err := fetchJSON(ctx, client, turkeyADM2URL, &fc); err != nil {
		return err
	}

	for _, f := range fc.Features {
		cityName, _ := f.Properties["NAME_1"].(string)
		districtName, _ := f.Properties["NAME_2"].(string)
		if strings.TrimSpace(cityName) == "" || strings.TrimSpace(districtName) == "" {
			continue
		}

		citySlug := slugifyTR(cityName)
		districtSlug := slugifyTR(districtName)
		if citySlug == "" || districtSlug == "" {
			continue
		}

		outDir := filepath.Join(ilceDir, citySlug)
		if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
			return err
		}

		outPath := filepath.Join(outDir, districtSlug+".geojson")
		if err := writeSingleFeatureCollection(outPath, f, "TR-ADM2-"+citySlug+"-"+districtSlug); err != nil {
			return err
		}
	}

	return nil
}

func fetchJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed %s: %s", url, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func writeSingleFeatureCollection(path string, feature geoFeature, name string) error {
	out := geoFeatureCollection{
		Type:     "FeatureCollection",
		Name:     name,
		Features: []geoFeature{feature},
	}

	data, err := json.Marshal(out)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func hasGeoJSONFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if e.IsDir() {
			subEntries, subErr := os.ReadDir(filepath.Join(dir, e.Name()))
			if subErr != nil {
				continue
			}

			for _, se := range subEntries {
				if !se.IsDir() && filepath.Ext(se.Name()) == ".geojson" {
					return true
				}
			}

			continue
		}

		if filepath.Ext(e.Name()) == ".geojson" {
			return true
		}
	}

	return false
}
