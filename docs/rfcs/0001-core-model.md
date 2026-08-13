# RFC 0001: Core Testule model

- **Status:** Draft
- **Target maturity:** incubating
- **Scope:** language-neutral model, execution boundary, evidence model, and first implementation sequence

## Summary

Testule should provide a portable testing capability layer built around declarative plans, explicit testability contracts, reusable data and environment specifications, executable scenarios, constrained capability contracts, adapters to native test tools, and normalized evidence.

The project should not replace ecosystem-native test runners. Its role is to make testing requirements and evidence portable, composable, reproducible, and safe to expose to automation and AI agents.

## Problem

Testing intent is commonly distributed across code, CI workflows, fixtures, environment scripts, fuzz harnesses, test framework configuration, and undocumented team conventions. As a result:

- test coverage is difficult to reason about across levels and behavioral classes;
- positive, negative, boundary, adversarial, and security cases are inconsistently represented;
- fuzzing and property testing are often separate experiments instead of reusable capabilities;
- generated data is frequently non-reproducible or detached from its provenance;
- test environments are implicit and differ between local and CI execution;
- failure artifacts are not consistently minimized or promoted into regression tests;
- agentic testing often receives excessive ambient authority and weak execution contracts;
- evidence is framework-specific and hard to aggregate into a trustworthy answer about capability completeness.

## Goals

1. Express test requirements independently from a particular language or runner.
2. Model the full applicable test pyramid without requiring irrelevant layers.
3. Represent black/gray/white-box visibility and positive/negative/boundary/adversarial behavior explicitly.
4. Make generated data, templates, environmental configuration, property testing, and fuzzing first-class.
5. Make failures reproducible through retained seeds, inputs, environment identity, tool versions, and artifacts.
6. Allow native testing tools to participate through adapters rather than replacement.
7. Provide bounded testing capabilities suitable for automation and AI agents.
8. Normalize evidence sufficiently to support gap analysis, quality gates, replay, and completion review.
9. Keep the initial implementation small enough to validate the model through one real ecosystem before generalizing.

## Non-goals

The first versions will not:

- provide a universal replacement test runner;
- implement every environment provider, test framework, fuzz engine, or data generator;
- own CI/CD scheduling, deployment, or release management;
- use AI judgement as the default correctness oracle;
- infer production secrets or ingest sensitive production data by default;
- guarantee hermetic execution when an adapter explicitly targets a real external dependency;
- require every test type for every repository.

## Conceptual model

```mermaid
flowchart TD
    Plan[TestPlan]
    Testability[TestabilityContract]
    Data[TestDataSpec]
    Env[TestEnvironmentSpec]
    Template[Template]
    Scenario[TestScenario]
    Capability[CapabilityContract]
    Adapter[Adapter]
    Native[Native test/fuzz/toolchain]
    Evidence[Evidence]
    Gate[Gap analysis / quality gate]

    Template --> Plan
    Template --> Data
    Template --> Env
    Plan --> Scenario
    Testability --> Scenario
    Data --> Scenario
    Env --> Scenario
    Capability --> Scenario
    Scenario --> Adapter
    Adapter --> Native
    Native --> Evidence
    Scenario --> Evidence
    Evidence --> Gate
```

## Core resources

### TestPlan

Declares **what must be verified** and which quality gates apply.

Representative concerns:

- subject/component/system under test;
- required test levels;
- behavioral classes;
- visibility model;
- generation strategies;
- quality attributes;
- execution profiles;
- coverage or mutation expectations where meaningful;
- required evidence and quality gates;
- justified omissions.

A plan must not imply that every available dimension is mandatory. It should make the applicable dimensions explicit and record intentional exclusions.

### TestabilityContract

Declares **how a subject can be controlled and observed for trustworthy testing**.

Representative capabilities:

- injectable dependencies;
- controllable clock;
- seedable randomness;
- state reset;
- filesystem virtualization;
- network virtualization;
- dependency/service substitution;
- fault injection;
- deterministic scheduling or concurrency controls;
- test identities and authorization contexts;
- observability hooks;
- invariants and externally observable properties.

This resource is intentionally about design-for-testability, not about a particular test framework.

### TestDataSpec

Declares fixtures and generated data together with provenance and reproducibility constraints.

Generation strategies should be extensible but initially include:

- canonical/example;
- seeded random;
- boundary;
- negative/invalid;
- combinatorial or pairwise;
- property-based;
- fuzz corpus;
- adversarial;
- synthetic scale;
- state-machine or event sequence.

A generated failure must retain enough information to reconstruct the exact input when the generator supports deterministic replay.

Production-derived data is outside the default trust model and requires explicit authorization, provenance, and privacy handling.

### TestEnvironmentSpec

