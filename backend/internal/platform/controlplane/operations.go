package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Operations (§F of the plan): the things an operator does to the deployment
// rather than to an organisation.
//
// The boundary this file keeps is the one §4 of the plan draws: **no shell, no
// SQL console, no environment editing.** A deployment is triggered by asking
// GitHub to run the workflow that already exists, with a token that can do
// nothing else; a backup is a script on the host reporting into a table; the
// catalogue and the migrations are read. Nothing here executes anything on the
// server, and there is deliberately no place to add it.

// BackgroundJob is one thing the platform does on a timer, and how it went.
type BackgroundJob struct {
	Name    string     `json:"name"`
	LastRun *time.Time `json:"last_run"`
	OK      bool       `json:"ok"`
	Detail  string     `json:"detail"`
	// Pending is how many are waiting — scheduled reports that have never run,
	// for instance. Zero for jobs where the idea does not apply.
	Pending int `json:"pending"`
}

// backgroundJobs answers "is anything quietly not running".
//
// Silent failure is the failure mode of every one of these: a scheduled report
// nobody receives is noticed weeks later by the person who was expecting it,
// and a catalogue that has not synced for a month looks exactly like one that
// has nothing to fetch.
func (s *Service) backgroundJobs(ctx context.Context) []BackgroundJob {
	jobs := make([]BackgroundJob, 0, 3)
	ctx = scoped(ctx)

	var lastRun *time.Time
	var failures, pending int
	if err := s.db.QueryRow(ctx,
		`SELECT max(last_run_at),
		        count(*) FILTER (WHERE last_status NOT IN ('', 'ok') AND active),
		        count(*) FILTER (WHERE last_run_at IS NULL AND active)
		   FROM report_schedules`).Scan(&lastRun, &failures, &pending); err != nil {
		slog.Warn("control plane: could not read the scheduled reports", "error", err)
	} else {
		detail := ""
		if failures > 0 {
			detail = fmt.Sprintf("%d schedule(s) failed on their last run", failures)
		}
		jobs = append(jobs, BackgroundJob{
			Name: "scheduled_reports", LastRun: lastRun, OK: failures == 0,
			Detail: detail, Pending: pending,
		})
	}

	if s.catalogStatusFrom != nil {
		at, ok, detail := s.catalogStatusFrom()
		var lastSync *time.Time
		if !at.IsZero() {
			lastSync = &at
		}
		jobs = append(jobs, BackgroundJob{
			Name: "catalog_sync", LastRun: lastSync, OK: ok, Detail: detail,
		})
	}

	// The Өртөө channel. Two numbers and no more: how much is queued for
	// another installation and has not landed, and how many live links have
	// stopped speaking. Neither reads a task or an envelope — migration 00064
	// grants the operator role exactly the two tables these come from, and
	// deliberately not the ones holding what was actually said.
	//
	// The hour is the threshold because the delivery backoff caps at six: a
	// link that has been silent for an hour is not one that is merely between
	// attempts.
	var undelivered, silent int
	if err := s.db.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM urtuu_deliveries WHERE delivered_at IS NULL),
		       (SELECT count(*) FROM urtuu_peers
		         WHERE status = 'active' AND revoked_at IS NULL
		           AND coalesce(last_seen_at, created_at) < NOW() - INTERVAL '1 hour')`).
		Scan(&undelivered, &silent); err != nil {
		// A deployment that has never run Өртөө still has the tables — the
		// migration creates them for everyone — so this really is a fault
		// rather than an absence, and it is worth a line.
		slog.Warn("control plane: could not read the Өртөө channel", "error", err)
	} else if undelivered > 0 || silent > 0 {
		detail := ""
		if silent > 0 {
			detail = fmt.Sprintf("%d link(s) have not been heard from for over an hour", silent)
		}
		jobs = append(jobs, BackgroundJob{
			Name: "urtuu_relay", OK: silent == 0, Detail: detail, Pending: undelivered,
		})
	} else {
		jobs = append(jobs, BackgroundJob{Name: "urtuu_relay", OK: true})
	}

	// The deletion sweep has no row of its own; what it leaves behind is the
	// organisations still counting down, which is the useful number anyway.
	var awaiting int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM tenants WHERE deletion_scheduled_at IS NOT NULL`).Scan(&awaiting); err == nil {
		jobs = append(jobs, BackgroundJob{
			Name: "deletion_sweep", OK: true, Pending: awaiting,
		})
	}
	return jobs
}

// TenantTrouble is an organisation having a bad day.
type TenantTrouble struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Failures int    `json:"failures"`
	Sample   string `json:"sample"`
}

