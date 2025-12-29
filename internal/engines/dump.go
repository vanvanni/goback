package engines

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type Dump struct {
	mariadbDumpPath string
}

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

	args = append(args, config.Database)
	cmd := exec.CommandContext(ctx, d.mariadbDumpPath, args...)

	if config.OutputFile != "" {
		file, err := os.Create(config.OutputFile)
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
