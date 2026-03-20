package engines

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewDumpFindsMariaDBDumpBinary(t *testing.T) {
	binDir := t.TempDir()
	createFakeBinary(t, binDir, "mariadb-dump")
	t.Setenv("PATH", binDir)

	dump, err := NewDump()
	if err != nil {
		t.Fatalf("expected NewDump to succeed, got error: %v", err)
	}

	got := strings.ToLower(filepath.Base(dump.mariadbDumpPath))
	if !strings.Contains(got, "mariadb-dump") {
		t.Fatalf("expected mariadb-dump binary path, got %q", dump.mariadbDumpPath)
	}
}

func TestNewDumpFallsBackToMysqldump(t *testing.T) {
	binDir := t.TempDir()
	createFakeBinary(t, binDir, "mysqldump")
	t.Setenv("PATH", binDir)

	dump, err := NewDump()
	if err != nil {
		t.Fatalf("expected NewDump fallback to mysqldump, got error: %v", err)
	}

	got := strings.ToLower(filepath.Base(dump.mariadbDumpPath))
	if !strings.Contains(got, "mysqldump") {
		t.Fatalf("expected mysqldump binary path, got %q", dump.mariadbDumpPath)
	}
}

func TestResolveMariaDBClientPathFindsMariaDB(t *testing.T) {
	binDir := t.TempDir()
	createFakeBinary(t, binDir, "mariadb")
	t.Setenv("PATH", binDir)

	path, err := resolveMariaDBClientPath()
	if err != nil {
		t.Fatalf("expected resolveMariaDBClientPath to succeed, got error: %v", err)
	}
	if got := strings.ToLower(filepath.Base(path)); !strings.Contains(got, "mariadb") {
		t.Fatalf("expected mariadb binary path, got %q", path)
	}
}

func TestResolveMariaDBClientPathFallsBackToMySQL(t *testing.T) {
	binDir := t.TempDir()
	createFakeBinary(t, binDir, "mysql")
	t.Setenv("PATH", binDir)

	path, err := resolveMariaDBClientPath()
	if err != nil {
		t.Fatalf("expected resolveMariaDBClientPath to succeed, got error: %v", err)
	}
	if got := strings.ToLower(filepath.Base(path)); !strings.Contains(got, "mysql") {
		t.Fatalf("expected mysql binary path, got %q", path)
	}
}

func TestNewDumpReturnsErrorWhenNoBinaryExists(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := NewDump()
	if err == nil {
		t.Fatal("expected error when no dump binary exists")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDumpKind(t *testing.T) {
	if got := (&Dump{}).Kind(); got != EngineKindDump {
		t.Fatalf("expected kind %q, got %q", EngineKindDump, got)
	}
}

func TestDumpMariaDBValidatesDatabase(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"-invalid",
		"bad name",
		"bad/name",
	}

	for _, db := range tests {
		t.Run(db, func(t *testing.T) {
			dump := &Dump{mariadbDumpPath: "mysqldump"}
			cmd, err := dump.DumpMariaDB(context.Background(), MariaDBDumpConfig{
				Database: db,
			})
			if err == nil {
				t.Fatalf("expected validation error for database %q", db)
			}
			if cmd != nil {
				t.Fatalf("expected nil command for invalid database %q", db)
			}
		})
	}
}

func TestDumpMariaDBBuildsCommandAndOutputFile(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "dump.sql")

	dump := &Dump{mariadbDumpPath: "mysqldump"}
	cmd, err := dump.DumpMariaDB(context.Background(), MariaDBDumpConfig{
		Host:       "db.local",
		Port:       "3306",
		User:       "alice",
		Password:   "secret",
		Database:   "app_db",
		OutputFile: outputPath,
	})
	if err != nil {
		t.Fatalf("DumpMariaDB returned error: %v", err)
	}

	wantArgs := []string{"mysqldump", "-h", "db.local", "-P", "3306", "-u", "alice", "-psecret", "--", "app_db"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("unexpected args length: got %d want %d (%v)", len(cmd.Args), len(wantArgs), cmd.Args)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("unexpected arg at index %d: got %q want %q", i, cmd.Args[i], wantArgs[i])
		}
	}

	if cmd.Stdout == nil {
		t.Fatal("expected stdout to be redirected to output file")
	}

	if closer, ok := cmd.Stdout.(interface{ Close() error }); ok {
		_ = closer.Close()
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestDumpMariaDBOmitsOptionalConnectionArgs(t *testing.T) {
	dump := &Dump{mariadbDumpPath: "mysqldump"}
	cmd, err := dump.DumpMariaDB(context.Background(), MariaDBDumpConfig{
		Database: "only_db",
	})
	if err != nil {
		t.Fatalf("DumpMariaDB returned error: %v", err)
	}

	wantArgs := []string{"mysqldump", "--", "only_db"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("unexpected args length: got %d want %d (%v)", len(cmd.Args), len(wantArgs), cmd.Args)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("unexpected arg at index %d: got %q want %q", i, cmd.Args[i], wantArgs[i])
		}
	}
}

