package internal

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	logFile     *os.File
	logFilename string
	logPath     string

	setupOnce   sync.Once
	multiWriter io.Writer

	loggerOnce sync.Once
	logger     *slog.Logger
	levelVar   *slog.LevelVar

	internalLoggerOnce sync.Once
	internalLogger     *slog.Logger
	internalLevelVar   *slog.LevelVar
)

// SetLogFilename sets the log filename (legacy).
// Deprecated: Use SetLogPath instead for full path support.
func SetLogFilename(filename string) {
	logFilename = filename
}

// SetLogPath sets the full path for the log file, including filename.
// Creates all necessary parent directories.
func SetLogPath(path string) {
	logPath = path
}

func setup() {
	setupOnce.Do(func() {
		var targetPath string

		if logPath != "" {
			// New behavior: use full path, create parent directories
			targetPath = logPath
			dir := filepath.Dir(targetPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				multiWriter = os.Stdout
				return
			}
		} else {
			// Legacy behavior: use "logs" directory with filename
			if err := os.MkdirAll("logs", 0755); err != nil {
				multiWriter = os.Stdout
				return
			}
			filename := logFilename
			if filename == "" {
				filename = "app.log"
			}
			targetPath = filepath.Join("logs", filename)
		}

		var err error
		logFile, err = os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Can't open log file, fall back to console-only
			multiWriter = os.Stdout
			return
		}

		multiWriter = io.MultiWriter(os.Stdout, logFile)
	})
}

func GetLogger() *slog.Logger {
	loggerOnce.Do(func() {
		levelVar = &slog.LevelVar{}

		setup()

		handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
			Level:     levelVar,
			AddSource: false,
		})
		logger = slog.New(handler)
	})
	return logger
}

func GetInternalLogger() *slog.Logger {
	internalLoggerOnce.Do(func() {
		internalLevelVar = &slog.LevelVar{}

		setup()

		handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
			Level:     internalLevelVar,
			AddSource: false,
		})
		internalLogger = slog.New(handler)
	})
	return internalLogger
}

func SetLogLevel(level slog.Level) {
	GetLogger()
	levelVar.Set(level)
}

func SetInternalLogLevel(level slog.Level) {
	GetInternalLogger()
	internalLevelVar.Set(level)
}

func SetRawLogLevel(rawLevel string) {
	var level slog.Level

	switch strings.ToLower(rawLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	GetLogger()
	levelVar.Set(level)
}

func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}
