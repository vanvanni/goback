package engines

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/vanvanni/goback/internal/helper"
)

type Dump struct {
	mariadbDumpPath   string
	mariadbClientPath string
}

var mariaDBIdentifierPattern = regexp.MustCompile(`^[a-zA-Z0-9_$-]+$`)

func NewDump() (*Dump, error) {
	path, err := exec.LookPath("mariadb-dump")
	if err != nil {
		path, err = exec.LookPath("mysqldump")
		if err != nil {
			return nil, fmt.Errorf("mariadb-dump or mysqldump not found in PATH")
		}
	}

	clientPath, _ := resolveMariaDBClientPath()

	return &Dump{
		mariadbDumpPath:   path,
		mariadbClientPath: clientPath,
	}, nil
}

func (d *Dump) Kind() EngineKind {
	return EngineKindDump
}

func (d *Dump) DumpMariaDB(ctx context.Context, config MariaDBDumpConfig) (*exec.Cmd, error) {
	if err := validateMariaDBDatabase(config.Database); err != nil {
		return nil, err
	}

	args := buildMariaDBConnectionArgs(config)
	args = append(args, "--", strings.TrimSpace(config.Database))
	// #nosec G204 -- command path is discovered via LookPath, input is validated, and args are passed without a shell.
	cmd := exec.CommandContext(ctx, d.mariadbDumpPath, args...)

	if config.OutputFile != "" {
		file, err := helper.OpenWriteOnlyFile(config.OutputFile, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		cmd.Stdout = file
	}

	return cmd, nil
}

func (d *Dump) RestoreMariaDB(ctx context.Context, config MariaDBDumpConfig, inputFile string) (*exec.Cmd, error) {
	if err := validateMariaDBDatabase(config.Database); err != nil {
		return nil, err
	}
	if strings.TrimSpace(inputFile) == "" {
		return nil, fmt.Errorf("input file is required")
	}

	clientPath, err := d.resolveClientPath()
	if err != nil {
		return nil, err
	}

	in, err := helper.OpenReadOnlyFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %w", err)
	}

	args := buildMariaDBConnectionArgs(config)
	args = append(args, "--", strings.TrimSpace(config.Database))
	// #nosec G204 -- command path is discovered via LookPath, input is validated, and args are passed without a shell.
	cmd := exec.CommandContext(ctx, clientPath, args...)
	cmd.Stdin = in
	return cmd, nil
}

func (d *Dump) CreateMariaDB(ctx context.Context, config MariaDBDumpConfig) (*exec.Cmd, error) {
	if err := validateMariaDBDatabase(config.Database); err != nil {
		return nil, err
	}
	return d.mariaDBExecCommand(ctx, config, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", strings.TrimSpace(config.Database)))
}

func (d *Dump) ReplaceMariaDB(ctx context.Context, config MariaDBDumpConfig) (*exec.Cmd, error) {
	if err := validateMariaDBDatabase(config.Database); err != nil {
		return nil, err
	}
	database := strings.TrimSpace(config.Database)
	return d.mariaDBExecCommand(ctx, config, fmt.Sprintf("DROP DATABASE IF EXISTS `%s`; CREATE DATABASE `%s`", database, database))
}

func (d *Dump) mariaDBExecCommand(ctx context.Context, config MariaDBDumpConfig, query string) (*exec.Cmd, error) {
	clientPath, err := d.resolveClientPath()
	if err != nil {
		return nil, err
	}

	args := buildMariaDBConnectionArgs(config)
	args = append(args, "-e", query)
	// #nosec G204 -- command path is discovered via LookPath, input is validated, and args are passed without a shell.
	return exec.CommandContext(ctx, clientPath, args...), nil
}

func (d *Dump) resolveClientPath() (string, error) {
	if d.mariadbClientPath != "" {
		return d.mariadbClientPath, nil
	}

	path, err := resolveMariaDBClientPath()
	if err != nil {
		return "", err
	}
	d.mariadbClientPath = path
	return d.mariadbClientPath, nil
}

func resolveMariaDBClientPath() (string, error) {
	path, err := exec.LookPath("mariadb")
	if err == nil {
		return path, nil
	}

	path, err = exec.LookPath("mysql")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("mariadb or mysql not found in PATH")
}

func buildMariaDBConnectionArgs(config MariaDBDumpConfig) []string {
	args := []string{}

	if config.Host != "" {
		args = append(args, "-h", config.Host)
	}

	if config.Port != "" {
		args = append(args, "-P", config.Port)
	}

	if config.User != "" {
		args = append(args, "-u", config.User)
	}

	if config.Password != "" {
		args = append(args, "-p"+config.Password)
	}

	return args
}

func validateMariaDBDatabase(database string) error {
	database = strings.TrimSpace(database)
	if database == "" {
		return fmt.Errorf("database is required")
	}
	if strings.HasPrefix(database, "-") {
		return fmt.Errorf("invalid database name")
	}
	if !mariaDBIdentifierPattern.MatchString(database) {
		return fmt.Errorf("invalid database name")
	}
	return nil
}

type MariaDBDumpConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	Database   string
	OutputFile string
}
