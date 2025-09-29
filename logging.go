package siglog

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
func formatByLevel(lm logEntry) (string, error) {
	if len(lm.entry) == 0 || lm.entry[len(lm.entry)-1] != '\n' {
		lm.entry += "\n"
	}

	now := time.Now().Format(TimeLayout)
	tok := levelName[GetLogLevel()]
	out := fmt.Sprintf("%s: [%s][%s] - %s", now, lm.caller, tok, lm.entry)

	return out, nil
}

// -------Logger----------------------------------------------------------------

type logEntry struct {
	// What "source" the log message is coming from.
	// This should be a file, struct, stream, etc.
	caller string

	// The message to log.
	entry string

	// What logging level is required in the environment for the log to be
	// written.
	level LogLevel
}

// logger writes to a dated log file.
// Use logging.GlobalLogger instead of making a new one.
type logger struct {
	file   *os.File
	in     chan *logEntry
	date   string
	writer *bufio.Writer
}

func newLogger() *logger {
	file, err := os.OpenFile(
		getTodaysLogFilePath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
	)
	if err != nil {
		file = nil
	}

	l := &logger{
		file:   file,
		date:   getToday(),
		in:     make(chan *logEntry, 1024),
		writer: bufio.NewWriter(file),
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

	for entry := range l.in {
		if entry == nil {
			continue
		}

		if GetLogLevel() == LL_NONE {
			continue
		}

		if entry.level <= GetLogLevel() {
			switch GetOutput() {
			case OUT_STDOUT:
				l.writeToStdOut(entry)
			case OUT_STDERR:
				l.writeToStdErr(entry)
			case OUT_FILE:
				l.writeToFile(entry)
			}
		}

		if getToday() != l.date {
			l.rotate()
		}
	}
}

func (l *logger) writeToStdOut(e *logEntry) {
	writer := bufio.NewWriter(os.Stdout)
	flush := func() {
		_ = writer.Flush()
	}
	defer flush()

	var msg string
	msg, err := formatByLevel(*e)
	if err != nil {
		msg = COULD_NOT_WRITE_ENTRY
	}

	_, err = writer.WriteString(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, COULD_NOT_WRITE_ENTRY)
	}
}

func (l *logger) writeToStdErr(e *logEntry) {
	writer := bufio.NewWriter(os.Stderr)
	flush := func() {
		_ = writer.Flush()
	}
	defer flush()

	var msg string
	msg, err := formatByLevel(*e)
	if err != nil {
		msg = COULD_NOT_WRITE_ENTRY
	}

	_, err = writer.WriteString(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, COULD_NOT_WRITE_ENTRY)
	}
}

func (l *logger) writeToFile(e *logEntry) {
	var msg string

	if l.file == nil {
		msg = "Could not open log file for writing."
		fmt.Fprintln(os.Stderr, msg)
		return
	}

	writer := bufio.NewWriter(l.file)
	flush := func() {
		_ = writer.Flush()
	}
	defer flush()

	msg, err := formatByLevel(*e)
	if err != nil {
		msg = COULD_NOT_WRITE_ENTRY
	}

	_, err = writer.WriteString(msg)
	if err != nil {
		fmt.Fprintln(os.Stderr, COULD_NOT_WRITE_ENTRY)
	}
}

// -------Global Singleton Logger-----------------------------------------------

// The global logger handles logs concurrently, so there is no need for more
// than one.
var globalLogger = newLogger()

// LogEntry sends entry to the logger, labeled using caller, when the current
// logging level is greater than or equal to level.
func LogEntry(entry, caller string, level LogLevel) {
	if GetLogLevel() == LL_NONE {
		return
	}

	ml := &logEntry{
		caller: caller,
		entry:  entry,
		level:  level,
	}
	globalLogger.in <- ml
}
