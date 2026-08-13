# Development and validation

Testule validates strict local `testule.dev/v1alpha1` resources and now includes one deliberately narrow native execution adapter for Go tests and fuzzing.

## Toolchain

The repository pins Go 1.26.5 in mise and CI. YAML decoding uses the maintained `go.yaml.in/yaml/v3` line rather than the still-prerelease v4 API, and Staticcheck is pinned to `v0.7.0` (2026.1).

## CLI

```text
testule validate [--format text|json] <plan.yaml>
testule fingerprint <plan.yaml>
testule gaps [--format text|json] --subject-revision <revision> <plan.yaml> [evidence.yaml ...]
testule go <test|fuzz|replay|promote> ...
```

`validate` reads one local TestPlan YAML document and performs strict structural and semantic validation.

`fingerprint` validates a TestPlan and returns the deterministic plan fingerprint used to bind Evidence.

`gaps` validates a TestPlan and zero or more supplied Evidence documents, verifies their identity/provenance binding, and emits a deterministic coverage/gap report.

The `go` command family executes only exact local Go test/fuzz targets through the first native adapter. See [Native Go test and fuzz adapter](adapters/go.md) for command forms, execution bounds, replay, promotion, and the explicit isolation limitations.

TestPlan and Evidence input files are each bounded to 1 MiB. Validation/gap commands do not resolve templates, access the network, dereference Evidence references, or mutate the workspace. Native Go execution uses constrained subprocesses and Testule-managed artifacts; explicit `go promote` is the only adapter operation in this slice intended to persist a regression corpus mutation.

See [Evidence and gap analysis](evidence.md) for the Evidence contract and evaluator semantics.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Validation/evaluation or native adapter operation succeeded |
| `1` | Internal output/runtime failure |
| `2` | CLI usage error |
| `3` | Plan or Evidence is malformed, semantically invalid, stale, or mismatched |
| `4` | Input or adapter workspace/artifact operation failed |
| `5` | Gap evaluation completed and blocking gaps/failures remain |
| `6` | Native adapter execution produced failed test/fuzz/replay Evidence |
| `7` | Requested native Go tool/target is unsupported or unavailable |

Validation JSON uses a stable top-level `valid`, `source`, and `diagnostics` envelope. Gap JSON uses the normalized `Report` envelope documented in `docs/evidence.md`. Native adapter runs persist their normalized Evidence as JSON in `.testule/artifacts/<run-id>/evidence.json` and print its path.

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

- **Unit/component:** strict TestPlan/Evidence decoding, semantic validation, fingerprinting, adapter validation, artifact policy, process bounds, and gap state evaluation.
- **Contract/integration:** CLI runner and real process tests cover output, exit codes, plan fingerprinting, native passing/failing Go tests, native bounded fuzz execution, and plan/evidence evaluation.
- **End-to-end:** a TestPlan drives Go test/fuzz adapter execution, emitted Evidence is consumed by `gaps`, and the remaining missing requirement is reported deterministically.
- **Fuzz/property:** bounded TestPlan and Evidence decoder smoke campaigns assert arbitrary bounded YAML cannot panic decoding/validation; the adapter integration suite also exercises the native Go fuzz failure/reproducer lifecycle without making a long-running campaign a PR gate.
- **Security/negative:** strict unknown fields, malformed/multiple documents, contradictory declarations, stale/mismatched revisions/fingerprints, exact target/package validation, traversal/symlink rejection, bounded output/time, disabled Go downloads, artifact digest verification, and explicit promotion.

The direct Go adapter does not provide OS-level network/CPU/memory isolation. Tests requiring those controls must run beneath a host/environment boundary that can enforce them.
