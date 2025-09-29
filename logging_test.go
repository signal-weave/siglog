package siglog

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -------helpers---------------------------------------------------------------

func captureStdStream(replace **os.File, run func()) string {
	orig := *replace
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	*replace = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	run()

	_ = w.Close()
	<-done
	*replace = orig
	return buf.String()
}

// -------tests-----------------------------------------------------------------

func TestSetLogDirectory_CreatesDirAndEnv(t *testing.T) {
	td := t.TempDir()

	// call under test
	if err := SetLogDirectory(td); err != nil {
		t.Fatalf("SetLogDirectory error: %v", err)
	}

	got := os.Getenv(ENV_SL_LOGDIR)
	if got != td {
		t.Fatalf("ENV_SL_LOGDIR not set: got %q want %q", got, td)
	}

	// Should be able to join today's path within the temp dir.
	path := getTodaysLogFilePath()
	if !strings.HasPrefix(path, td) {
		t.Fatalf("log path not inside temp dir: %q", path)
	}
}

func TestFormatByLevel_IncludesCallerLevelAndMessage(t *testing.T) {
	// Ensure a non-NONE level so formatting uses a real token.
	_ = SetLogLevel(LL_INFO)

	lm := logEntry{
		caller: "UnitTest",
		entry:  "hello world",
		level:  LL_INFO,
	}
	msg, err := formatByLevel(lm)
	if err != nil {
		t.Fatalf("formatByLevel error: %v", err)
	}

	// Expected fragments (timestamp is variable, so we match stable parts).
	if !strings.Contains(msg, "[UnitTest][INFO] - hello world\n") {
		t.Fatalf("formatted text missing pieces:\n%q", msg)
	}
}

func TestLogger_WriteToStdOut_Writes(t *testing.T) {
	_ = SetLogLevel(LL_INFO)

	l := newLogger()
	defer func() { _ = l.file.Close() }()

	out := captureStdStream(&os.Stdout, func() {
		l.writeToStdOut(&logEntry{
			caller: "StdoutTest",
			entry:  "to stdout",
			level:  LL_INFO,
		})
	})

	if !strings.Contains(out, "[StdoutTest][INFO] - to stdout") {
		t.Fatalf("stdout missing expected content:\n%q", out)
	}
}

func TestLogger_WriteToStdErr_Writes(t *testing.T) {
	_ = SetLogLevel(LL_INFO)

	l := newLogger()
	defer func() { _ = l.file.Close() }()

	out := captureStdStream(&os.Stderr, func() {
		l.writeToStdErr(&logEntry{
			caller: "StderrTest",
			entry:  "to stderr",
			level:  LL_INFO,
		})
	})

	if !strings.Contains(out, "[StderrTest][INFO] - to stderr") {
		t.Fatalf("stderr missing expected content:\n%q", out)
	}
}

func TestLogger_WriteToFile_Writes(t *testing.T) {
	_ = SetLogLevel(LL_INFO)

	td := t.TempDir()
	if err := SetLogDirectory(td); err != nil {
		t.Fatalf("SetLogDirectory error: %v", err)
	}

	l := newLogger()
	defer func() { _ = l.file.Close() }()

	l.writeToFile(&logEntry{
		caller: "FileTest",
		entry:  "to file",
		level:  LL_INFO,
	})

	// Read back today's file from the temp dir.
	path := getTodaysLogFilePath()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read log file %q: %v", path, err)
	}
	got := string(b)
	if !strings.Contains(got, "[FileTest][INFO] - to file") {
		t.Fatalf("file missing expected content:\n%q", got)
	}
}

func TestLogEntryAndFlush_UsesGlobalLogger(t *testing.T) {
	// Route to STDOUT to make capture deterministic and avoid file IO here.
	_ = SetOutput(OUT_STDOUT)
	_ = SetLogLevel(LL_INFO)

	out := captureStdStream(&os.Stdout, func() {
		LogEntry("hello via global", "GlobalTest", LL_INFO)
		Flush() // wait for the goroutine to drain the queue
	})

	if !strings.Contains(out, "[GlobalTest][INFO] - hello via global") {
		t.Fatalf("global logger output missing content:\n%q", out)
	}
}

func TestGetTodaysLogFileNameAndPath_AreStableShapes(t *testing.T) {
	td := t.TempDir()
	if err := SetLogDirectory(td); err != nil {
		t.Fatalf("SetLogDirectory error: %v", err)
	}
	name := getTodaysLogFileName()
	path := getTodaysLogFilePath()

	if !strings.HasPrefix(name, "mycelia-log-") || !strings.HasSuffix(name, ".log") {
		t.Fatalf("unexpected file name: %q", name)
	}
	if filepath.Base(path) != name {
		t.Fatalf("path base mismatch: path=%q name=%q", path, name)
	}
}