Declares environmental state that materially affects execution.

Representative dimensions:

- OS and architecture;
- runtime/compiler/tool versions;
- services and dependencies;
- environment variables;
- ephemeral secret references;
- network policy;
- filesystem layout and lifecycle;
- locale and timezone;
- clock mode;
- random seed;
- feature flags;
- database or durable state;
- identity and authorization context;
- CPU, memory, disk, and concurrency limits;
- external service virtualization;
- cleanup and reset semantics.

Environment matrices should distinguish required compatibility combinations from sampled or risk-based combinations to avoid uncontrolled Cartesian expansion.

### Template

Templates provide reusable semantic defaults for plans, data, environments, scenarios, or capability policies.

Templates should be parameterized and composable. Composition must have explicit precedence and conflict behavior; opaque inheritance chains are undesirable.

Example categories:

- `service/http-api`;
- `security/untrusted-input`;
- `storage/postgres-integration`;
- `fuzz/parser`;
- `environment/hermetic-linux`.

### TestScenario

Composes the resources needed to describe an executable behavioral outcome.

A scenario may contain:

- subject and entrypoint;
- environment;
- data or generation strategy;
- initial state;
- actions/events;
- injected faults;
- expected assertions or invariants;
- execution budget;
- evidence requirements.

A single scenario may be executed through multiple adapters or generation strategies when the semantics remain meaningful.

### CapabilityContract

Declares a bounded testing operation suitable for a CLI, automation, or agent.

Candidate capability families:

- `testing.validate-plan`;
- `testing.inspect-gaps`;
- `testing.generate-tests`;
- `testing.generate-properties`;
- `testing.generate-data`;
- `testing.generate-fuzz-harness`;
- `testing.execute-plan`;
- `testing.execute-fuzz`;
- `testing.triage-failure`;
- `testing.minimize-reproducer`;
- `testing.propose-regression`;
- `testing.review-testability`.

Each capability invocation should be able to constrain:

- readable and writable paths;
- process execution;
- network access;
- secret access;
- environment mutation;
- execution time/resources;
- accepted adapters/tools;
- deterministic seed requirements;
- permitted output mutations;
- required evidence.

AI-specific behavior is layered on these capabilities; the core contract should remain useful for non-AI automation.

### Adapter

An adapter translates Testule resources into ecosystem-native execution and translates results back into normalized evidence.

Adapters should prefer process isolation and stable tool interfaces over in-process plugin mechanisms that create ABI or trust coupling.

The first implementation target is Go because it provides a useful vertical slice across unit tests, integration tests, coverage, static analysis, and native fuzzing without requiring Testule to invent those mechanisms.

### Evidence

Evidence is a normalized record of **what actually happened**.

Minimum candidate fields include:

- plan/scenario identity and version;
- subject revision;
- adapter and native tool versions;
- environment identity/fingerprint;
- generator and seed/corpus identity;
- start/end time and execution budget;
- result/status;
- assertions or invariant failures;
- coverage/mutation/security signals when available;
- logs/traces/artifact references;
- minimized reproducer reference;
- skipped or unsupported requirements;
- provenance and integrity metadata.

A passing adapter result is insufficient when required evidence is missing or a required TestPlan dimension was not exercised.

## Test dimensions

The model treats these as orthogonal dimensions rather than separate competing taxonomies.

### Level

- unit/component;
- contract;
- integration;
- system;
- end-to-end.

### Visibility

- black-box;
- gray-box;
- white-box.

### Behavior

- positive;
- negative;
- boundary;
- adversarial.

### Generation

- example/table-driven;
- generated;
- property-based;
- fuzz;
- model/state-machine;
- AI-assisted.

### Oracle

Preferred order when suitable:

1. deterministic assertion;
2. invariant/property;
3. differential/reference implementation;
4. metamorphic relation;
5. statistical oracle;
6. AI-assisted judgement.

AI judgement is intentionally last because it is harder to reproduce, calibrate, and treat as authoritative.

### Quality attribute

- functional correctness;
- security;
- performance/capacity;
- resilience/recovery;
- compatibility/migration;
- operational behavior.

## Fuzzing model

Fuzzing is not restricted to byte-level unit tests. Testule should be able to represent fuzzing at several boundaries:

- functions and parsers;
- serialization/protocol layers;
- API request bodies and sequences;
- state machines;
- storage inputs and migrations;
- event/order/concurrency schedules where supported;
- environmental and fault conditions.

A fuzz execution should conceptually follow:

```text
generate/mutate -> execute -> detect failure -> preserve -> minimize -> replay -> promote regression -> retain corpus
```

The first implementation only needs to prove this lifecycle for Go native fuzzing.

## Generated data model

