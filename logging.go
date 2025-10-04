package siglog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func init() {
	SetLogDirectory("")
	SetLogLevel(LL_NONE)
	SetOutput(OUT_STDOUT)
}

// -------Constants-------------------------------------------------------------

const (
	Developer = "Signal Weave"
)

const (
	DateLayout = "01-02-2006"  // MM-DD-YYYY
	TimeLayout = "15-04-05-00" // HH-MM-SS-XX
)

const ENV_SL_LOGDIR = "ENV_SL_LOGDIR"

const COULD_NOT_WRITE_ENTRY = "Could not write entry to log buffer."

// -------Log Directory---------------------------------------------------------

func getLogDirectory() string {
	return os.Getenv(ENV_SL_LOGDIR)
}

// SetLogDirectory sets the directory for log files to be written to.
// Will make directory and set corersponding environmnet variable.
func SetLogDirectory(p string) error {
	err := os.Setenv(ENV_SL_LOGDIR, p)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(getLogDirectory(), 0755); err != nil {
		return err
	}

	return nil
}

// -------Helpers/Formatters----------------------------------------------------

func getToday() string {
	return time.Now().Format(DateLayout)
}

func getTodaysLogFileName() string {
	filename := fmt.Sprintf(
		"mycelia-log-%s.log", getToday(),
	)
	return filename
}

func getTodaysLogFilePath() string {
	return filepath.Join(getLogDirectory(), getTodaysLogFileName())
}

// Returns a formatted entry based on the current logging level environment var.
func defaultFormatByLevel(le Entry) (string, error) {
	if len(le.Entry) == 0 || le.Entry[len(le.Entry)-1] != '\n' {
		le.Entry += "\n"
	}

	now := time.Now().Format(TimeLayout)
	tok := levelName[GetLogLevel()]
	out := fmt.Sprintf("%s: [%s][%s] - %s", now, le.Caller, tok, le.Entry)

	return out, nil
}

// A function type that takes an Entry struct and formats the values to create
// the log entry that will be written to the configured output.
type LogFormatter func(Entry) (string, error)

var (
	fmtMu         sync.RWMutex
	formatterFunc LogFormatter = defaultFormatByLevel
)

// SetLogFormatter replaces the global fallback formatter used.
// f is the function that will be called to format log Entry structs into the
// string entry that is written to the configured output.
func SetLogFormatter(f LogFormatter) {
	if f == nil {
		return
	}
	fmtMu.Lock()
	formatterFunc = f
	fmtMu.Unlock()
}

// formatByLevel calls the configured LogFormatter that will convert an Entry
// struct into the string message that will be written to the configured output.
//
// This is a wrapper to separate the configured function with the actual calls.
func formatByLevel(le Entry) (string, error) {
	return formatterFunc(le)
}

// -------Logger----------------------------------------------------------------

type Entry struct {
	// What "source" the log message is coming from.
	// This should be a file, struct, stream, etc.
	Caller string

	// The message to log.
	Entry string

	// What logging Level is required in the environment for the log to be
	// written.
	Level LogLevel
}

// logger writes to a dated log file.
// Use logging.GlobalLogger instead of making a new one.
type logger struct {
	file *os.File
	in   chan *Entry
	date string

	writer *bufio.Writer

	// Batching
	maxItems int
	maxBytes int
	batchBuf []string
	maxWait  time.Duration
	timer    *time.Timer

	wg sync.WaitGroup
}

func newLogger() *logger {
	file, err := os.OpenFile(
		getTodaysLogFilePath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
	)
	if err != nil {
		file = nil
	}

	l := &logger{
		file: file,
		date: getToday(),
		in:   make(chan *Entry, 1024),

		writer: bufio.NewWriter(file),

		maxItems: 128,
		maxBytes: 512,
		maxWait:  250 * time.Millisecond,
		batchBuf: []string{},
	}
	l.timer = time.NewTimer(l.maxWait)
	if !l.timer.Stop() {
		<-l.timer.C
	}
	l.start()

	return l
}

func (l *logger) start() { go l.loop() }

// Rotates the tracked log file to the new dated log file.
func (l *logger) rotate() {
	_ = (l.writer).Flush()
	_ = l.file.Close()

	f, err := os.OpenFile(
		getTodaysLogFilePath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not create next day log file.")
		return
	}
	l.file = f
	l.date = getToday()
	l.writer = bufio.NewWriter(l.file)
}

