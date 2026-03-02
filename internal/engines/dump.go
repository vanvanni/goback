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
	mariadbDumpPath string
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

	return &Dump{
		mariadbDumpPath: path,
	}, nil
}

func (d *Dump) Kind() EngineKind {
	return EngineKindDump
}

func (d *Dump) DumpMariaDB(ctx context.Context, config MariaDBDumpConfig) (*exec.Cmd, error) {
	database := strings.TrimSpace(config.Database)
	if database == "" {
		return nil, fmt.Errorf("database is required")
	}
	if strings.HasPrefix(database, "-") {
		return nil, fmt.Errorf("invalid database name")
	}
	if !mariaDBIdentifierPattern.MatchString(database) {
		return nil, fmt.Errorf("invalid database name")
	}

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

	args = append(args, "--", database)
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

type MariaDBDumpConfig struct {
	Host       string
	Port       string
	User       string
	Password   string
	Database   string
	OutputFile string
}
