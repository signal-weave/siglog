package siglog

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// captureWriter swaps *os.File (stdout or stderr) for the duration of fn and
// returns what was written to it.
func captureWriter(target **os.File, fn func()) string {
	orig := *target
	r, w, _ := os.Pipe()
	*target = w
	defer func() {
		_ = w.Close()
		*target = orig
	}()

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

// restore default formatter after tests that change it.
func resetFormatter() {
	SetLogFormatter(defaultFormatByLevel)
}

// -------Tests-----------------------------------------------------------------

func TestLogEntry_Stdout_DefaultFormatter(t *testing.T) {
	t.Cleanup(func() {
		SetBatchMode(BATCH_NONE)
		SetOutput(OUT_STDOUT)
		SetLogLevel(LL_NONE)
		resetFormatter()
	})

	// Ensure messages at/above level are emitted.
	if err := SetLogLevel(LL_DEBUG); err != nil {
		t.Fatalf("SetLogLevel: %v", err)
	}
	if err := SetOutput(OUT_STDOUT); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	if err := SetBatchMode(BATCH_NONE); err != nil {
		t.Fatalf("SetBatchMode: %v", err)
	}

	got := captureWriter(&os.Stdout, func() {
		LogEntry("hello world", "SYSTEM", LL_DEBUG)
		Flush()
	})

	// Expect stable substrings from defaultFormatByLevel:
	// "<time>: [SYSTEM][DEBUG] - hello world\n"
	if !strings.Contains(got, "[SYSTEM][DEBUG] - hello world\n") {
		t.Fatalf("unexpected stdout content:\n%s", got)
	}
}

func TestLogEntry_File_Output(t *testing.T) {
	t.Cleanup(func() {
		SetBatchMode(BATCH_NONE)
		SetOutput(OUT_STDOUT)
		SetLogLevel(LL_NONE)
		// Clean up the log file we created (best-effort).
		_ = os.Remove(getTodaysLogFilePath())
	})

	if err := SetLogLevel(LL_INFO); err != nil {
		t.Fatalf("SetLogLevel: %v", err)
	}
	if err := SetOutput(OUT_FILE); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	if err := SetBatchMode(BATCH_NONE); err != nil {
		t.Fatalf("SetBatchMode: %v", err)
	}

	LogEntry("file path test", "SYSTEM", LL_INFO)
	Flush()

	// Read whichever path the logger actually opened at init time.
	path := getTodaysLogFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log file %q: %v", path, err)
	}
	if !strings.Contains(string(data), "[SYSTEM][INFO] - file path test\n") {
		t.Fatalf("unexpected file content:\n%s", string(data))
	}
}

func TestSetLogFormatter_Custom(t *testing.T) {
	t.Cleanup(func() {
		resetFormatter()
		SetBatchMode(BATCH_NONE)
		SetOutput(OUT_STDOUT)
		SetLogLevel(LL_DEBUG)
	})

	if err := SetLogLevel(LL_DEBUG); err != nil {
		t.Fatalf("SetLogLevel: %v", err)
	}
	if err := SetOutput(OUT_STDOUT); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	if err := SetBatchMode(BATCH_NONE); err != nil {
		t.Fatalf("SetBatchMode: %v", err)
	}

	SetLogFormatter(func(e Entry) (string, error) {
		return "custom\n", nil
	})

	got := captureWriter(&os.Stdout, func() {
		LogEntry("ignored", "SYSTEM", LL_DEBUG)
		Flush()
	})

	if got != "custom\n" {
		t.Fatalf("expected custom formatter output, got:\n%q", got)
	}
}

func TestBatchItem_FlushOnCount(t *testing.T) {
	t.Cleanup(func() {
		SetBatchMode(BATCH_NONE)
		SetMaxItems(128) // restore default-ish
		SetOutput(OUT_STDOUT)
		SetLogLevel(LL_DEBUG)
	})

	if err := SetLogLevel(LL_DEBUG); err != nil {
		t.Fatalf("SetLogLevel: %v", err)
	}
	if err := SetOutput(OUT_STDOUT); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	if err := SetBatchMode(BATCH_ITEM); err != nil {
		t.Fatalf("SetBatchMode: %v", err)
	}
	SetMaxItems(2)

	got := captureWriter(&os.Stdout, func() {
		// Item batching writes immediately when threshold is met; no Flush needed.
		LogEntry("one", "SYSTEM", LL_DEBUG)
		LogEntry("two", "SYSTEM", LL_DEBUG)
		// Give buffered writer a nudge by an extra newline write to stdout.
		// (Not strictly necessary; the bufio.NewWriter(Stdout).Flush() is called in writeToOut.)
	})

	// Expect two formatted lines, joined.
	// We don't know the exact time token, so match the stable pieces.
	reader := bufio.NewScanner(strings.NewReader(got))
	var lines []string
	for reader.Scan() {
		lines = append(lines, reader.Text())
	}
	if len(lines) < 2 {
		t.Fatalf("expected >=2 lines from item batch, got %d\nFull output:\n%s", len(lines), got)
	}
	if !strings.Contains(lines[0], "[SYSTEM][DEBUG] - one") || !strings.Contains(lines[1], "[SYSTEM][DEBUG] - two") {
		t.Fatalf("unexpected batch item output:\n%s", got)
	}
}

func TestBatchTime_FiresAfterWaitAndFlushes(t *testing.T) {
	t.Cleanup(func() {
		SetBatchMode(BATCH_NONE)
		SetMaxWait(250 * time.Millisecond) // restore default-ish
		SetOutput(OUT_STDOUT)
		SetLogLevel(LL_DEBUG)
	})

	if err := SetLogLevel(LL_DEBUG); err != nil {
		t.Fatalf("SetLogLevel: %v", err)
	}
	if err := SetOutput(OUT_STDOUT); err != nil {
		t.Fatalf("SetOutput: %v", err)
	}
	if err := SetBatchMode(BATCH_TIME); err != nil {
		t.Fatalf("SetBatchMode: %v", err)
	}
	SetMaxWait(30 * time.Millisecond)

	got := captureWriter(&os.Stdout, func() {
		LogEntry("tick", "SYSTEM", LL_DEBUG)
		// Flush will block until the timer path decrements wg.
		Flush()
	})

	if !strings.Contains(got, "[SYSTEM][DEBUG] - tick\n") {
		t.Fatalf("expected time-batch flush, got:\n%s", got)
	}
}
