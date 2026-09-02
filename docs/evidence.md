# Evidence and gap analysis

Testule treats evidence as a generated record of what was actually observed, not as user-authored proof that a requirement passed.

## Commands

```text
testule fingerprint <plan.yaml>
testule gaps [--format text|json] --subject-revision <revision> <plan.yaml> [evidence.yaml ...]
```

`fingerprint` returns a deterministic SHA-256 fingerprint of the decoded TestPlan. Evidence records bind to that fingerprint, the plan name, the subject component, and the tested subject revision.

`gaps` validates the plan and every supplied Evidence document before evaluation. It does not discover tests, execute adapters, dereference artifact references, access the network, or mutate the workspace.

Exit code `5` means evaluation completed successfully but at least one required requirement is unsatisfied or an observed optional requirement failed. Invalid plans/evidence continue to use exit code `3`.

## Evidence envelope

A minimal Evidence record is:

```yaml
apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: parser-run
plan:
  name: parser
  fingerprint: sha256:...
subject:
  component: parser
  revision: git-sha-or-other-revision
environment:
  id: linux-amd64-go1.26
provenance:
  producer: go-test
  runId: ci-run-123
  references:
    - artifact://junit.xml
observations:
  - id: parser-negative
    status: passed
    coverage:
      levels: [unit]
      behaviors: [negative]
      generation: [fuzz]
      visibility: [whiteBox]
      qualityAttributes: [security]
```

Evidence files are bounded to 1 MiB. Provenance references are bounded opaque strings; Testule does not dereference them in this slice.

### Observation status

Supported observation statuses are:

- `passed`
- `failed`
- `skipped`
- `unsupported`

These are verification outcomes, **not adapter/process terminal states**. The incubating adapter contract may report an operation as `completed` while its Evidence contains one or more `failed` observations. Conversely, `invalid_request`, `timed_out`, or `infrastructure_failed` terminal adapter results need not fabricate observations at all. See [Incubating adapter semantic contract](adapters/contract.md).

`inapplicable` is deliberately not an Evidence status. Applicability is a requirement decision and therefore belongs to the TestPlan with a rationale.

### Optional execution provenance

Evidence produced by an execution adapter may include an `execution` block. Importers and analyzers that do not execute a native target are not required to fabricate one.

```yaml
execution:
  adapter: go-native/v1alpha1
  operation: test
  tool: go
  toolVersion: go1.26.5
  scope: ./parser
  target: TestDecode
  command: [go, test, ...]
  exitCode: 0
  durationMillis: 142
```

`execution.scope` and `execution.target` are optional opaque adapter provenance. Core TestPlan authoring and gap evaluation do not interpret ecosystem-specific selector syntax. For example, the Go adapter stores its package selector in `scope`, while Python execution can omit `scope` and retain its native test ID as an opaque `target`.

The `execution` block records what an adapter actually invoked; it is not a caller-controlled generic command envelope and does not grant execution authority.

## Plan requirements

This slice extends the alpha TestPlan with generation requirements and explicit inapplicability:

```yaml
requirements:
  levels:
    unit: required
    integration: required
  behaviors:
    negative: required
  generation:
    fuzz: required
  inapplicable:
    - dimension: level
      value: endToEnd
      rationale: No deployed external system exists for this component.
```

An inapplicable entry cannot also be declared required or optional.

## Gap states

Each declared requirement resolves deterministically to one of:

- `satisfied`
- `missing`
- `unsupported`
- `skipped`
- `failed`
- `inapplicable`

A required requirement is complete only when it is `satisfied`. A required unsupported capability is therefore a gap, never a pass.

If multiple current-revision evidence observations cover the same requirement, state precedence is fail-closed:

1. `failed`
2. `passed` -> `satisfied`
3. `skipped`
4. `unsupported`
5. no evidence -> `missing`

A failure blocks completeness even when the underlying plan requirement was optional. Optional missing/skipped/unsupported requirements do not block completeness.

## Identity and provenance

Evidence is rejected before evaluation when any of these differ from the requested evaluation:

- plan name;
- plan fingerprint;
- subject component;
- subject revision.

Evidence `metadata.name` must also be unique across supplied files, and observation IDs must be unique within an Evidence record.

This prevents evidence for a stale revision, a different plan, or a different component from silently satisfying current requirements.

## Orthogonal dimensions and an important limitation

Testule models testing dimensions independently. In this slice, a requirement such as `level.integration` and a separate `behavior.negative` are evaluated independently.

Evidence can record that a single observation covered both:

```yaml
coverage:
  levels: [integration]
  behaviors: [negative]
```

However, the evaluator does **not** infer a required cross-product such as “negative integration testing” merely because the plan independently requires `integration` and `negative`. Cross-dimensional requirements belong in a future scenario/requirement model where the relationship is explicit.

This avoids overstating coverage.

## Coverage percentages

Line/branch coverage, mutation score, and similar metrics can become Evidence signals later, but they do not substitute for requirement evidence. A high coverage percentage does not prove that a declared negative, fuzz, resilience, or security property was exercised.