Generated data should be declarative enough to express constraints while allowing generator-specific extensions.

Required properties for the first version:

- deterministic seed support where the underlying generator permits it;
- explicit invalid/negative generation;
- boundary generation;
- provenance of generated fixtures;
- bounded record/size/resource limits;
- no production PII or secrets by default;
- serialization suitable for replay and CI artifacts.

Schema-driven generation is desirable but the core model must not assume JSON Schema is the only source type.

## Environment model

An environment is a test input, not hidden runner state.

Environment providers may eventually include:

- local process;
- containers;
- compose-like multi-service environments;
- Kubernetes;
- VM/sandbox providers;
- external integration environments.

The first implementation should avoid building a general environment orchestrator. It should validate and fingerprint environment requirements, then introduce provisioning only where a concrete adapter requires it.

## Gap analysis

A principal user-facing capability is to compare declared requirements with discovered or produced evidence.

Conceptually:

```text
TestPlan + repository inspection + adapter capabilities + Evidence -> coverage/gap matrix
```

A matrix should be able to identify gaps such as:

- missing negative integration coverage;
- parser accepts untrusted bytes but has no fuzz target;
- TestabilityContract declares controllable time but tests use wall clock;
- required static analysis exists locally but is absent from CI;
- an end-to-end path exists but lacks failure-path validation;
- a required environment combination has never produced evidence;
- an adapter cannot satisfy a declared plan requirement.

Testule should report unsupported requirements explicitly rather than silently treating them as passing.

## Security and trust model

Testule will execute or orchestrate code, generated inputs, external tools, and potentially agent-produced artifacts. These are trust boundaries, not implementation details.

Initial requirements:

- repository content, templates, test data, adapter output, logs, and agent output are untrusted inputs;
- path traversal and unintended workspace writes must be prevented;
- secrets are referenced, not embedded in plans or evidence;
- network and process permissions should be explicit for agent-accessible capabilities;
- evidence must redact secret values and avoid copying sensitive environment data;
- generated tests or harnesses are proposals until validated by compilation, static analysis, and execution;
- adapters must declare the tools/commands they invoke and their required permissions;
- fail-open behavior is unacceptable for required quality gates;
- future remote execution requires a stronger artifact-integrity and provenance model.

A dedicated threat model should be created before Testule executes untrusted generated code in a privileged or remote environment.

## Specification and compatibility

The proposed API identifier is initially:

```text
testule.dev/v1alpha1
```

`v1alpha1` communicates that resource shape and semantics may change while the first adapters validate the model.

The initial implementation should support YAML input while defining semantics independently of YAML. JSON compatibility is desirable if the chosen schema mechanism supports it without divergent behavior.

Unknown fields, extension namespaces, template merge semantics, and validation strictness require explicit decisions before the first compatibility promise.

## First implementation slice

After this RFC is accepted, the first executable slice should deliver:

- Go module and CLI skeleton;
- a small `v1alpha1` resource envelope;
- parsing and structural validation for `TestPlan`;
- `testule validate <file>`;
- deterministic machine-readable diagnostics;
- unit tests for parsing/validation and negative cases;
- a CLI integration test for exit codes and diagnostics;
- compile/vet/static-analysis/lint entry points wired into canonical task tooling and CI;
- no adapter execution yet.

This is deliberately smaller than implementing the full domain model in one PR.

## Open decisions

The following are intentionally unresolved and should be decided using evidence from the first vertical slices rather than prematurely generalized:

1. **Schema implementation:** JSON Schema, CUE, Go-native schema generation, or another mechanism.
2. **Template composition:** merge rules, conflict detection, parameter typing, and remote template trust.
3. **Adapter protocol:** subprocess/structured stdio, JSON-RPC, gRPC, or another boundary.
4. **Extension model:** namespaced fields, typed extension resources, or adapter-owned configuration.
5. **Evidence serialization:** canonical JSON/YAML, structured directory bundle, OCI artifact, or layered model.
6. **Environment provisioning:** validation-only core versus provider interface in the initial architecture.
7. **Agent mutation policy:** which generated outputs may be written directly versus only proposed as patches/artifacts.
8. **Static repository inspection:** language-specific discovery plugins versus explicit declarations as the initial source of truth.

## Acceptance criteria for this RFC

The RFC is ready to accept when:

- the project boundary and non-goals are agreed;
- the seven core resources are coherent and ownership does not materially overlap;
- the first Go vertical slice can be implemented without deciding all future adapter/provider details;
- fuzzing, generated data, environments, templates, and agent capabilities have an explicit place in the model;
- evidence and reproducibility are first-class rather than afterthoughts;
- material trust boundaries are documented;
- unresolved decisions are explicitly deferred rather than accidentally encoded into a stable contract.