func (l *logger) loop() {
	defer func() {
		_ = l.file.Close()
	}()

	for {
		select {

		case entry, ok := <-l.in:
			if !ok {
				return
			}
			if entry == nil {
				l.wg.Done()
				continue
			}

			if GetLogLevel() == LL_NONE {
				continue
			}

			if entry.Level <= GetLogLevel() {
				msg, err := formatByLevel(*entry)
				if err != nil {
					msg = COULD_NOT_WRITE_ENTRY
				}
				l.writeToOut(msg)
			}

			if getToday() != l.date {
				l.rotate()
			}

			l.wg.Done()

		case <-l.timer.C:
			out := strings.Join(l.batchBuf, "")
			l.writeToOut(out)
			l.batchBuf = []string{}

			if l.maxWait > 0 {
				l.timer.Reset(l.maxWait)
			}

			l.wg.Done()
		}
	}
}

func (l *logger) writeToOut(out string) {
	switch GetOutput() {

	case OUT_STDOUT:
		w := bufio.NewWriter(os.Stdout)
		w.WriteString(out)
		w.Flush()

	case OUT_STDERR:
		w := bufio.NewWriter(os.Stderr)
		w.WriteString(out)
		w.Flush()

	case OUT_FILE:
		if l.file == nil {
			fmt.Fprintln(os.Stderr, "Could not open log file for writing.")
			return
		} else {
			l.writer.WriteString(out)
			l.writer.Flush()
		}
	}
}

// -------Batching--------------------------------------------------------------

func (l *logger) appendItemToBatch(e *Entry) bool {
	msg, err := formatByLevel(*e)
	if err != nil {
		msg = COULD_NOT_WRITE_ENTRY
	}
	l.batchBuf = append(l.batchBuf, msg)

	if !(l.maxItems > 0 && len(l.batchBuf) >= l.maxItems) {
		return false
	}

	out := strings.Join(l.batchBuf, "")
	l.writeToOut(out)
	l.batchBuf = []string{}

	return true
}

func (l *logger) appendBytesToBatch(e *Entry) bool {
	msg, err := formatByLevel(*e)
	if err != nil {
		msg = COULD_NOT_WRITE_ENTRY
	}
	l.batchBuf = append(l.batchBuf, msg)

	out := strings.Join(l.batchBuf, "")

	if !(l.maxBytes > 0 && len(out) >= l.maxBytes) {
		return false
	}

	l.writeToOut(out)
	l.batchBuf = []string{}

	return true
}

func (l *logger) appendToTimer(e *Entry) {
	msg, err := formatByLevel(*e)
	if err != nil {
		msg = COULD_NOT_WRITE_ENTRY
	}
	l.batchBuf = append(l.batchBuf, msg)

	if l.maxWait > 0 {
		// Drain the channel if needed to avoid spurious wakeups.
		if !l.timer.Stop() {
			select {
			case <-l.timer.C:
			default:
			}
		}

		l.wg.Add(1)
		l.timer.Reset(l.maxWait)
	}
}

// -------Primary Exported Functions--------------------------------------------

// The global logger handles logs concurrently, so there is no need for more
// than one.
var globalLogger = newLogger()

// LogEntry sends entry to the logger, labeled using caller, when the current
// logging level is greater than or equal to level.
func LogEntry(entry, caller string, level LogLevel) {
	if GetLogLevel() == LL_NONE {
		return
	}

	le := &Entry{
		Caller: caller,
		Entry:  entry,
		Level:  level,
	}

	if le.Level > GetLogLevel() {
		return
	}

	switch GetBatchMode() {

	case BATCH_ITEM:
		globalLogger.appendItemToBatch(le)

	case BATCH_BYTE:
		globalLogger.appendBytesToBatch(le)

	case BATCH_TIME:
		globalLogger.appendToTimer(le)

	default:
		globalLogger.wg.Add(1)
		globalLogger.in <- le
	}
}

// Flush blocks until all currently enqueued log entries are processed.
func Flush() {
	globalLogger.wg.Wait()
}

// Shutdown flushes all logs, closes the input channel, and waits for the
// logger goroutine to exit cleanly.
func Shutdown() {
	// If there’s a time/item/byte batch sitting in memory, flush it now.
	if len(globalLogger.batchBuf) > 0 {
		out := strings.Join(globalLogger.batchBuf, "")
		globalLogger.writeToOut(out)
		globalLogger.batchBuf = []string{}
	}

	// Ensure everything enqueued so far is processed.
	Flush()

	// Closing 'in' lets the loop's 'range' exit.
	close(globalLogger.in)
}

func SetMaxItems(n int)          { globalLogger.maxItems = n }
func SetMaxBytes(n int)          { globalLogger.maxBytes = n }
func SetMaxWait(d time.Duration) { globalLogger.maxWait = d }
