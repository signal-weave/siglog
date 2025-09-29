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

### Example

```go
func main() {
    siglog.SetLogDirectory("some/directory")
    siglog.SetLogLevel(siglog.LL_WARN)
    siglog.SetOutput(siglog.OUT_STDOUT)

    entry := "I want to log this warning."
    siglog.LogEntry(entry, "main", siglog.LL_WARN)
}
```
