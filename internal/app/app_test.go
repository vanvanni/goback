package app

import (
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/spec"
)

func newTestApp(t *testing.T) *App {
	t.Helper()

	s, err := gocron.NewScheduler()
	if err != nil {
		t.Fatalf("failed to create scheduler: %v", err)
	}

	return &App{
		scheduler:    s,
		repositories: map[string]*engines.S3{},
		engines:      map[string]engines.Engine{},
		backups:      []BackupTask{},
	}
}

func newDefinition(source string) *spec.BackupDefinition {
	return &spec.BackupDefinition{
		FileName: "daily.yml",
		Repositories: []spec.RepositorySpec{
			{Source: source, Dest: "backups"},
		},
		ScheduleTimer: gocron.DurationJob(1 * time.Hour),
	}
}

func TestRegisterDefRejectsInvalidSourceFormat(t *testing.T) {
	a := newTestApp(t)
	defer a.Shutdown()

	a.RegisterDef(newDefinition("invalid"))
	if len(a.backups) != 0 {
		t.Fatalf("expected no backups to be registered, got %d", len(a.backups))
	}
}

func TestRegisterDefRejectsNonStorageEngine(t *testing.T) {
	a := newTestApp(t)
	defer a.Shutdown()

	a.RegisterDef(newDefinition("docker:default"))
	if len(a.backups) != 0 {
		t.Fatalf("expected no backups to be registered, got %d", len(a.backups))
	}
}

func TestRegisterDefRejectsMissingRepositoryConnection(t *testing.T) {
	a := newTestApp(t)
	defer a.Shutdown()

	a.RegisterDef(newDefinition("s3:missing"))
	if len(a.backups) != 0 {
		t.Fatalf("expected no backups to be registered, got %d", len(a.backups))
	}
}

func TestRegisterDefRegistersValidDefinition(t *testing.T) {
	a := newTestApp(t)
	defer a.Shutdown()

	a.globalEncryptionKey = "global-key"
	a.repositories["main"] = nil

	def := newDefinition("s3:main")
	a.RegisterDef(def)

	if len(a.backups) != 1 {
		t.Fatalf("expected one backup task to be registered, got %d", len(a.backups))
	}
	if a.backups[0].Definition != def {
		t.Fatal("registered backup definition pointer mismatch")
	}
	if a.backups[0].GlobalEncryptionKey != "global-key" {
		t.Fatalf("expected global encryption key %q, got %q", "global-key", a.backups[0].GlobalEncryptionKey)
	}
}

func TestRegisterConfSetsMariaAndGlobalKey(t *testing.T) {
	a := newTestApp(t)
	defer a.Shutdown()

	conf := &spec.CoreConfig{
		S3: map[string]engines.S3Config{},
		MariaDB: engines.MariaDBDumpConfig{
			Host: "db.internal",
		},
		EncryptionKey: "key-from-config",
	}

	a.RegisterConf(conf)

	if a.mariadbConfig.Host != "db.internal" {
		t.Fatalf("expected mariadb host %q, got %q", "db.internal", a.mariadbConfig.Host)
	}
	if a.globalEncryptionKey != "key-from-config" {
		t.Fatalf("expected global key %q, got %q", "key-from-config", a.globalEncryptionKey)
	}
}