// tenantTrouble is the per-organisation error view §E asks for.
//
// It comes from audit_events rather than from Prometheus, and that is a
// consequence of a decision made in the very first phase: **no tenant label on
// any metric**, because a label whose values are customers is a series count
// that only grows. The trade is that this question has to be answered from the
// database — which is a good answer anyway, since what an operator wants to
// know is "whose work is failing", and the audit trail records acts rather
// than requests.
func (s *Service) tenantTrouble(ctx context.Context) []TenantTrouble {
	rows, err := s.db.Query(scoped(ctx),
		`SELECT a.tenant_id::text, COALESCE(t.name, ''), count(*), min(a.action)
		   FROM audit_events a
		   LEFT JOIN tenants t ON t.id = a.tenant_id
		  WHERE a.created_at > NOW() - INTERVAL '24 hours'
		    AND a.tenant_id IS NOT NULL
		    AND (a.action LIKE '%fail%' OR a.action LIKE '%error%' OR a.action LIKE '%denied%')
		  GROUP BY a.tenant_id, t.name
		 HAVING count(*) >= 5
		  ORDER BY count(*) DESC
		  LIMIT 10`)
	if err != nil {
		slog.Warn("control plane: could not read the per-organisation failures", "error", err)
		return []TenantTrouble{}
	}
	defer rows.Close()

	trouble := make([]TenantTrouble, 0, 4)
	for rows.Next() {
		var row TenantTrouble
		if err := rows.Scan(&row.TenantID, &row.Name, &row.Failures, &row.Sample); err != nil {
			slog.Warn("control plane: could not read a failure row", "error", err)
			return trouble
		}
		trouble = append(trouble, row)
	}
	return trouble
}

// BackupStatus is what the console shows about the thing nobody thinks about
// until the morning they need it.
type BackupStatus struct {
	// Configured is false when nothing has ever reported a backup, which is
	// the state a deployment that never installed the cron job is in — and it
	// is shown as a warning rather than as an empty panel.
	Configured   bool       `json:"configured"`
	LastBackupAt *time.Time `json:"last_backup_at"`
	LastSizeMB   float64    `json:"last_size_mb"`
	LastOK       bool       `json:"last_ok"`
	LastDetail   string     `json:"last_detail"`
	// LastRestoreTestAt is recorded by hand. An untested backup is not a
	// backup, and the only way to know it has been tested is that somebody
	// says so.
	LastRestoreTestAt *time.Time `json:"last_restore_test_at"`
}

func (s *Service) backupStatus(ctx context.Context) BackupStatus {
	status := BackupStatus{}
	ctx = scoped(ctx)

	var size *int64
	err := s.db.QueryRow(ctx,
		`SELECT started_at, size_bytes, ok, detail FROM platform_backups
		  WHERE kind = 'backup' ORDER BY started_at DESC LIMIT 1`).
		Scan(&status.LastBackupAt, &size, &status.LastOK, &status.LastDetail)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return status
	case err != nil:
		slog.Warn("control plane: could not read the backup status", "error", err)
		return status
	}
	status.Configured = true
	if size != nil {
		status.LastSizeMB = float64(*size) / (1024 * 1024)
	}

	if err := s.db.QueryRow(ctx,
		`SELECT started_at FROM platform_backups
		  WHERE kind = 'restore_test' ORDER BY started_at DESC LIMIT 1`).
		Scan(&status.LastRestoreTestAt); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		slog.Warn("control plane: could not read the restore tests", "error", err)
	}
	return status
}

