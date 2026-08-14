package golang

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLimitedBufferRecordsTruncation(t *testing.T) {
	buffer := newLimitedBuffer(4)
	input := []byte("abcdef")
	written, err := buffer.Write(input)
	if err != nil {
		t.Fatal(err)
	}
	if written != len(input) || string(buffer.Bytes()) != "abcd" || !buffer.truncated {
		t.Fatalf("unexpected bounded write: written=%d bytes=%q truncated=%t", written, buffer.Bytes(), buffer.truncated)
	}
}

func TestRunCommandTimeoutIsBounded(t *testing.T) {
	if os.Getenv("TESTULE_HELPER_SLEEP") == "1" {
		time.Sleep(5 * time.Second)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	env := append(os.Environ(), "TESTULE_HELPER_SLEEP=1")
	result, err := runCommand(ctx, os.Args[0], []string{"-test.run=TestRunCommandTimeoutIsBounded"}, ".", env)
	if err != nil {
		t.Fatal(err)
	}
	if !result.timedOut || result.exitCode != -1 {
		t.Fatalf("expected bounded timeout, got timedOut=%t exit=%d stderr=%s", result.timedOut, result.exitCode, strings.TrimSpace(string(result.stderr)))
	}
	if result.duration > 3*time.Second {
		t.Fatalf("timeout exceeded bounded shutdown window: %s", result.duration)
	}
}
