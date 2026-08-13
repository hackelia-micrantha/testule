# RFC 0002: Evidence and gap-analysis semantics

- **Status:** Accepted
- **Target maturity:** `v1alpha1`
- **Scope:** normalized Evidence, plan identity, requirement applicability, and deterministic gap evaluation

## Summary

Testule needs a portable answer to a narrow question before it executes native adapters:

> Given a TestPlan and a set of supplied Evidence records for one subject revision, which declared requirements are satisfied and which remain gaps?

This RFC defines the first normalized Evidence envelope and deterministic evaluator. It extends the alpha TestPlan only where needed to represent generation requirements and explicitly justified inapplicable requirements.

It does not define test discovery, adapter execution, artifact storage, coverage-percentage policy, or cross-dimensional scenario semantics.

## Decisions

### Evidence is generated output

`Evidence` is a versioned generated record. It is not a TestPlan resource that users can author to waive requirements.

Each Evidence record binds to:

- TestPlan name;
- deterministic TestPlan fingerprint;
- subject component;
- subject revision;
- environment identity;
- producer/run provenance.

Evidence whose plan identity, fingerprint, component, or revision does not match the current evaluation is rejected before it can satisfy a requirement.

### Evidence observations

An Evidence record contains one or more observations. An observation has:

- a unique ID within the record;
- execution status;
- one or more coverage values.

Supported execution status in this slice:

- `passed`;
- `failed`;
- `skipped`;
- `unsupported`.

Supported coverage dimensions represented by Evidence:

- level;
- behavior;
- generation;
- visibility;
- quality attribute.

The first evaluator currently compares plan requirements for level, behavior, and generation. Visibility and quality attributes are retained in normalized Evidence so later TestPlan/scenario slices do not need a new evidence shape merely to represent them.

### Applicability is plan-authored

`inapplicable` is not an Evidence status.

A TestPlan may explicitly mark a supported requirement value inapplicable with a non-empty rationale:

```yaml
requirements:
  inapplicable:
    - dimension: level
      value: endToEnd
      rationale: No deployed external system exists for this component.
```

The same dimension/value cannot simultaneously be required/optional and inapplicable.

This keeps requirement waivers on the policy/authoring side rather than allowing test output to declare itself exempt.

### Generation requirements

The alpha TestPlan adds the `generation` requirement group with:

- `example`;
- `generated`;
- `property`;
- `fuzz`;
- `model`;
- `aiAssisted`.

As with existing level and behavior requirements, generation values are `required` or `optional`.

### Deterministic gap states

Each declared requirement evaluates to exactly one:

- `satisfied`;
- `missing`;
- `unsupported`;
- `skipped`;
- `failed`;
- `inapplicable`.

For multiple observations covering one current requirement, precedence is:

1. any `failed` observation -> `failed`;
2. otherwise any `passed` observation -> `satisfied`;
3. otherwise any `skipped` observation -> `skipped`;
4. otherwise any `unsupported` observation -> `unsupported`;
5. otherwise -> `missing`.

This is deliberately fail-closed when current-revision evidence disagrees.

A required requirement is complete only when `satisfied`. In particular, required `unsupported` is a gap.

An observed `failed` requirement blocks completeness even when the TestPlan marks that requirement optional. Optional missing/skipped/unsupported requirements do not block completeness.

### Orthogonal dimensions remain orthogonal

The evaluator must not overclaim cross-dimensional coverage.

If a plan independently requires:

```text
level.integration
behavior.negative
```

separate evidence can satisfy those independent requirements. That does **not** mean the plan required or proved the combined property “negative integration testing.”

Evidence may record a single observation covering both dimensions, but cross-dimensional requirements must later be represented explicitly through scenarios or another relationship-aware requirement model.

### Plan fingerprint

The plan fingerprint is SHA-256 over the deterministic JSON serialization of the decoded TestPlan and is represented as:

```text
sha256:<64 lowercase hexadecimal characters>
```

The fingerprint is an alpha identity mechanism, not a permanent content-addressing/storage protocol.

### Bounded and inert references

Evidence input is treated as untrusted:

- one YAML document per file;
- strict unknown-field rejection;
- 1 MiB maximum file size;
- bounded observations, coverage values, provenance references, and strings;
- provenance references are opaque strings and are not dereferenced by the evaluator;
- no network access is performed.

## CLI

```text
testule fingerprint <plan.yaml>
testule gaps [--format text|json] --subject-revision <revision> <plan.yaml> [evidence.yaml ...]
```

`gaps` may be run with no Evidence files to show the fully missing baseline.

Exit status:

- `0`: evaluation completed and no blocking gaps remain;
- `3`: plan/evidence is invalid, stale, contradictory, or mismatched;
- `4`: input could not be read or exceeded its bound;
- `5`: evaluation completed and blocking gaps/failures remain.

## Evidence references

Evidence carries provenance references rather than embedding arbitrary native artifacts. Raw JUnit/SARIF/coverage/fuzz artifacts can remain native artifacts and be linked from Evidence.

A later adapter/storage RFC may standardize artifact bundles, integrity metadata, or OCI storage. This RFC intentionally does not.

## Non-goals

This slice does not:

- inspect source repositories to discover tests;
- execute native test tools;
- define adapter protocols;
- define remote evidence storage;
- make line/branch coverage percentages sufficient proof of requirements;
- infer cross-dimensional requirements;
- define evidence freshness by wall-clock age;
- allow evidence to waive requirements.

## Consequences

The next Go adapter can target a concrete normalized Evidence contract rather than inventing result semantics while executing tests.

The evaluator also creates an explicit gap between “the tool ran” and “the plan requirement is satisfied,” which is central to Testule's evidence-first model.
