# Evidence streaming contract

Status: incubating `v1alpha1` contract

Tracking: #27 under Unix-composability epic #25

## Purpose

Testule Evidence is already a normalized, versioned domain record. Unix streaming should transport that same record without introducing a second evidence model or requiring temporary interchange files.

The first proven producer is the native Go adapter: `testule go test` and `testule go fuzz` already construct normalized Evidence. With `--format jsonl`, those commands emit the same in-memory Evidence record on stdout as one JSON Lines record while retaining the adapter's existing persistent Evidence artifact.

`testule gaps` consumes Evidence JSONL with `--evidence-format jsonl`.

## JSON versus JSONL

Use **JSON** for one finite command result, such as a validation result or gap report:

```sh
testule gaps --format json --subject-revision "$REV" plan.yaml evidence.yaml
```

Use **JSONL** when stdout represents a sequence of independent normalized Evidence records:

```sh
testule go test ... --format jsonl
```

JSONL is a transport framing format, not a new Testule resource kind. Every line is a normal Evidence object carrying its existing schema identity:

```json
{"apiVersion":"testule.dev/v1alpha1","kind":"Evidence",...}
```

There is no stream envelope. `apiVersion` and `kind` version each record directly.

## Framing and bounds

The `v1alpha1` Evidence JSONL contract is deliberately strict and bounded:

- each physical line contains exactly one JSON object;
- blank lines are invalid;
- unknown object fields are invalid;
- a complete final record is accepted without a trailing newline;
- a truncated or otherwise malformed final record is invalid;
- each record is bounded by the normal Evidence size limit of 1 MiB;
- one JSONL input stream is bounded to 16 MiB and 128 Evidence records;
- records are decoded and returned in input order;
- duplicate records are preserved rather than silently deduplicated; downstream Evidence/gap semantics remain authoritative.

Malformed input fails deterministically with an `invalid_jsonl`, `input_too_large`, or other normal Evidence diagnostic. A partially valid prefix is never silently evaluated after a malformed record: the input stream is rejected as a whole.

## Identity and provenance

Streaming must not rewrite Evidence identity or provenance. In particular, these fields survive encode/decode unchanged:

- `apiVersion` and `kind`;
- `metadata.name`;
- `plan.name` and `plan.fingerprint`;
- `subject.component` and `subject.revision`;
- `environment.id`;
- `provenance.producer`, `provenance.runId`, and references;
- execution metadata, artifacts, observations, and coverage.

The Go adapter's JSONL output is serialized from the same normalized Evidence object that is persisted to its Evidence artifact. The stream is therefore another transport for the same result, not a separately synthesized summary.

## Composition

### Direct Testule-to-Testule pipeline

```sh
testule go test \
  --plan plan.yaml \
  --subject-revision "$REV" \
  --workspace . \
  --package ./... \
  --target TestExample \
  --level unit \
  --environment local \
  --run-id "$RUN_ID" \
  --format jsonl \
| testule gaps \
    --evidence-format jsonl \
    --subject-revision "$REV" \
    --format json \
    plan.yaml -
```

No temporary Evidence file is required for interchange between the two Testule operations. The adapter may still retain its normal persistent Evidence artifact for reproducibility and auditability.

### Preserve the stream with `tee`

```sh
testule go test ... --format jsonl \
| tee evidence.jsonl \
| testule gaps \
    --evidence-format jsonl \
    --subject-revision "$REV" \
    --format json \
    plan.yaml -
```

### Filter records with `jq`

`jq -c` preserves one compact JSON object per output line and can therefore participate in a JSONL pipeline:

```sh
testule go test ... --format jsonl \
| jq -c 'select(.kind == "Evidence")' \
| testule gaps \
    --evidence-format jsonl \
    --subject-revision "$REV" \
    --format json \
    plan.yaml -
```

Filtering Evidence is an explicit operator transformation. Testule does not reconstruct removed provenance or claim that a filtered stream is identical to the producer's complete output.

## Exit status and pipelines

Machine output and process status remain separate contracts. A native test failure can produce valid failure Evidence on stdout and still return adapter exit status `6`.

Without shell `pipefail`, a pipeline's status is normally the final consumer's status. With `set -o pipefail`, an upstream adapter failure remains visible even if `gaps` successfully consumes the failure Evidence. This is intentional: Evidence describes what happened; exit status remains shell control flow.

EPIPE/broken-pipe handling follows the general CLI contract in `docs/cli-unix-composability.md`.

## Agent and supervisor transports

Future agent/supervisor bindings should transport the same Evidence objects and preserve the same schema identity, provenance, ordering, and validation semantics. JSONL is specifically the Unix stream framing; it is not required as the wire framing for non-Unix transports.

Transport metadata such as actor identity, approval context, or capability grants may wrap an invocation externally, but must not mutate the normalized Evidence record to create different CLI versus agent meanings.
