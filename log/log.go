package log

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

const DefaultLogName = "damon.log"
const DefaultDamonNomadLogSuffix = ".damon.log"

type LogConfig struct {
	MaxLogFiles    int
	MaxSizeMB      int
	LogDir         string
	LogName        string
	NomadAllocDir  string
	NomadTaskName  string
	NomadLogSuffix string
}

func (c LogConfig) Dir() (string, error) {
	if c.LogDir != "" {
		return c.LogDir, nil
	}
	if c.NomadAllocDir != "" {
		return filepath.Join(c.NomadAllocDir, "logs"), nil
	}
	return os.Getwd()
}

func (c LogConfig) Name() string {
	if c.LogName != "" {
		return c.LogName
	}
	if c.NomadTaskName != "" {
		if c.NomadLogSuffix != "" {
			return fmt.Sprintf("%s%s", c.NomadTaskName, c.NomadLogSuffix)
		}
		return fmt.Sprintf("%s%s", c.NomadTaskName, DefaultDamonNomadLogSuffix)
	}
	return DefaultLogName
}

func (c LogConfig) Path() (string, error) {
	dir, err := c.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, c.Name()), nil
}

type Logger struct {
	zl zerolog.Logger
}

func (l Logger) WithFields(fs map[string]interface{}) Logger {
	return Logger{
		zl: l.zl.With().Fields(fs).Logger(),
	}
}

func (l Logger) Logln(v ...interface{}) {
	l.zl.Info().Msg(fmt.Sprint(v...))
}

func (l Logger) Logf(format string, v ...interface{}) {
	l.zl.Info().Msgf(format, v...)
}

func (l Logger) Error(err error, msg string) {
	if err == nil {
		return
	}
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}
	if v, ok := err.(stackTracer); ok {
		var stacktrace []string
		for _, frame := range v.StackTrace() {
			stacktrace = append(stacktrace, fmt.Sprintf("%+v", frame))
		}
		logger := l.zl.With().Fields(map[string]interface{}{
			"stacktrace": stacktrace,
		}).Logger()
		logger.Error().Err(err).Msg(msg)
	} else {
		l.zl.Error().Err(err).Msg(msg)
	}
}

func NewLogger(cfg LogConfig) (Logger, error) {
	filename, err := cfg.Path()
	if err != nil {
		return Logger{}, errors.Wrapf(err, "unable to get log directory")
	}
	logOut := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxLogFiles,
	}
	logger := zerolog.New(logOut).With().Timestamp().Logger()
	return Logger{
		zl: logger,
	}, nil
}
