# SigLog
Signal Weave Logging

This module is the standardized logging methodology and formatting used by all
Signal Weave applications.

## Usage

SigLog has 5 levels of logging:

* `LL_NONE`
* `LL_ERROR`
* `LL_WARN`
* `LL_INFO`
* `LL_DEBUG`

These are listed in in priority order. If the current log level is set to
LL_INFO, then all entries set for `LL_INFO`, `LL_WARN`, and `LL_ERROR` will be
logged.

SigLog currently supports 3 outputs:

* `OUT_STDOUT`
* `OUT_STDERR`
* `OUT_FILE`

The log files are dated and written to a specified directory using
`siglog.SetLogDirectory()`.

By default the logging level is set to `LL_NONE`, the output is set to
`OUT_STDOUT`, and the output directory is blank.

There are three supported batching methods:

* `BATCH_NONE`
* `BATCH_ITEM`
* `BATCH_BYTE`
* `BATCH_TIEM`

Where `BATCH_NONE` disables batching and resumes individual writes.

You can edit batch values with the following:

* `siglog.SetMaxItems()`
* `siglog.SetMaxBytes()`
* `siglog.SetMaxWait()`

The logger is asynchronus so it will not log all items unless either
`siglog.Flush()` is manually called, or `siglog.Shutdown()` is called in the
application shutdown process.

Logs are output as the following:

```
HH-MM-SS-XX: [Caller][Level] - Entry.
```

For example:

```
16-14-49-00: [SYSTEM][WARN] - I want to log this warning.
```

A custom formatter can be provided using `siglog.SetLogFormatter()`. The format
function must be a `siglog.LogFormatter`, which is
```go
type LogFormatter func(Entry) (string, error)
```

### Example

```go
package main

import (
	"fmt"
	"time"

	"github.com/SignalWeave/siglog"
)

func main() {
    siglog.SetLogDirectory("some/directory")
	siglog.SetLogFormatter(NewFormatter)

    siglog.SetLogLevel(siglog.LL_WARN)

    siglog.SetBatchMode(siglog.BATCH_ITEM)
    siglog.SetMaxItems(3)

    siglog.SetOutput(siglog.OUT_STDOUT)

    siglog.LogEntry("I want to log this warning.", "SYSTEM", siglog.LL_WARN)
    siglog.LogEntry("I want to log this error.", "SYSTEM", siglog.LL_ERROR)
    siglog.LogEntry("This should not show up.", "SYSTEM", siglog.LL_INFO)

    siglog.Shutdown()
}

// Formats to <Entry.Caller> - <Entry.Entry>
// e.g. "SYSTEM - hello world".
var NewFormatter siglog.LogFormatter = func(e siglog.Entry) (string, error) {
	out := fmt.Sprintf("%s - %s\n", e.Caller, e.Entry)
	return out, nil
}
```
