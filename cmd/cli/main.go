package main

import (
	"context"
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

		key, err := resolveDecryptKey(decryptKey, decryptDir, spec)
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

var recoverCmd = &cobra.Command{
	Use:     "recover <backup> <path>",
	Short:   "Download and extract a backup archive",
	Example: "goback recover backups/backup-daily-123.tar.gz.enc ./restore --spec daily.yml --dir /etc/goback",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		backup := args[0]
		outputPath := args[1]
		return runRecover(cmd.Context(), backup, outputPath)
	},
}

var baseDir string
var decryptDir string
var decryptKey string
var decryptOutput string
var recoverDir string
var recoverSpec string
var recoverKey string
var recoverSource string

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&baseDir, "dir", "d", "", "goback directory")
	_ = startCmd.MarkFlagRequired("dir")

	rootCmd.AddCommand(decryptCmd)
	decryptCmd.Flags().StringVarP(&decryptKey, "key", "k", "", "decryption key")
	decryptCmd.Flags().StringVarP(&decryptOutput, "output", "o", "", "decrypted output file")
	decryptCmd.Flags().StringVarP(&decryptDir, "dir", "d", "", "goback directory (used to resolve key from config/spec)")

	rootCmd.AddCommand(recoverCmd)
	recoverCmd.Flags().StringVarP(&recoverSpec, "spec", "s", "", "spec file used to resolve repository and key")
	recoverCmd.Flags().StringVarP(&recoverDir, "dir", "d", "", "goback directory")
	recoverCmd.Flags().StringVarP(&recoverKey, "key", "k", "", "decryption key")
	recoverCmd.Flags().StringVar(&recoverSource, "source", "", "repository source selector, for example s3:main")
	_ = recoverCmd.MarkFlagRequired("dir")
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

	conf, err := spec.LoadConfig(filepath.Join(dir, "config.yml"))
	if err != nil {
		return "", fmt.Errorf("could not load core configuration: %w", err)
	}

	resolvedKey := strings.TrimSpace(conf.EncryptionKey)
	if specArg != "" {
		specPath, err := resolveSpecPath(dir, specArg)
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

func runRecover(ctx context.Context, backup, outputPath string) error {
	if strings.TrimSpace(recoverDir) == "" {
		return fmt.Errorf("--dir is required")
	}
	if strings.TrimSpace(recoverSpec) == "" && strings.TrimSpace(recoverSource) == "" {
		return fmt.Errorf("recover without --spec requires --source")
	}

	conf, err := loadCoreConfig(recoverDir)
	if err != nil {
		return err
	}

	def, err := loadOptionalSpec(recoverDir, recoverSpec)
	if err != nil {
		return err
	}

	repository, err := resolveRecoverRepository(def, recoverSource)
	if err != nil {
		return err
	}

	recoveryRuntime, err := app.CreateRecoveryRuntime(ctx, conf)
	if err != nil {
		return err
	}

	downloader, err := resolveRecoveryDownloader(recoveryRuntime, repository.Source)
	if err != nil {
		return err
	}

	decryptionKey := ""
	if strings.HasSuffix(backup, ".enc") {
		decryptionKey, err = resolveDecryptKey(recoverKey, recoverDir, recoverSpec)
		if err != nil {
			return err
		}
	}

	task := &app.RecoveryTask{
		RemoteKey:     resolveRecoveryRemoteKey(repository, backup),
		OutputPath:    outputPath,
		DecryptionKey: decryptionKey,
		Downloader:    downloader,
	}

	_, err = task.Run(ctx)
	return err
}

func loadCoreConfig(dir string) (*spec.CoreConfig, error) {
	conf, err := spec.LoadConfig(filepath.Join(dir, "config.yml"))
	if err != nil {
		return nil, fmt.Errorf("could not load core configuration: %w", err)
	}
	return conf, nil
}

func loadOptionalSpec(dir, specArg string) (*spec.BackupDefinition, error) {
	if strings.TrimSpace(specArg) == "" {
		return nil, nil
	}

	specPath, err := resolveSpecPath(dir, specArg)
	if err != nil {
		return nil, err
	}

	def, err := spec.LoadSpec(specPath)
	if err != nil {
		return nil, fmt.Errorf("could not load spec %q: %w", specPath, err)
	}
	def.FileName = filepath.Base(specPath)
	return def, nil
}

func resolveRecoverRepository(def *spec.BackupDefinition, source string) (spec.RepositorySpec, error) {
	if def == nil {
		if strings.TrimSpace(source) == "" {
			return spec.RepositorySpec{}, fmt.Errorf("repository source is required")
		}
		return spec.RepositorySpec{Source: source}, nil
	}

	if len(def.Repositories) == 0 {
		return spec.RepositorySpec{}, fmt.Errorf("spec does not define any repositories")
	}
	if strings.TrimSpace(source) == "" {
		return def.Repositories[0], nil
	}

	for _, repo := range def.Repositories {
		if repo.Source == source {
			return repo, nil
		}
	}

	return spec.RepositorySpec{}, fmt.Errorf("repository source %q was not found in spec", source)
}

func resolveRecoveryDownloader(runtime *app.RecoveryRuntime, source string) (app.RecoveryDownloader, error) {
	parts := strings.Split(source, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repository source %q", source)
	}
	if parts[0] != "s3" {
		return nil, fmt.Errorf("unsupported repository source %q", source)
	}

	repo := runtime.Repositories[parts[1]]
	if repo == nil {
		return nil, fmt.Errorf("repository source connection %q does not exist", parts[1])
	}

	return repo, nil
}

func resolveRecoveryRemoteKey(repository spec.RepositorySpec, backup string) string {
	if repository.Dest == "" {
		return backup
	}
	if strings.Contains(backup, "/") || strings.Contains(backup, "\\") {
		return filepath.ToSlash(backup)
	}
	return filepath.ToSlash(filepath.Join(repository.Dest, backup))
}

func resolveSpecPath(baseDir, specArg string) (string, error) {
	specDir := filepath.Join(baseDir, "specs.d")
	candidates := []string{}
	if filepath.IsAbs(specArg) || strings.Contains(specArg, "/") || strings.Contains(specArg, "\\") {
		candidates = append(candidates, specArg)
	} else {
		candidates = append(candidates, filepath.Join(specDir, specArg))
		if filepath.Ext(specArg) == "" {
			candidates = append(candidates, filepath.Join(specDir, specArg+".yml"))
			candidates = append(candidates, filepath.Join(specDir, specArg+".yaml"))
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