// RecordRestoreTest writes down that somebody restored a backup and it worked.
func (s *Service) RecordRestoreTest(ctx context.Context, sess Session, detail, reason string) error {
	return s.Do(ctx, sess, Change{
		Action:     "backup.restore_test",
		TargetType: "platform",
		TargetID:   "backups",
		Reason:     reason,
		After:      map[string]any{"detail": detail},
	}, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO platform_backups (kind, finished_at, ok, detail, recorded_by)
			 VALUES ('restore_test', NOW(), TRUE, $1, $2::uuid)`, detail, sess.ID)
		return err
	})
}

// CatalogStatus is where the app catalogue came from and what is installed.
type CatalogStatus struct {
	LastSyncAt *time.Time     `json:"last_sync_at"`
	OK         bool           `json:"ok"`
	Detail     string         `json:"detail"`
	Apps       []AppInstalled `json:"apps"`
}

// AppInstalled is one app and how its versions are spread across organisations.
type AppInstalled struct {
	AppID    string         `json:"app_id"`
	Name     string         `json:"name"`
	Versions map[string]int `json:"versions"`
	Total    int            `json:"total"`
}

func (s *Service) catalogStatus(ctx context.Context) CatalogStatus {
	status := CatalogStatus{OK: true, Apps: []AppInstalled{}}
	if s.catalogStatusFrom != nil {
		at, ok, detail := s.catalogStatusFrom()
		if !at.IsZero() {
			status.LastSyncAt = &at
		}
		status.OK, status.Detail = ok, detail
	}

	// The version spread is what says whether a release actually landed. One
	// organisation left on the previous version of an app is invisible
	// everywhere else — it looks like a working deployment until somebody
	// telephones about a feature that is missing.
	rows, err := s.db.Query(scoped(ctx),
		`SELECT i.app_id, COALESCE(a.name, i.app_id), i.installed_version, count(*)
		   FROM app_installations i
		   LEFT JOIN apps a ON a.id = i.app_id
		  WHERE i.enabled AND i.status = 'installed'
		  GROUP BY i.app_id, a.name, i.installed_version
		  ORDER BY i.app_id`)
	if err != nil {
		slog.Warn("control plane: could not read the installed versions", "error", err)
		return status
	}
	defer rows.Close()

	byApp := map[string]*AppInstalled{}
	for rows.Next() {
		var appID, name, version string
		var count int
		if err := rows.Scan(&appID, &name, &version, &count); err != nil {
			slog.Warn("control plane: could not read an installation row", "error", err)
			return status
		}
		app, known := byApp[appID]
		if !known {
			app = &AppInstalled{AppID: appID, Name: name, Versions: map[string]int{}}
			byApp[appID] = app
			status.Apps = append(status.Apps, AppInstalled{})
		}
		app.Versions[version] += count
		app.Total += count
	}

	status.Apps = status.Apps[:0]
	for _, app := range byApp {
		status.Apps = append(status.Apps, *app)
	}
	return status
}

// VersionInfo is what is actually running here.
type VersionInfo struct {
	Platform string `json:"platform"`
	Release  string `json:"release"`
	// Migration is the schema version the database is at, which is the number
	// that matters when a deployment half-landed.
	Migration int64      `json:"migration"`
	AppliedAt *time.Time `json:"migration_applied_at"`
}

func (s *Service) version(ctx context.Context) VersionInfo {
	info := VersionInfo{
		Platform: s.platformVersion,
		Release:  firstNonEmpty(os.Getenv("RELEASE_VERSION"), os.Getenv("IMAGE_TAG")),
	}
	// goose's own table. Read rather than joined into anything: it is the one
	// place that says which migrations this database has actually seen, and a
	// deployment whose image is newer than its schema is a real and quiet
	// failure mode.
	if err := s.db.QueryRow(scoped(ctx),
		`SELECT version_id, tstamp FROM goose_db_version
		  WHERE is_applied ORDER BY id DESC LIMIT 1`).
		Scan(&info.Migration, &info.AppliedAt); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		slog.Warn("control plane: could not read the migration version", "error", err)
	}
	return info
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// Deployment.
//
// The console asks GitHub to run the workflow this repository already has. It
// does not ship anything, does not touch the server, and cannot: the token it
// uses is a fine-grained one with permission for exactly this workflow, and
// the console never sees the machine it runs on.

var (
	// ErrDeployNotConfigured is a deployment with no token.
	ErrDeployNotConfigured = errors.New("this deployment has no GitHub token for the deploy workflow")
	// ErrDeployRefused is GitHub saying no.
	ErrDeployRefused = errors.New("GitHub refused to start the workflow")
)

// deployWorkflow is the file name in .github/workflows. Configurable, because a
// fork may have renamed it, and defaulted because most have not.
func deployWorkflow() string {
	return firstNonEmpty(os.Getenv("GITHUB_DEPLOY_WORKFLOW"), "deploy.yml")
}

// TriggerDeploy asks GitHub to run the deployment workflow at a ref.
//
// Returns the address of the workflow's own page, because the console
// deliberately does not follow the run: watching it would mean polling
// somebody else's API for minutes, and GitHub already has a screen for it that
// shows more than this console ever should.
func (s *Service) TriggerDeploy(ctx context.Context, sess Session, ref, reason string) (string, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_DEPLOY_TOKEN"))
	repository := strings.TrimSpace(os.Getenv("GITHUB_REPOSITORY"))
	if token == "" || repository == "" {
		return "", ErrDeployNotConfigured
	}
	if ref = strings.TrimSpace(ref); ref == "" {
		ref = "main"
	}

	runsURL := fmt.Sprintf("https://github.com/%s/actions/workflows/%s", repository, deployWorkflow())
	err := s.Do(ctx, sess, Change{
		Action:     "deploy.trigger",
		TargetType: "platform",
		TargetID:   repository,
		Reason:     reason,
		After:      map[string]any{"ref": ref, "workflow": deployWorkflow()},
	}, func(ctx context.Context, tx pgx.Tx) error {
		// Inside the transaction: a deployment that GitHub refused should not
		// leave an audit row saying it was triggered, and one that GitHub
		// accepted must not be missing from the trail because a commit failed
		// afterwards.
		return dispatchWorkflow(ctx, token, repository, deployWorkflow(), ref)
	})
	if err != nil {
		return "", err
	}
	slog.Warn("control plane: a deployment was triggered",
		"operator_email", sess.Email, "ref", ref, "repository", repository)
	return runsURL, nil
}

func dispatchWorkflow(ctx context.Context, token, repository, workflow, ref string) error {
	body, err := json.Marshal(map[string]string{"ref": ref})
	if err != nil {
		return err
	}
	address := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches",
		repository, workflow)

	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDeployRefused, err)
	}
	defer func() { _ = response.Body.Close() }()

	// 204 is what a dispatch answers with. Anything else is reported without
	// the response body, which on this endpoint can name the token.
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%w: %d", ErrDeployRefused, response.StatusCode)
	}
	return nil
}
