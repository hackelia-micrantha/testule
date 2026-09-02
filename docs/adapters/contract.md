# Incubating adapter semantic contract

Status: **implementation proof for #16/#18; not a stable public ABI or transport**.

Testule integrates native execution and existing tool reports through a capability-oriented adapter seam. The current proof uses three materially different shapes: the existing Go native adapter, Python standard-library `unittest` execution, and a non-executing JUnit XML importer.

## Proven common semantics

```text
describe()   -> identity + declared capabilities
probe()      -> capabilities actually available in this environment
discover()   -> optional opaque target inventory
invoke()     -> bounded terminal result + normalized Evidence
```

The implementation lives under `internal/adapter`. Its Go interfaces are an implementation vehicle for conformance testing, not the promised external adapter ABI.

### Describe

Static metadata declares adapter identity, its incubating semantic protocol version, implementation version, and supported typed capabilities. Declared support is not the same thing as runtime availability.

The proof currently uses:

- `test.execute` for Python execution;
- `evidence.import` for JUnit XML ingestion.

The operation vocabulary should expand only when another concrete adapter requires it.

### Probe

`probe` reports whether a declared capability is actually usable in the current environment. The Python probe resolves its native interpreter; an unavailable interpreter is reported unavailable and invocation returns `unsupported`. The built-in JUnit importer is available without an external executable.

This distinction prevents a configured adapter from being mistaken for a runnable capability.

### Discover is optional

Discovery is a separate optional interface. The JUnit importer does not implement it and does not fabricate targets. Explicit Python `unittest` selectors remain opaque adapter-owned identifiers in this proof; Testule core does not parse their module/class/method structure.

Future adapters may implement discovery when the native ecosystem provides useful stable target inventory.

### Invoke

The generic invocation carries Testule-domain intent and attribution:

- typed capability;
- immutable TestPlan binding: API version, plan name/fingerprint, subject component;
- exact subject revision;
- environment and run identities;
- optional opaque adapter target ID;
- Testule coverage dimensions;
- bounded namespaced adapter options;
- bounded input bytes for importer-style capabilities.

It deliberately has no executable, shell, command, script, credential, network, or effective-authority field. Native command construction belongs to an execution adapter. Host authorization remains the separate #6/#10/#13 boundary.

## Plan binding rather than plan ABI

The proof revised the initial in-process idea of handing every adapter a `*plan.TestPlan`. Adapters need immutable attribution sufficient to produce Evidence, not the complete Go authoring object. The current `PlanBinding` contains the plan API version, name, fingerprint, and subject component.

This has two useful properties:

1. adapter implementations do not need to understand TestPlan requirement structure merely to execute/import a typed operation;
2. the semantic request/result shapes serialize cleanly to JSON without making a Go interface or Go struct ABI the external contract.

The core remains responsible for authoring semantics and gap evaluation.

## Terminal state is not verification outcome

Adapter terminal status and Evidence observations are separate dimensions.

Examples proven by conformance tests:

```text
Python unittest process ran normally, test failed:
  adapter status: completed
  observation:    failed

JUnit XML parsed normally, report contains failure:
  adapter status: completed
  observation:    failed

Python interpreter absent:
  adapter status: unsupported
  observation:    none

Malformed adapter input:
  adapter status: invalid_request
  observation:    none
```

Current terminal classes are incubating:

- `completed`
- `invalid_request`
- `denied`
- `unsupported`
- `timed_out`
- `cancelled`
- `infrastructure_failed`

`invalid_request` was added by the multi-adapter proof: malformed XML, unknown adapter options, unsafe native selectors, and other caller/configuration errors are neither verification failures nor infrastructure failures. They must fail closed without being mislabeled as runtime outages.

Terminal classes must not be collapsed into Evidence's verification statuses.

## Evidence correction discovered by the proof

The Go reference adapter exposed `execution.package` in normalized Evidence. That field encoded a Go concept in the language-neutral core. The multi-adapter proof replaces it with optional `execution.scope`.

`scope` is opaque adapter provenance. The Go adapter records its exact package selector there and only the Go replay/promotion implementation interprets it as a Go package. Other adapters may omit it or use a native scope meaningful to that adapter. Core gap evaluation never interprets it.

Because the Evidence schema remains `v1alpha1`, this is an incubating schema correction rather than a compatibility promise for the removed Go-specific field.

## Python proof

The Python adapter intentionally uses only the standard library and invokes:

```text
python -m unittest <opaque-target-id>
```

Properties:

- direct process execution; no shell;
- command construction owned by the adapter;
- only `python.workspace` is accepted as a namespaced adapter option;
- target IDs are bounded, reject control characters and option-like leading `-` values, and round-trip unchanged;
- output is bounded;
- user site packages and bytecode writes are disabled and hash seeding is deterministic;
- missing Python is `unsupported`;
- malformed adapter options/selectors are `invalid_request`;
- a normal runner exit that reports test failure is terminal `completed` with failed Evidence.

This is a conformance probe, not a decision that `unittest` is Testule's permanent Python framework.

## JUnit XML importer proof

The JUnit adapter executes no native target. It consumes bounded report bytes and normalizes test cases into Evidence observations.

Properties:

- typed `evidence.import` capability only;
- no discovery interface;
- no target ID or adapter options are silently accepted;
- no execution metadata is fabricated;
- malformed or oversized XML is `invalid_request`;
- DTD/entity declarations are rejected;
- command-looking XML strings remain inert report data;
- report failures become failed observations while import remains terminal `completed`.

This proves that the common seam is not merely a generalized test runner.

## Transport spike result

The proven semantic request/result objects can be JSON serialized. No field requires a Go pointer, function, channel, native command object, or other implementation-language-only construct.

A future out-of-process transport can therefore plausibly map:

```text
Testule core
  -> versioned JSON request
  -> adapter executable
  -> versioned terminal result + normalized Evidence
```

This proof does **not** stabilize framing, lifecycle, process discovery, multiplexing, streaming, cancellation, authentication, or distribution. Those remain intentionally deferred until the semantic interface survives further adapter work.

The Unix CLI contract and future adapter transport should remain compatible in spirit—bounded machine-readable stdin/stdout with diagnostics separated—but they are not automatically the same protocol.

## Rejected assumptions

The proof rejects these assumptions from a Go-only implementation:

- every adapter executes a process;
- every adapter discovers targets;
- a package is a universal execution concept;
- nonzero verification outcome means adapter infrastructure failed;
- invalid caller input should be classified as infrastructure failure;
- adapters need the full TestPlan authoring object;
- callers should provide arbitrary native commands;
- adapter protocol and host authorization are one boundary.

## Conformance boundary

Canonical tests prove that actual Go execution Evidence, actual Python execution Evidence when Python is available, and JUnit-imported Evidence satisfy the same `level.unit` TestPlan requirement without changing TestPlan or gap semantics. Imported failures fail that same requirement without fabricating an execution event.

The next stable-protocol decision should be based on this evidence plus further adapter demand, not on the current internal Go interface alone.
