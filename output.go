package siglog

import (
	"os"
)

type Output int

const (
	OUT_STDOUT Output = iota
	OUT_STDERR
	OUT_FILE
)

var outName = map[Output]string{
	OUT_STDOUT: "STDOUT",
	OUT_STDERR: "STDERR",
	OUT_FILE:   "FILE",
}

var outValue = map[string]Output{
	"STDOUT": OUT_STDOUT,
	"STDERR": OUT_STDERR,
	"FILE":   OUT_FILE,
}

func (o Output) String() string {
	return outName[o]
}

const (
	ENV_SL_OUTPUT = "ENV_SL_OUTPUT"
)

func GetOutput() Output {
	token := os.Getenv(ENV_SL_OUTPUT)
	if token == "" {
		SetOutput(OUT_STDOUT)
		return OUT_STDOUT
	}

	return outValue[token]
}

func SetOutput(o Output) error {
	return os.Setenv(ENV_SL_OUTPUT, o.String())
}
