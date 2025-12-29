package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/vanvanni/goback/internal/app"
	"github.com/vanvanni/goback/internal/logging"
	"github.com/vanvanni/goback/internal/spec"
)

func main() {
	logging.Init(zerolog.DebugLevel, "console")

	var baseDir string
	flag.StringVar(&baseDir, "dir", "", "goback directory")
	flag.Parse()

	if baseDir == "" {
		logging.Log.Error().Msg("Missing goback directory path")
		os.Exit(1)
	}

	app := app.CreateApp()

	err := spec.EnsureDirectoryStructure(baseDir)
	if err != nil {
		logging.Log.Error().Err(err).Msg("Could not ensure base structure")
		os.Exit(1)
	}

	conf, err := spec.LoadConfig(spec.GetCoreConfigPath())
	if err != nil {
		logging.Log.Error().Err(err).Msg("Could not core configuration")
		os.Exit(1)
	}

	defs, err := spec.LoadSpecs(spec.GetSpecDirectory())
	if err != nil {
		logging.Log.Panic().Err(err).Msg("could not load specs")
	}

	app.RegisterConf(conf)
	for _, def := range defs {
		app.RegisterDef(def)
	}

	app.Start()

	logging.Log.Info().Msg("GoBack has been started")

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	app.Shutdown()
}
