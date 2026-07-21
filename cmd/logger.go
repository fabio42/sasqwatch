package cmd

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// logFileName is the debug log file name written under os.TempDir().
const logFileName = "sasqwatch.log"

// LevelWriter wraps an io.Writer and only forwards log entries at or above Level.
type LevelWriter struct {
	io.Writer
	Level zerolog.Level
}

func (lw *LevelWriter) WriteLevel(l zerolog.Level, p []byte) (n int, err error) {
	if l >= lw.Level {
		return lw.Write(p)
	}
	return len(p), nil
}

// setLogger initialises the global zerolog logger. When debug is true a debug
// log is written to os.TempDir()/sasqwatch.log; otherwise only error-and-above
// messages are forwarded to stderr.
func setLogger(debug bool) error {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	consoleWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.PartsExclude = []string{"time"}
	})
	consoleWriterLeveled := &LevelWriter{Writer: consoleWriter, Level: zerolog.ErrorLevel}

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)

		logPath := filepath.Join(os.TempDir(), logFileName)
		logWriter, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			return err
		}

		fileWriter := zerolog.New(zerolog.ConsoleWriter{
			Out:          logWriter,
			NoColor:      true,
			PartsExclude: []string{"time", "level"},
		})
		log.Logger = zerolog.New(zerolog.MultiLevelWriter(fileWriter, consoleWriterLeveled)).
			With().Timestamp().Logger()
		return nil
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(consoleWriterLeveled).With().Timestamp().Logger()
	return nil
}
