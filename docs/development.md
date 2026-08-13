# Development and validation

Testule validates strict local `testule.dev/v1alpha1` resources and keeps execution behind later adapter slices.

## Toolchain

The repository pins Go 1.26.5 in mise and CI. YAML decoding uses the maintained `go.yaml.in/yaml/v3` line rather than the still-prerelease v4 API, and Staticcheck is pinned to `v0.7.0` (2026.1).

## CLI

```text
testule validate [--format text|json] <plan.yaml>
testule fingerprint <plan.yaml>
testule gaps [--format text|json] --subject-revision <revision> <plan.yaml> [evidence.yaml ...]
```

`validate` reads one local TestPlan YAML document and performs strict structural and semantic validation.

`fingerprint` validates a TestPlan and returns the deterministic plan fingerprint used to bind Evidence.

`gaps` validates a TestPlan and zero or more supplied Evidence documents, verifies their identity/provenance binding, and emits a deterministic coverage/gap report.

These commands do not execute tests, resolve templates, access the network, dereference Evidence references, or mutate the workspace. TestPlan and Evidence input files are each bounded to 1 MiB.

See [Evidence and gap analysis](evidence.md) for the Evidence contract and evaluator semantics.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Validation/evaluation succeeded; required gap requirements are satisfied |
| `1` | Internal output/runtime failure |
| `2` | CLI usage error |
| `3` | Plan or Evidence is malformed, semantically invalid, stale, or mismatched |
| `4` | Input could not be read or exceeds the bounded input size |
| `5` | Gap evaluation completed and blocking gaps/failures remain |

Validation JSON uses a stable top-level `valid`, `source`, and `diagnostics` envelope. Gap JSON uses the normalized `Report` envelope documented in `docs/evidence.md`.

## Canonical repository checks

With [mise](https://mise.jdx.dev/) installed:

```text
mise run fmt
mise run tidy
mise run vet
mise run staticcheck
mise run test
mise run fuzz-smoke
mise run build
mise run ci
```

CI mirrors the same required checks. Staticcheck, Go, and third-party GitHub Actions are version/SHA pinned.

## Validation layers

- **Unit/component:** strict TestPlan/Evidence decoding, semantic validation, fingerprinting, and gap state evaluation.
- **Contract/integration:** CLI runner and real process tests cover output, exit codes, plan fingerprinting, and plan/evidence evaluation.
- **Fuzz/property:** bounded TestPlan and Evidence decoder smoke campaigns assert arbitrary bounded YAML cannot panic decoding/validation.
- **End-to-end:** not applicable yet because Testule does not execute a native external test path.
- **Security/negative:** strict unknown-field handling, malformed/multiple documents, contradictory declarations, duplicate identities, stale/mismatched revisions/fingerprints, bounded strings/collections, and bounded file reads.
