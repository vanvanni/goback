package app

import (
	"context"
	"os"
	"strings"

	"github.com/go-co-op/gocron/v2"
	"github.com/vanvanni/goback/internal/engines"
	"github.com/vanvanni/goback/internal/logging"
	"github.com/vanvanni/goback/internal/spec"
)

type App struct {
	scheduler     gocron.Scheduler
	repositories  map[string]*engines.S3
	mariadbConfig engines.MariaDBDumpConfig
	engines       map[string]engines.Engine
	docker        *engines.Docker
	dump          *engines.Dump
	backups       []BackupTask
}

var a *App

func A() *App {
	return a
}

func CreateApp() *App {
	s, err := gocron.NewScheduler()
	if err != nil {
		logging.Log.Error().Err(err).Msg("Failed to start scheduler")
		os.Exit(1)
	}

	dockerEngine, err := engines.NewDocker()
	if err != nil {
		logging.Log.Warn().Err(err).Msg("Could not initialize Docker engine, Docker operations will not be available")
	}

	dumpEngine, err := engines.NewDump()
	if err != nil {
		logging.Log.Warn().Err(err).Msg("Could not initialize Dump engine, database dump operations will not be available")
	}

	a = &App{
		scheduler:    s,
		repositories: make(map[string]*engines.S3),
		engines:      make(map[string]engines.Engine),
		docker:       dockerEngine,
		dump:         dumpEngine,
		backups:      []BackupTask{},
	}

	if dockerEngine != nil {
		a.engines["docker:default"] = dockerEngine
	}
	if dumpEngine != nil {
		a.engines["dump:default"] = dumpEngine
	}

	return A()
}

func (a *App) Start() {
	a.scheduler.Start()
}

func (a *App) Shutdown() {
	err := a.scheduler.Shutdown()
	if err != nil {
		logging.Log.Error().Err(err).Msg("Failed to shutdown scheduler")
	}
}

func (a *App) RegisterConf(conf *spec.CoreConfig) {
	logging.Log.Info().Msg("Loading main configuration")
	ctx := context.Background()
	for name, s3 := range conf.S3 {
		engine, err := engines.NewClient(ctx, s3)
		if err != nil {
			logging.Log.Error().Err(err).Str("s3", name).Msg("Could not create S3 connection")
			continue
		}
		a.repositories[name] = engine
		a.engines["s3:"+name] = engine
		logging.Log.Info().Str("s3", name).Msg("S3 repository added")
	}

	a.mariadbConfig = conf.MariaDB
	logging.Log.Info().Msg("MariaDB configuration added")
}

func (a *App) RegisterDef(definition *spec.BackupDefinition) {
	logging.Log.Info().Str("spec", definition.FileName).Msg("Loading specification")

	// Validate repositories exist
	for _, repo := range definition.Repositories {
		sources := strings.Split(repo.Source, ":")
		if len(sources) != 2 {
			logging.Log.Warn().Str("spec", definition.FileName).Str("source", repo.Source).Msg("Repository is invalid")
			return
		}

		engineKind := engines.EngineKind(sources[0])
		if !engineKind.IsStorageEngine() {
			logging.Log.Warn().Str("spec", definition.FileName).Str("source", sources[0]).Msg("Repository source is not a valid storage engine")
			return
		}

		sourceName := sources[1]
		_, ok := a.repositories[sourceName]
		if !ok {
			logging.Log.Warn().Str("spec", definition.FileName).Str("source", sourceName).Msg("Repository source connection does not exist")
			return
		}
	}

	backupTask := BackupTask{
		Definition:    definition,
		MariaDB:       definition.MariaDB,
		Directory:     definition.Directory,
		Volumes:       definition.Volumes,
		Repositories:  definition.Repositories,
		Retention:     definition.Retention,
		MariaDBConfig: a.mariadbConfig,
		docker:        a.docker,
		dump:          a.dump,
		engines:       a.engines,
	}

	_, err := a.scheduler.NewJob(
		definition.ScheduleTimer,
		gocron.NewTask(func(task BackupTask) {
			logging.Log.Info().Str("spec", task.Definition.FileName).Msg("Starting backup task")
			ctx := context.Background()
			archive, err := task.Run(ctx)
			if err != nil {
				logging.Log.Error().Err(err).Str("spec", task.Definition.FileName).Msg("Backup failed")
				return
			}
			logging.Log.Info().Str("spec", task.Definition.FileName).Str("archive", archive).Msg("Backup finished successfully")
		}, backupTask),
	)

	if err != nil {
		logging.Log.Error().Err(err).Str("spec", definition.FileName).Msg("Failed to create scheduled job")
		return
	}

	a.backups = append(a.backups, backupTask)
	logging.Log.Info().Str("spec", definition.FileName).Msg("Backup task registered successfully")
}
