package siglog

import (
	"os"
)

type BatchMode int

const (
	BATCH_NONE BatchMode = iota
	BATCH_ITEM
	BATCH_BYTE
	BATCH_TIME
)

var batchName = map[BatchMode]string{
	BATCH_NONE: "NONE",
	BATCH_ITEM: "ITEM",
	BATCH_BYTE: "BYTE",
	BATCH_TIME: "TIME",
}

var batchValue = map[string]BatchMode{
	"NONE": BATCH_NONE,
	"ITEM": BATCH_ITEM,
	"BYTE": BATCH_BYTE,
	"TIME": BATCH_TIME,
}

func (b BatchMode) String() string {
	return batchName[b]
}

const (
	ENV_SL_BATCH = "ENV_SL_BATCH"
)

func GetBatchMode() BatchMode {
	token := os.Getenv(ENV_SL_BATCH)
	if token == "" {
		SetBatchMode(BATCH_NONE)
		return BATCH_NONE
	}

	return batchValue[token]
}

func SetBatchMode(b BatchMode) error {
	return os.Setenv(ENV_SL_BATCH, b.String())
}
