package siglog

import (
	"os"
)

type LogLevel int

const (
	LL_NONE LogLevel = iota
	LL_ERROR
	LL_WARN
	LL_INFO
	LL_DEBUG
)

var levelName = map[LogLevel]string{
	LL_NONE:  "NONE",
	LL_DEBUG: "DEBUG",
	LL_INFO:  "INFO",
	LL_WARN:  "WARN",
	LL_ERROR: "ERROR",
}

var levelValue = map[string]LogLevel{
	"NONE":  LL_NONE,
	"DEBUG": LL_DEBUG,
	"INFO":  LL_INFO,
	"WARN":  LL_WARN,
	"ERROR": LL_ERROR,
}

func (ll LogLevel) String() string {
	return levelName[ll]
}

const (
	ENV_SL_LOGGING_LEVEL = "ENV_SL_LOGGING_LEVEL"
)

func GetLogLevel() LogLevel {
	token := os.Getenv(ENV_SL_LOGGING_LEVEL)
	if token == "" {
		SetLogLevel(LL_NONE)
		return LL_NONE
	}

	return levelValue[token]
}

func SetLogLevel(l LogLevel) error {
	return os.Setenv(ENV_SL_LOGGING_LEVEL, l.String())
}
