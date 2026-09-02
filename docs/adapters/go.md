# Native Go test and fuzz adapter

The first Testule execution adapter deliberately targets a narrow, inspectable subset of the Go toolchain. It proves the adapter/Evidence boundary without defining a general plugin system. Go-specific selectors and package semantics remain inside this adapter; normalized Evidence uses language-neutral execution fields.

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

The adapter executes the requested exact target once and uses the resulting `go test -json` event stream to determine whether that target actually ran. It does not perform a preliminary `go test -list` invocation, avoiding duplicate package initialization or `TestMain` lifecycle execution. A missing Go tool or exact target is normalized as `unsupported`; compilation/setup errors, test failures, fuzz failures, and timeouts are normalized as `failed` Evidence on the legacy Go-specific result surface. The incubating cross-adapter contract in #16/#18 separates adapter terminal state from observation outcome.

## Execution bounds

Testule invokes Go directly with `exec.CommandContext`; no shell is involved. The first adapter fixes or bounds:

- wall time: greater than zero and at most 2 minutes;
- fuzz campaign time: greater than zero and at most 30 seconds, with additional timeout headroom;
- Go test parallelism: `-parallel=1`;
- captured stdout and stderr: 1 MiB each, with truncation recorded;
- retained adapter artifacts, including fuzz reproducers: 1 MiB each;
- child-process pipe shutdown: bounded with a short wait delay;
- module/toolchain downloads: disabled with `GOPROXY=off`, `GOSUMDB=off`, and `GOTOOLCHAIN=local`;
- CGO: disabled for this direct adapter;
- Go caches, temporary files, HOME, and GOPATH: scoped to a private OS temporary directory and removed after the adapter operation.

The child environment is constructed explicitly instead of inheriting ambient environment variables and secrets. Tests needing dependencies must therefore be self-contained or use dependency material already made available through an explicit future environment policy, such as vendored dependencies.

### Isolation boundary

The direct adapter does **not** claim OS-level network, CPU, memory, process-count, or syscall isolation. Test code can still use authority available to the operating-system process, including host network access and same-user workspace access. When those controls are mandatory, a capability host or TestEnvironment provider must enforce them before invoking the adapter; unsupported mandatory isolation must fail closed rather than being represented as isolated execution.

The direct adapter also treats its artifact and corpus paths defensively: directory chains are rechecked for symlinks after native execution, result files are created exclusively rather than overwritten, reported fuzz corpus entries must be regular non-symlink files, and every recorded artifact digest is revalidated immediately before Evidence is committed. Fuzz cleanup is ownership-scoped: only the concrete reproducer paths reported by this invocation's Go event stream are captured and removed; unrelated files created concurrently in the same corpus directory are left untouched. These controls prevent ordinary path-substitution, stale-artifact, and cross-run cleanup mistakes; they do not turn same-user native test code into a hostile-code sandbox.

## Evidence

Adapter-produced Evidence adds optional execution and artifact metadata to the normalized Evidence envelope:

```yaml
execution:
  adapter: go-native/v1alpha1
  operation: fuzz
  tool: go
  toolVersion: go1.26.5
  scope: ./parser
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

`execution.scope` is language-neutral adapter provenance. The Go adapter maps its exact package selector into that field; Python, JVM, JavaScript, analyzers, or importers are not required to model their native structure as a Go package. Replay interprets `scope` only after verifying that the source Evidence was produced by the Go adapter.

Every adapter record remains bound to the exact TestPlan fingerprint, subject component/revision, environment identity, producer, and run ID. Artifact paths are relative, bounded, validated, and never dereferenced by gap analysis.

`--level` is required because the native tool cannot infer Testule's intended test layer reliably. Behavior, visibility, and quality attribute are optional explicit annotations. `go test` defaults generation to `example`; `go fuzz` always records generation `fuzz`.

## Fuzz failure lifecycle

A native fuzz failure follows this lifecycle:

```text
Go fuzz failure
  -> Go reports the concrete minimized corpus entry
  -> copy + SHA-256 into .testule/artifacts/<run>/reproducers/
  -> remove only that reported package corpus entry
  -> failed normalized Evidence
  -> explicit replay
  -> optional explicit promotion
```

Go's native fuzz engine performs its own failure minimization before writing the corpus entry. Testule preserves the concrete reported entry and its digest. Testule does not classify arbitrary "new since snapshot" corpus files as its own output; unrelated concurrent corpus writes remain untouched.

`replay` verifies the Evidence binding, exact subject revision, Evidence/artifact path symlink policy, digest, Go adapter identity, and the adapter-owned Go package encoded as the generic execution scope. It stages the reproducer into the package corpus only for the bounded replay command, then removes the staged copy. Replay emits new Evidence under the explicitly supplied replay environment identity.

`promote` is intentionally separate and mutating. It verifies the same source Evidence/revision/digest and copies the reproducer into `testdata/fuzz/<FuzzTarget>/<name>` as a persistent regression corpus entry. Repeating an identical promotion is idempotent; conflicting content fails closed. Promotion uses exclusive creation for a new corpus entry so a path that changes concurrently is not silently overwritten.

For agent/capability hosts, promotion is a consequential workspace mutation and should be separately authorized from fuzz execution or replay. Successful testing Evidence is not authority to mutate canonical source state.

## Exit codes

In addition to the base CLI exit codes:

| Code | Meaning |
| --- | --- |
| `6` | Native adapter execution completed with failed test/fuzz/replay Evidence |
| `7` | Requested Go tool or exact target is unsupported/unavailable |

Adapter configuration/path-policy errors remain explicit command failures and do not silently broaden execution authority.
