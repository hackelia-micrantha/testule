package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hackelia-micrantha/testule/internal/evidence"
)

func TestGoAdapterEvidenceJSONLPipesDirectlyIntoGaps(t *testing.T) {
	binary := buildTestuleBinary(t)
	workspace := copyGoAdapterFixture(t)
	planPath := filepath.Clean(filepath.Join("..", "..", "testdata", "valid", "gap-plan.yaml"))

	producer := exec.Command(binary,
		"go", "test",
		"--plan", planPath,
		"--subject-revision", "rev-stream",
		"--workspace", workspace,
		"--package", "./sample",
		"--target", "TestPass",
		"--level", "unit",
		"--behavior", "negative",
		"--environment", "ci-linux",
		"--run-id", "stream-unit",
		"--format", "jsonl",
	)
	consumer := exec.Command(binary,
		"gaps",
		"--format", "json",
		"--evidence-format", "jsonl",
		"--subject-revision", "rev-stream",
		planPath,
		"-",
	)

	reader, writer := io.Pipe()
	var streamed, producerStderr, consumerStdout, consumerStderr bytes.Buffer
	producer.Stdout = io.MultiWriter(writer, &streamed)
	producer.Stderr = &producerStderr
	consumer.Stdin = reader
	consumer.Stdout = &consumerStdout
	consumer.Stderr = &consumerStderr

	if err := consumer.Start(); err != nil {
		t.Fatal(err)
	}
	producerErr := producer.Run()
	_ = writer.CloseWithError(producerErr)
	consumerErr := consumer.Wait()

	if producerErr != nil {
		t.Fatalf("producer failed: %v; stderr=%q", producerErr, producerStderr.String())
	}
	var exitErr *exec.ExitError
	if !errors.As(consumerErr, &exitErr) || exitErr.ExitCode() != 5 {
		t.Fatalf("expected gaps exit 5, got %v; stdout=%q stderr=%q", consumerErr, consumerStdout.String(), consumerStderr.String())
	}
	if producerStderr.Len() != 0 || consumerStderr.Len() != 0 {
		t.Fatalf("unexpected diagnostics: producer=%q consumer=%q", producerStderr.String(), consumerStderr.String())
	}

	lines := strings.Split(strings.TrimSuffix(streamed.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected exactly one JSONL record, got %d: %q", len(lines), streamed.String())
	}
	var record evidence.Evidence
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("producer output is not JSON: %v", err)
	}
	if record.APIVersion != "testule.dev/v1alpha1" || record.Kind != "Evidence" {
		t.Fatalf("missing stream schema identity: %#v", record)
	}
	if record.Subject.Revision != "rev-stream" || record.Provenance.RunID != "stream-unit" {
		t.Fatalf("identity/provenance changed in stream: %#v", record)
	}

	if !strings.Contains(consumerStdout.String(), `"value":"unit"`) || !strings.Contains(consumerStdout.String(), `"state":"satisfied"`) {
		t.Fatalf("gaps did not consume streamed evidence: %s", consumerStdout.String())
	}
}
