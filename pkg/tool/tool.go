package tool

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/charlieegan3/tool-webhook-rss/pkg/tool/handlers"
	"github.com/charlieegan3/tool-webhook-rss/pkg/tool/jobs"
	"github.com/charlieegan3/toolbelt/pkg/apis"
	"github.com/gorilla/mux"
)

//go:embed migrations
var webhookRSSToolMigrations embed.FS

// WebhookRSS is an example tool which demonstrates the use of the database feature
type WebhookRSS struct {
	db         *sql.DB
	loadedJobs []apis.Job

	JobsDeadManEndpoint string
	JobsDeadManSchedule string

	JobsCheckSchedule      string
	JobsCheckPushoverToken string
	JobsCheckPushoverApp   string

	JobsCleanSchedule string
}

func (d *WebhookRSS) Name() string {
	return "webhook-rss"
}

func (d *WebhookRSS) FeatureSet() apis.FeatureSet {
	return apis.FeatureSet{
		HTTP:     true,
		Database: true,
		Jobs:     true,
	}
}

func (d *WebhookRSS) HTTPPath() string {
	return "webhook-rss"
}

// SetConfig is a no-op for this tool
func (d *WebhookRSS) SetConfig(config map[string]any) error {
	return nil
}

func (d *WebhookRSS) DatabaseMigrations() (*embed.FS, string, error) {
	return &webhookRSSToolMigrations, "migrations", nil
}

func (d *WebhookRSS) DatabaseSet(db *sql.DB) {
	d.db = db
}

func (d *WebhookRSS) HTTPAttach(router *mux.Router) error {
	if d.db == nil {
		return fmt.Errorf("database not set")
	}

	// handler for the creation of new items in feeds
	router.HandleFunc(
		"/feeds/{feed}/items",
		handlers.BuildItemCreateHandler(d.db),
	).Methods("POST")

	// handler used to serve rss clients
	router.HandleFunc(
		"/feeds/{feed}.rss",
		handlers.BuildFeedGetHandler(d.db),
	).Methods("GET")

	return nil
}

func (d *WebhookRSS) Jobs() []apis.Job {
	if len(d.loadedJobs) > 0 {
		return d.loadedJobs
	}

	return []apis.Job{
		&jobs.DeadMan{
			Endpoint:         d.JobsDeadManEndpoint,
			ScheduleOverride: d.JobsDeadManSchedule,
		},
		&jobs.Check{
			DB:               d.db,
			ScheduleOverride: d.JobsCheckSchedule,
			PushoverApp:      d.JobsCheckPushoverApp,
			PushoverToken:    d.JobsCheckPushoverToken,
		},
		&jobs.Clean{
			DB:               d.db,
			ScheduleOverride: d.JobsCleanSchedule,
		},
	}
}