func TestDumpMariaDBReturnsErrorForInvalidOutputPath(t *testing.T) {
	badOutputPath := filepath.Join(t.TempDir(), "missing", "dump.sql")
	dump := &Dump{mariadbDumpPath: "mysqldump"}

	_, err := dump.DumpMariaDB(context.Background(), MariaDBDumpConfig{
		Database:   "valid_db",
		OutputFile: badOutputPath,
	})
	if err == nil {
		t.Fatal("expected error when output path parent does not exist")
	}
	if !strings.Contains(err.Error(), "failed to create output file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRestoreMariaDBBuildsCommandAndInputFile(t *testing.T) {
	inputFile := filepath.Join(t.TempDir(), "dump.sql")
	if err := os.WriteFile(inputFile, []byte("select 1;"), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	dump := &Dump{mariadbClientPath: "mysql"}
	cmd, err := dump.RestoreMariaDB(context.Background(), MariaDBDumpConfig{
		Host:     "db.local",
		Port:     "3306",
		User:     "alice",
		Password: "secret",
		Database: "app_db",
	}, inputFile)
	if err != nil {
		t.Fatalf("RestoreMariaDB returned error: %v", err)
	}

	wantArgs := []string{"mysql", "-h", "db.local", "-P", "3306", "-u", "alice", "-psecret", "--", "app_db"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("unexpected args length: got %d want %d (%v)", len(cmd.Args), len(wantArgs), cmd.Args)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("unexpected arg at index %d: got %q want %q", i, cmd.Args[i], wantArgs[i])
		}
	}
	if cmd.Stdin == nil {
		t.Fatal("expected stdin to be redirected from input file")
	}
}

func TestReplaceMariaDBBuildsCommand(t *testing.T) {
	dump := &Dump{mariadbClientPath: "mysql"}
	cmd, err := dump.ReplaceMariaDB(context.Background(), MariaDBDumpConfig{
		Host:     "db.local",
		User:     "alice",
		Database: "app_db",
	})
	if err != nil {
		t.Fatalf("ReplaceMariaDB returned error: %v", err)
	}

	wantArgs := []string{"mysql", "-h", "db.local", "-u", "alice", "-e", "DROP DATABASE IF EXISTS `app_db`; CREATE DATABASE `app_db`"}
	if len(cmd.Args) != len(wantArgs) {
		t.Fatalf("unexpected args length: got %d want %d (%v)", len(cmd.Args), len(wantArgs), cmd.Args)
	}
	for i := range wantArgs {
		if cmd.Args[i] != wantArgs[i] {
			t.Fatalf("unexpected arg at index %d: got %q want %q", i, cmd.Args[i], wantArgs[i])
		}
	}
}

func TestS3Kind(t *testing.T) {
	if got := (&S3{}).Kind(); got != EngineKindS3 {
		t.Fatalf("expected kind %q, got %q", EngineKindS3, got)
	}
}

func TestS3UploadFileReturnsErrorWhenSourceMissing(t *testing.T) {
	client := &S3{}
	err := client.UploadFile(context.Background(), filepath.Join(t.TempDir(), "missing.tar.gz"), "backups")
	if err == nil {
		t.Fatal("expected error when source file does not exist")
	}
	if !strings.Contains(err.Error(), "failed to open file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestS3DownloadFileReturnsErrorWhenTargetCannotBeCreated(t *testing.T) {
	client := &S3{}
	err := client.DownloadFile(context.Background(), "backups/archive.tar.gz", filepath.Join(t.TempDir(), "missing", "archive.tar.gz"))
	if err == nil {
		t.Fatal("expected error when target file parent does not exist")
	}
	if !strings.Contains(err.Error(), "failed to create local file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestS3KeepMaxRejectsNegativeMax(t *testing.T) {
	client := &S3{}
	err := client.KeepMax(context.Background(), "backups", -1)
	if err == nil {
		t.Fatal("expected error when max is negative")
	}
	if !strings.Contains(err.Error(), "non-negative") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerKind(t *testing.T) {
	if got := (&Docker{}).Kind(); got != EngineKindDocker {
		t.Fatalf("expected kind %q, got %q", EngineKindDocker, got)
	}
}

func TestDockerWaitForPausedReturnsContextErrorWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (&Docker{}).WaitForPaused(ctx, "container-id")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if err != context.Canceled {
		t.Fatalf("expected %v, got %v", context.Canceled, err)
	}
}

func TestDockerWaitForAllPausedWithNoContainers(t *testing.T) {
	if err := (&Docker{}).WaitForAllPaused(context.Background(), nil); err != nil {
		t.Fatalf("expected nil error for empty container list, got: %v", err)
	}
}

func createFakeBinary(t *testing.T, dir, name string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := []byte("@echo off\r\nexit /b 0\r\n")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("failed to create fake binary %q: %v", path, err)
		}
		return path
	}

	path := filepath.Join(dir, name)
	content := []byte("#!/bin/sh\nexit 0\n")
	if err := os.WriteFile(path, content, 0755); err != nil {
		t.Fatalf("failed to create fake binary %q: %v", path, err)
	}
	return path
}
