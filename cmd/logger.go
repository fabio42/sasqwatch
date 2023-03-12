package cmd

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	logFile = "./saslwatch.log"
)

// LevelWriter interface
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

func setLogger(debug bool) error {
	var logWriter *os.File
	var err error
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)

		logWriter, err = os.OpenFile(
			logFile,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0664,
		)
		if err != nil {
			panic(err)
		}
	}

	fileWriter := zerolog.New(zerolog.ConsoleWriter{
		Out:          logWriter,
		NoColor:      true,
		PartsExclude: []string{"time", "level"},
	})
	consoleWriter := zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stderr
			w.PartsExclude = []string{"time"}
		},
	)
	consoleWriterLeveled := &LevelWriter{Writer: consoleWriter, Level: zerolog.ErrorLevel}
	log.Logger = zerolog.New(zerolog.MultiLevelWriter(fileWriter, consoleWriterLeveled)).With().Timestamp().Logger()
	return nil
}
