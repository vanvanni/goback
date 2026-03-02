package app

import (
	"testing"

	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/spec"
)

type fakeEngine struct {
	kind engines.EngineKind
}

func (f fakeEngine) Kind() engines.EngineKind {
	return f.kind
}

func TestCreateAppSetsGlobalInstance(t *testing.T) {
	app := CreateApp()
	if app == nil {
		t.Fatal("expected CreateApp to return an app instance")
	}
	defer app.Shutdown()

	if A() != app {
		t.Fatal("expected A() to return the app created by CreateApp")
	}

	if app.scheduler == nil {
		t.Fatal("expected scheduler to be initialized")
	}
	if app.repositories == nil {
		t.Fatal("expected repositories map to be initialized")
	}
	if app.engines == nil {
		t.Fatal("expected engines map to be initialized")
	}
}

func TestAppStartAndShutdown(t *testing.T) {
	app := newTestApp(t)
	app.Start()
	app.Shutdown()
}

func TestBackupTaskEngineAccessors(t *testing.T) {
	dockerEngine := &engines.Docker{}
	dumpEngine := &engines.Dump{}
	s3Engine := &engines.S3{}

	task := BackupTask{
		docker: dockerEngine,
		dump:   dumpEngine,
		engines: map[string]engines.Engine{
			"custom":       fakeEngine{kind: engines.EngineKind("custom")},
			"s3:repo":      s3Engine,
			"s3:not-s3":    fakeEngine{kind: engines.EngineKindS3},
			"dump:default": dumpEngine,
		},
	}

	if task.Docker() != dockerEngine {
		t.Fatal("Docker accessor returned unexpected engine")
	}
	if task.Dump() != dumpEngine {
		t.Fatal("Dump accessor returned unexpected engine")
	}
	if got := task.GetEngine("custom"); got == nil {
		t.Fatal("expected custom engine to be returned")
	}
	if got := task.GetEngine("missing"); got != nil {
		t.Fatalf("expected nil engine for missing key, got %#v", got)
	}
	if got := task.GetS3("repo"); got != s3Engine {
		t.Fatal("expected GetS3 to return matching S3 engine")
	}
	if got := task.GetS3("missing"); got != nil {
		t.Fatalf("expected nil from GetS3 for missing repository, got %#v", got)
	}
	if got := task.GetS3("not-s3"); got != nil {
		t.Fatalf("expected nil from GetS3 when mapped engine is not *engines.S3, got %#v", got)
	}
}

func TestRegisterConfRegistersS3EngineWhenConfigValid(t *testing.T) {
	app := newTestApp(t)
	defer app.Shutdown()

	conf := &spec.CoreConfig{
		S3: map[string]engines.S3Config{
			"main": {
				Bucket:            "bucket",
				Region:            "us-east-1",
				AccessKey:         "access",
				SecretKey:         "secret",
				Endpoint:          "http://127.0.0.1:9000",
				ForcePathStyle:    true,
				HostnameImmutable: true,
			},
		},
	}

	app.RegisterConf(conf)

	if _, ok := app.repositories["main"]; !ok {
		t.Fatal("expected S3 repository to be registered")
	}
	engine, ok := app.engines["s3:main"]
	if !ok {
		t.Fatal("expected S3 engine to be registered")
	}
	if _, ok := engine.(*engines.S3); !ok {
		t.Fatalf("expected registered engine to be *engines.S3, got %T", engine)
	}
}
