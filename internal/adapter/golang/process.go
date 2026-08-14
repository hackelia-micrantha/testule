package golang

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

type commandResult struct {
	stdout    []byte
	stderr    []byte
	exitCode  int
	duration  time.Duration
	timedOut  bool
	truncated bool
}

type limitedBuffer struct {
	buffer    bytes.Buffer
	remaining int64
	truncated bool
}

func newLimitedBuffer(limit int64) *limitedBuffer {
	return &limitedBuffer{remaining: limit}
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	if w.remaining <= 0 {
		w.truncated = w.truncated || original > 0
		return original, nil
	}
	writeCount := int64(len(p))
	if writeCount > w.remaining {
		writeCount = w.remaining
		w.truncated = true
	}
	if _, err := w.buffer.Write(p[:writeCount]); err != nil {
		return 0, err
	}
	w.remaining -= writeCount
	return original, nil
}

func (w *limitedBuffer) Bytes() []byte { return w.buffer.Bytes() }

func runCommand(ctx context.Context, binary string, args []string, dir string, env []string) (commandResult, error) {
	stdout := newLimitedBuffer(MaxOutputBytes)
	stderr := newLimitedBuffer(MaxOutputBytes)
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = 2 * time.Second

	started := time.Now()
	err := cmd.Run()
	result := commandResult{
		stdout:    append([]byte(nil), stdout.Bytes()...),
		stderr:    append([]byte(nil), stderr.Bytes()...),
		duration:  time.Since(started),
		timedOut:  errors.Is(ctx.Err(), context.DeadlineExceeded),
		truncated: stdout.truncated || stderr.truncated,
	}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.exitCode = exitError.ExitCode()
		return result, nil
	}
	if result.timedOut {
		result.exitCode = -1
		return result, nil
	}
	return result, err
}
