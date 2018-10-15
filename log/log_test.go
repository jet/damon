package log

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLogConfig(t *testing.T) {
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd error: %v", err)
	}
	suffix := ".log"
	logName := "damon-static.log"
	taskName := "damon-task"
	damonDir := `c:\damon\logs`
	allocDir := `c:\nomad\alloc\c6f9c416-6d1b-4e76-94dc-724d1cd5134a\alloc`
	allocLogsDir := `c:\nomad\alloc\c6f9c416-6d1b-4e76-94dc-724d1cd5134a\alloc\logs`
	tests := []struct {
		cfg          LogConfig
		expectedDir  string
		expectedFile string
	}{
		{
			cfg:          LogConfig{},
			expectedDir:  workingDir,
			expectedFile: filepath.Join(workingDir, DefaultLogName),
		},
		{
			cfg:          LogConfig{LogDir: damonDir},
			expectedDir:  damonDir,
			expectedFile: filepath.Join(damonDir, DefaultLogName),
		},
		{
			cfg:          LogConfig{LogDir: damonDir, NomadAllocDir: allocDir},
			expectedDir:  damonDir,
			expectedFile: filepath.Join(damonDir, DefaultLogName),
		},
		{
			cfg:          LogConfig{NomadAllocDir: allocDir},
			expectedDir:  allocLogsDir,
			expectedFile: filepath.Join(allocLogsDir, DefaultLogName),
		},
		{
			cfg:          LogConfig{NomadAllocDir: allocDir, NomadTaskName: taskName},
			expectedDir:  allocLogsDir,
			expectedFile: filepath.Join(allocLogsDir, fmt.Sprintf("%s%s", taskName, DefaultDamonNomadLogSuffix)),
		},
		{
			cfg:          LogConfig{NomadAllocDir: allocDir, NomadTaskName: taskName, NomadLogSuffix: suffix},
			expectedDir:  allocLogsDir,
			expectedFile: filepath.Join(allocLogsDir, fmt.Sprintf("%s%s", taskName, suffix)),
		},
		{
			cfg:          LogConfig{LogName: logName},
			expectedDir:  workingDir,
			expectedFile: filepath.Join(workingDir, logName),
		},
		{
			cfg:          LogConfig{LogName: logName, LogDir: damonDir},
			expectedDir:  damonDir,
			expectedFile: filepath.Join(damonDir, logName),
		},
		{
			cfg:          LogConfig{LogName: logName, LogDir: damonDir, NomadAllocDir: allocDir},
			expectedDir:  damonDir,
			expectedFile: filepath.Join(damonDir, logName),
		},
		{
			cfg:          LogConfig{LogName: logName, NomadAllocDir: allocDir},
			expectedDir:  allocLogsDir,
			expectedFile: filepath.Join(allocLogsDir, logName),
		},
		{
			cfg:          LogConfig{LogName: logName, NomadAllocDir: allocDir, NomadTaskName: taskName},
			expectedDir:  allocLogsDir,
			expectedFile: filepath.Join(allocLogsDir, logName),
		},
		{
			cfg:          LogConfig{LogName: logName, NomadAllocDir: allocDir, NomadTaskName: taskName},
			expectedDir:  allocLogsDir,
			expectedFile: filepath.Join(allocLogsDir, logName),
		},
		{
			cfg:          LogConfig{LogName: logName, NomadAllocDir: allocDir, NomadTaskName: taskName, NomadLogSuffix: suffix},
			expectedDir:  allocLogsDir,
			expectedFile: filepath.Join(allocLogsDir, logName),
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Logf("%#v", test.cfg)
			dir, err := test.cfg.Dir()
			if err != nil {
				t.Fatal(err)
			}
			path, err := test.cfg.Path()
			if err != nil {
				t.Fatal(err)
			}
			if dir != test.expectedDir {
				t.Errorf("directory: expected %s but got %s", test.expectedDir, dir)
			}
			if path != test.expectedFile {
				t.Errorf("path: expected %s but got %s", test.expectedFile, path)
			}
		})
	}
}
