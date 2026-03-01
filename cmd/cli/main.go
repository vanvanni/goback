package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/vanvanni/goback/internal/app"
	"github.com/vanvanni/goback/internal/helper"
	"github.com/vanvanni/goback/internal/logging"
	"github.com/vanvanni/goback/internal/spec"
)

var rootCmd = &cobra.Command{
	Use:   "goback",
	Short: "Interact with the GoBack backup solution",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start backup daemon",
	Example: `  goback start
  goback start --dir /etc/goback`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return startApp(baseDir)
	},
}

var decryptCmd = &cobra.Command{
	Use:     "decrypt <archive> [spec]",
	Short:   "Decrypt an archive",
	Example: "goback decrypt data.enc",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		file := args[0]

		spec := ""
		if len(args) > 1 {
			spec = args[1]
		}

		key, err := resolveDecryptKey(decryptKey, baseDir, spec)
		if err != nil {
			return err
		}

		output := decryptOutput
		if output == "" {
			output = defaultDecryptOutput(file)
		}

		if err := helper.DecryptFile(file, output, key); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "decrypted: %s\n", output)
		return nil
	},
}

var baseDir string
var decryptKey string
var decryptOutput string

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&baseDir, "dir", "d", "", "goback directory")
	_ = startCmd.MarkFlagRequired("dir")

	rootCmd.AddCommand(decryptCmd)
	decryptCmd.Flags().StringVarP(&decryptKey, "key", "k", "", "decryption key")
	decryptCmd.Flags().StringVarP(&decryptOutput, "output", "o", "", "decrypted output file")
	decryptCmd.Flags().StringVarP(&baseDir, "dir", "d", "", "goback directory (used to resolve key from config/spec)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startApp(dir string) error {
	logging.Init(zerolog.DebugLevel, "console")

	backupApp := app.CreateApp()

	if err := spec.EnsureDirectoryStructure(dir); err != nil {
		return fmt.Errorf("could not ensure base structure: %w", err)
	}

	conf, err := spec.LoadConfig(spec.GetCoreConfigPath())
	if err != nil {
		return fmt.Errorf("could not load core configuration: %w", err)
	}

	defs, err := spec.LoadSpecs(spec.GetSpecDirectory())
	if err != nil {
		return fmt.Errorf("could not load specs: %w", err)
	}

	backupApp.RegisterConf(conf)
	for _, def := range defs {
		backupApp.RegisterDef(def)
	}

	backupApp.Start()
	logging.Log.Info().Msg("GoBack has been started")

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quitChannel)
	<-quitChannel

	backupApp.Shutdown()
	return nil
}

func resolveDecryptKey(key, dir, specArg string) (string, error) {
	if key != "" {
		return key, nil
	}

	if dir == "" {
		return "", fmt.Errorf("missing decryption key: pass --key or provide --dir with configured encryption_key")
	}

	if err := spec.EnsureDirectoryStructure(dir); err != nil {
		return "", fmt.Errorf("could not ensure base structure: %w", err)
	}

	conf, err := spec.LoadConfig(spec.GetCoreConfigPath())
	if err != nil {
		return "", fmt.Errorf("could not load core configuration: %w", err)
	}

	resolvedKey := strings.TrimSpace(conf.EncryptionKey)
	if specArg != "" {
		specPath, err := resolveSpecPath(specArg)
		if err != nil {
			return "", err
		}

		def, err := spec.LoadSpec(specPath)
		if err != nil {
			return "", fmt.Errorf("could not load spec %q: %w", specPath, err)
		}

		if strings.TrimSpace(def.EncryptionKey) != "" {
			resolvedKey = def.EncryptionKey
		}
	}

	if strings.TrimSpace(resolvedKey) == "" {
		return "", fmt.Errorf("no decryption key found: set --key or configure encryption_key")
	}

	return resolvedKey, nil
}

func resolveSpecPath(specArg string) (string, error) {
	candidates := []string{}
	if filepath.IsAbs(specArg) || strings.Contains(specArg, "/") || strings.Contains(specArg, "\\") {
		candidates = append(candidates, specArg)
	} else {
		candidates = append(candidates, filepath.Join(spec.GetSpecDirectory(), specArg))
		if filepath.Ext(specArg) == "" {
			candidates = append(candidates, filepath.Join(spec.GetSpecDirectory(), specArg+".yml"))
			candidates = append(candidates, filepath.Join(spec.GetSpecDirectory(), specArg+".yaml"))
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("spec %q was not found", specArg)
}

func defaultDecryptOutput(input string) string {
	if strings.HasSuffix(input, ".enc") {
		return strings.TrimSuffix(input, ".enc")
	}

	return input + ".dec"
}
