# Native Go test and fuzz adapter

The first Testule execution adapter deliberately targets a narrow, inspectable subset of the Go toolchain. It proves the adapter/Evidence boundary without defining a general plugin system.

## Commands

```text
testule go test \
  --plan <plan.yaml> \
  --subject-revision <revision> \
  --workspace <go-module> \
  --package <./package> \
  --target <TestName> \
  --level <level> \
  --environment <environment-id> \
  --run-id <run-id>

testule go fuzz \
  --plan <plan.yaml> \
  --subject-revision <revision> \
  --workspace <go-module> \
  --package <./package> \
  --target <FuzzName> \
  --level <level> \
  --environment <environment-id> \
  --run-id <run-id> \
  [--fuzztime 1s] [--timeout 30s]

testule go replay \
  --evidence <failed-fuzz-evidence.json> \
  --subject-revision <revision> \
  --workspace <go-module> \
  --environment <environment-id> \
  --run-id <run-id>

testule go promote \
  --evidence <failed-fuzz-evidence.json> \
  --subject-revision <revision> \
  --workspace <go-module>
```

`--package` is restricted to `.` or an exact `./relative/package` path. `...`, import paths, traversal, backslashes, and symlink escapes are rejected. Test targets must be exact `Test*` identifiers and fuzz targets exact `Fuzz*` identifiers; callers do not supply arbitrary Go regular expressions or shell fragments.

The adapter verifies target existence with `go test -list` before execution. A missing Go tool or missing exact target is normalized as `unsupported`. Compilation/listing errors, test failures, fuzz failures, and timeouts are normalized as `failed` Evidence.

## Execution bounds

Testule invokes Go directly with `exec.CommandContext`; no shell is involved. The first adapter fixes or bounds:

- wall time: greater than zero and at most 2 minutes;
- fuzz campaign time: greater than zero and at most 30 seconds, with additional timeout headroom;
- Go test parallelism: `-parallel=1`;
- captured stdout and stderr: 1 MiB each, with truncation recorded;
- child-process pipe shutdown: bounded with a short wait delay;
- module/toolchain downloads: disabled with `GOPROXY=off`, `GOSUMDB=off`, and `GOTOOLCHAIN=local`;
- CGO: disabled for this direct adapter;
- Go caches, temporary files, HOME, and GOPATH: scoped beneath the Testule run artifact directory.

The child environment is constructed explicitly instead of inheriting ambient environment variables and secrets. Tests needing dependencies must therefore be self-contained or use dependency material already made available through an explicit future environment policy, such as vendored dependencies.

### Isolation boundary

The direct adapter does **not** claim OS-level network, CPU, memory, process-count, or syscall isolation. Test code can still use authority available to the operating-system process, including host network access. When those controls are mandatory, a capability host or TestEnvironment provider must enforce them before invoking the adapter; unsupported mandatory isolation must fail closed rather than being represented as isolated execution.

## Evidence

Adapter-produced Evidence adds optional execution and artifact metadata to the normalized Evidence envelope:

```yaml
execution:
  adapter: go-native/v1alpha1
  operation: fuzz
  tool: go
  toolVersion: go1.26.5
  package: ./parser
  target: FuzzDecode
  command: [go, test, ...]
  exitCode: 1
  durationMillis: 842
  timedOut: false
  outputTruncated: false
artifacts:
  - name: 3f2a...
    role: fuzz-reproducer
    path: reproducers/3f2a...
    sha256: sha256:...
    mediaType: application/vnd.go.fuzz-corpus
```

Every adapter record remains bound to the exact TestPlan fingerprint, subject component/revision, environment identity, producer, and run ID. Artifact paths are relative, bounded, validated, and never dereferenced by gap analysis.

`--level` is required because the native tool cannot infer Testule's intended test layer reliably. Behavior, visibility, and quality attribute are optional explicit annotations. `go test` defaults generation to `example`; `go fuzz` always records generation `fuzz`.

## Fuzz failure lifecycle

A new native fuzz failure follows this lifecycle:

```text
Go fuzz failure
  -> Go-minimized corpus entry
  -> copy + SHA-256 into .testule/artifacts/<run>/reproducers/
  -> remove the newly created package corpus entry
  -> failed normalized Evidence
  -> explicit replay
  -> optional explicit promotion
```

Go's native fuzz engine performs its own failure minimization before writing the corpus entry. Testule preserves that concrete entry and its digest.

`replay` verifies the Evidence binding, exact subject revision, artifact path, symlink policy, and digest. It stages the reproducer into the package corpus only for the bounded replay command, then removes the staged copy. Replay emits new Evidence under the explicitly supplied replay environment identity.

`promote` is intentionally separate and mutating. It verifies the same source Evidence/revision/digest and copies the reproducer into `testdata/fuzz/<FuzzTarget>/<name>` as a persistent regression corpus entry. Repeating an identical promotion is idempotent; conflicting content fails closed.

For agent/capability hosts, promotion is a consequential workspace mutation and should be separately authorized from fuzz execution or replay. Successful testing Evidence is not authority to mutate canonical source state.

## Exit codes

In addition to the base CLI exit codes:

| Code | Meaning |
| --- | --- |
| `6` | Native adapter execution completed with failed test/fuzz/replay Evidence |
| `7` | Requested Go tool or exact target is unsupported/unavailable |

Adapter configuration/path-policy errors remain explicit command failures and do not silently broaden execution authority.
