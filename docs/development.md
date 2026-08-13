# Development and validation

Testule's first executable slice intentionally validates only the strict local `testule.dev/v1alpha1` `TestPlan` subset accepted by RFC 0001.

## Toolchain

The repository pins Go 1.26.5 in mise and CI. YAML decoding uses the maintained `go.yaml.in/yaml/v3` line rather than the still-prerelease v4 API, and Staticcheck is pinned to `v0.7.0` (2026.1).

## CLI

```text
testule validate [--format text|json] <plan.yaml>
```

`validate` reads one local YAML document, performs strict decoding and semantic validation, and does not execute tests, resolve references or templates, access the network, or mutate the workspace. Input is bounded to 1 MiB.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | Plan is valid |
| `1` | Internal output/runtime failure |
| `2` | CLI usage error |
| `3` | Plan is malformed or semantically invalid |
| `4` | Input could not be read or exceeds the bounded input size |

JSON output uses a stable top-level `valid`, `source`, and `diagnostics` envelope. Diagnostics expose a stable `code`, optional field `path`, and human-readable `message`. A valid result emits `diagnostics` as an empty array rather than `null`.

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

CI mirrors the same required checks. Staticcheck and the Go runtime are version-pinned.

## Validation layers in this slice

- **Unit/component:** YAML decoding and semantic validation, including boundaries and negative cases.
- **Contract/integration:** CLI runner and a real process test cover output streams and exit codes.
- **Fuzz/property:** a bounded parser fuzz smoke test asserts arbitrary bounded YAML input cannot panic the decoder/validator. Longer-running fuzz campaigns are intentionally deferred from the per-PR gate.
- **End-to-end:** not applicable yet because Testule does not execute an external test path in this slice.
- **Security/negative:** strict unknown-field handling, malformed input, multiple YAML documents, invalid values, control characters, and bounded input size.
