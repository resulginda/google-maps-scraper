package web

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gosom/google-maps-scraper/exiter"
)

const (
	maxLiveLogs   = 100
	maxLivePlaces = 120
)

type LiveLogEntry struct {
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}

type CSVPlacePreview struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Address  string `json:"address"`
	Link     string `json:"link"`
}

type LiveJobState struct {
	JobID           string            `json:"job_id"`
	JobName         string            `json:"job_name"`
	Status          string            `json:"status"`
	StartedAt       time.Time         `json:"started_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	ElapsedSeconds  int               `json:"elapsed_seconds"`
	SeedCount       int               `json:"seed_count"`
	SeedCompleted   int               `json:"seed_completed"`
	PlacesFound     int               `json:"places_found"`
	PlacesCompleted int               `json:"places_completed"`
	CSVRows         int               `json:"csv_rows"`
	ProgressPercent int               `json:"progress_percent"`
	Logs            []LiveLogEntry    `json:"logs"`
	RecentPlaces    []CSVPlacePreview `json:"recent_places"`
}

type liveJobInternal struct {
	state           LiveJobState
	lastSeedDone    int
	lastPlacesFound int
	lastPlacesDone  int
	lastCSVRows     int
	completeSince   time.Time
	lastSnap        exiter.Snapshot
	lastSnapChange  time.Time
}

var (
	liveMu   sync.RWMutex
	liveJobs = map[string]*liveJobInternal{}
)

func StartLiveProgress(jobID, jobName string, seedCount int) {
	liveMu.Lock()
	defer liveMu.Unlock()

	now := time.Now().UTC()
	liveJobs[jobID] = &liveJobInternal{
		lastSnapChange: now,
		state: LiveJobState{
			JobID:      jobID,
			JobName:    jobName,
			Status:     StatusWorking,
			StartedAt:  now,
			UpdatedAt:  now,
			SeedCount:  seedCount,
			Logs:       []LiveLogEntry{{Time: now, Message: "Tarama başladı"}},
		},
	}
}

func AppendLiveLog(jobID, message string) {
	liveMu.Lock()
	defer liveMu.Unlock()

	item, ok := liveJobs[jobID]
	if !ok {
		return
	}

	appendLiveLog(item, message)
	item.state.UpdatedAt = time.Now().UTC()
}

func SetLiveJobStatus(jobID, status string) {
	liveMu.Lock()
	defer liveMu.Unlock()

	item, ok := liveJobs[jobID]
	if !ok {
		return
	}

	item.state.Status = status
	item.state.UpdatedAt = time.Now().UTC()
	if status == StatusOK {
		item.state.ProgressPercent = 100
	}
	msg := "İş tamamlandı"
	if status == StatusFailed {
		msg = "İş başarısız"
	}

	appendLiveLog(item, msg)
}

func UpdateLiveProgress(jobID string, snap exiter.Snapshot, svc *Service) {
	liveMu.Lock()
	defer liveMu.Unlock()

	item, ok := liveJobs[jobID]
	if !ok {
		return
	}

	now := time.Now().UTC()
	item.state.UpdatedAt = now
	item.state.ElapsedSeconds = int(now.Sub(item.state.StartedAt).Seconds())
	item.state.SeedCount = snap.SeedCount
	item.state.SeedCompleted = snap.SeedCompleted
	item.state.PlacesFound = snap.PlacesFound
	item.state.PlacesCompleted = snap.PlacesCompleted

	if snap.SeedCompleted > item.lastSeedDone {
		appendLiveLog(item, fmt.Sprintf("Arama tamamlandı: %d / %d", snap.SeedCompleted, maxInt(snap.SeedCount, 1)))
		item.lastSeedDone = snap.SeedCompleted
	}

	if snap.PlacesFound > item.lastPlacesFound {
		appendLiveLog(item, fmt.Sprintf("Listeden %d yer bulundu (toplam)", snap.PlacesFound))
		item.lastPlacesFound = snap.PlacesFound
	}

	if snap.PlacesCompleted > item.lastPlacesDone {
		appendLiveLog(item, fmt.Sprintf("Yer detayı işlendi: %d / %d", snap.PlacesCompleted, maxInt(snap.PlacesFound, 1)))
		item.lastPlacesDone = snap.PlacesCompleted
	}

	totalWork := snap.SeedCount + snap.PlacesFound
	doneWork := snap.SeedCompleted + snap.PlacesCompleted
	if totalWork > 0 {
		item.state.ProgressPercent = minInt(99, (doneWork*100)/totalWork)
	}

	if svc != nil {
		rowCount, previews, err := svc.GetCSVPreview(jobID, maxLivePlaces)
		if err == nil {
			item.state.CSVRows = rowCount
			item.state.RecentPlaces = previews
			if rowCount > item.lastCSVRows {
				appendLiveLog(item, fmt.Sprintf("CSV satır sayısı: %d", rowCount))
				item.lastCSVRows = rowCount
			}
		}
	}

	if snap.SeedCount > 0 && snap.SeedCompleted >= snap.SeedCount && snap.PlacesCompleted >= snap.PlacesFound {
		item.state.ProgressPercent = 99
	}

	if snap != item.lastSnap {
		item.lastSnap = snap
		item.lastSnapChange = now
		item.completeSince = time.Time{}
	}

	workDone := snap.SeedCount > 0 &&
		snap.SeedCompleted >= snap.SeedCount &&
		snap.PlacesCompleted >= snap.PlacesFound

	if workDone {
		if item.completeSince.IsZero() {
			item.completeSince = now
		}
	} else {
		item.completeSince = time.Time{}
	}
}

// ShouldForceCancel returns true when scrape counters are done but scrapemate is still running.
func ShouldForceCancel(jobID string, stallAfterComplete, stallAfterIdle time.Duration) (bool, string) {
	liveMu.RLock()
	defer liveMu.RUnlock()

	item, ok := liveJobs[jobID]
	if !ok {
		return false, ""
	}

	now := time.Now().UTC()
	snap := item.lastSnap

	workDone := snap.SeedCount > 0 &&
		snap.SeedCompleted >= snap.SeedCount &&
		snap.PlacesCompleted >= snap.PlacesFound

	if workDone && !item.completeSince.IsZero() && now.Sub(item.completeSince) >= stallAfterComplete {
		return true, "Tüm aramalar ve yerler işlendi; scrapemate hâlâ açık — otomatik kapatılıyor"
	}

	if !item.lastSnapChange.IsZero() && now.Sub(item.lastSnapChange) >= stallAfterIdle {
		return true, "Uzun süredir ilerleme yok — otomatik kapatılıyor"
	}

	return false, ""
}

func GetLiveJobState(jobID string) (LiveJobState, bool) {
	liveMu.RLock()
	defer liveMu.RUnlock()

	item, ok := liveJobs[jobID]
	if !ok {
		return LiveJobState{}, false
	}

	return item.state, true
}

func appendLiveLog(item *liveJobInternal, message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}

	item.state.Logs = append(item.state.Logs, LiveLogEntry{
		Time:    time.Now().UTC(),
		Message: message,
	})

	if len(item.state.Logs) > maxLiveLogs {
		item.state.Logs = item.state.Logs[len(item.state.Logs)-maxLiveLogs:]
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}
