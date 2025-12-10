package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type Formatter string

const (
	Text Formatter = "text"
	Json Formatter = "json"
)

var Log = logrus.New()


func SetupLogger(level string, format Formatter) error {
	msg := "logger.setupLogger"

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("%s fail parse level string %s: %w", msg, level, err)
	}

	Log.SetLevel(logLevel)
	formatter, err := getFormatter(format)
	if err != nil {
		return fmt.Errorf("%s fail get formatter: %w", msg, err)
	}
	Log.SetFormatter(formatter)
	Log.SetOutput(os.Stdout)
	
	return nil
}

func getFormatter(format Formatter) (logrus.Formatter, error) {
	form := Formatter(strings.ToLower(string(format)))

	switch form {
	case Text:
		return &logrus.TextFormatter{
			FullTimestamp: true,
		}, nil
	case Json:
		return &logrus.JSONFormatter{}, nil
	default:
		// Фолбэк по умолчанию, если формат неизвестен
		return nil, fmt.Errorf("unknown formatter: %s", format)
	}
}


