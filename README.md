# Testule

Reusable testability contracts, test plans, generated data, environments, fuzzing, and agent-accessible testing capabilities.

> **Status:** incubating. RFC 0001 defines the accepted core model; executable interfaces remain intentionally narrow and versioned as `v1alpha1`.

## Why Testule

Testing concerns are usually fragmented across framework-specific tests, CI configuration, fixture directories, fuzz harnesses, environment scripts, and agent prompts. That makes it difficult to answer a more important question: **what evidence exists that a declared behavior, boundary, or safety property is actually verified?**

Testule aims to provide a language-neutral capability layer that can describe, compose, execute, and evaluate testing requirements while delegating execution to ecosystem-native tools.

The project is intentionally **not another universal test runner**. Native tools remain authoritative for execution; Testule provides portable specifications, orchestration contracts, adapters, and normalized evidence.

## Core model

Testule treats testing as multiple independent dimensions rather than forcing every concern into a single hierarchy.

| Dimension | Examples |
| --- | --- |
| Test level | unit/component, contract, integration, system, end-to-end |
| Visibility | black-box, gray-box, white-box |
| Behavior | positive, negative, boundary, adversarial |
| Generation | example-based, generated, property-based, fuzz, model-based, AI-assisted |
| Oracle | assertion, invariant, differential, metamorphic, statistical, AI-assisted |
| Quality attribute | functional, security, performance, resilience, compatibility |
| Execution | local, CI, scheduled, pre-release, continuous |
| Evidence | coverage, minimized reproducer, trace, mutation score, artifact, result |

The initial domain model separates authoring resources from execution mechanisms and evidence:

### Authoring resources

- **TestPlan** — declares required validation and quality gates.
- **TestabilityContract** — declares controllable seams and observable properties exposed by a system under test.
- **TestDataSpec** — declares fixtures, generators, provenance, constraints, and reproducibility requirements.
- **TestEnvironmentSpec** — declares runtime, services, network, filesystem, clock, randomness, secrets, resource limits, and reset behavior.
- **Template** — provides parameterized, composable defaults for plans, data, environments, and scenarios.
- **TestScenario** — composes plan, data, environment, faults, actions, and expected properties into an executable outcome.

### Execution and governance

- **CapabilityContract** — bounds testing operations for humans, automation, and agents.
- **Adapter** — maps Testule intent to ecosystem-native tools and translates native results back into Testule evidence. It is an execution boundary, not a required user-authored resource.

### Evidence

- **Evidence** — a generated, versioned record of what ran, under which inputs and environment, with which result and artifacts.

See [RFC 0001](docs/rfcs/0001-core-model.md) for the accepted core model and [RFC 0002](docs/rfcs/0002-evidence-gap-analysis.md) for the current Evidence and gap-analysis semantics.

## Principles

### Language-agnostic core

Testule's product contract is language- and framework-agnostic. `TestPlan`, testability, data, environment, scenario, capability, Evidence, and gap semantics must not depend on Go, Rust, JVM, Python, JavaScript/TypeScript, .NET, mobile, or any other ecosystem.

Go is the bootstrap implementation language and the first reference adapter only. Native package conventions, target syntax, commands, corpus layouts, tool-specific result parsing, and replay mechanics belong behind an adapter boundary or an explicitly namespaced adapter extension. Adding another adapter must not require changing the meaning of core Testule requirements or gap states.

The incubating `testule go ...` CLI is therefore an adapter-specific convenience surface, not the permanent cross-language execution architecture. A stable generic adapter contract should be derived only after at least two materially different ecosystems have proven the common requirements. See issue #16.

### Evidence over test count

Coverage percentages and test counts are supporting signals. The primary question is whether declared behavior, failure modes, boundaries, and safety properties have trustworthy evidence.

### Full pyramid, risk-based

Testule can model unit/component, contract, integration, system, and end-to-end validation, plus negative, security, compatibility, migration, performance, and operational testing. Layers are required by risk and boundaries, not mechanically.

### Fuzzing is first-class

Fuzzing is a generation/execution strategy that may apply at multiple layers: functions, parsers, protocols, APIs, state machines, integration boundaries, event sequences, and fault conditions. A fuzz failure should preserve the concrete failing input or corpus artifact, minimize the reproducer when possible, retain deterministic seeds when the underlying tool exposes them, and be promotable into a permanent regression case.

### Data and environments are inputs

Generated data, fixtures, environment configuration, clocks, randomness, service dependencies, feature flags, resource limits, and fault injection are modeled explicitly so failures can be reproduced rather than inferred from hidden CI state.

### Native tools remain native

Adapters should integrate tools such as `go test`, native Go fuzzing, cargo test/cargo-fuzz, JUnit/Jazzer, pytest/Hypothesis/Atheris, Vitest/Playwright/fast-check, and comparable ecosystem tools instead of replacing them.

### Agents receive capabilities, not ambient authority

AI-assisted testing should operate through explicit contracts such as generating candidate tests, producing fuzz harnesses, analyzing gaps, minimizing failures, or proposing regression cases. Capability contracts must bound filesystem, network, secret, process, and mutation access and record resulting evidence.

### Determinism where possible

Time, randomness, generated data, service state, and environment configuration should be reproducible. When nondeterminism is intentional, it should be declared and enough evidence retained to investigate failures.

## Current `v1alpha1` CLI

```text
testule validate [--format text|json] <plan.yaml>
testule fingerprint <plan.yaml>
testule gaps [--format text|json] --subject-revision <revision> <plan.yaml> [evidence.yaml ...]
testule go <test|fuzz|replay|promote> ...
```

The current TestPlan supports level, behavior, and generation requirements plus explicitly justified inapplicable requirements. Evidence records bind to a TestPlan fingerprint and subject revision before they can satisfy requirements.

Gap evaluation distinguishes `satisfied`, `missing`, `unsupported`, `skipped`, `failed`, and `inapplicable`. A required unsupported capability is never treated as passing.

The first native adapter executes exact local Go tests and bounded Go fuzz targets, emits normalized Evidence, preserves native fuzz reproducers as digested Testule artifacts, supports bounded replay, and keeps persistent regression promotion explicit. See [Native Go test and fuzz adapter](docs/adapters/go.md).

The adapter does not claim host network/CPU/memory isolation. Those controls remain the responsibility of a capability host or future TestEnvironment provider when required.

See [Evidence and gap analysis](docs/evidence.md) for the current Evidence and evaluator contract.

## Intended users

- maintainers who want a reusable test plan and gap model across repositories;
- platform and DevEx teams standardizing validation without replacing native toolchains;
- security engineers applying negative, adversarial, fuzz, and fault-oriented testing;
- CI/CD systems that need normalized test requirements and evidence;
- agents that need constrained, auditable testing capabilities rather than unrestricted shell access.

## Non-goals

Testule does not initially aim to:

- implement replacements for language-native test frameworks;
- own CI/CD scheduling or deployment;
- decide business correctness without an authoritative contract or oracle;
- make AI output authoritative simply because it was generated by a model;
- store production secrets or sensitive production-derived data;
- require every repository to implement every possible test layer.

## Initial delivery sequence

1. Define and version the language-agnostic core specification and conformance rules.
2. Bootstrap the reference CLI implementation and strict minimal TestPlan validation with canonical local/CI validation and static analysis. The current implementation language is Go, but this is not a product-language constraint.
3. Add normalized evidence and a test-coverage/gap matrix.
4. Deliver the first reference adapter vertical slice using Go tests and native fuzzing, including replayable failure evidence and regression promotion.
5. Define and harden agent-accessible capability contracts.
6. Add generated-data, environment, template, and scenario resources against the proven core model.
7. Prove the language-neutral adapter boundary with at least one materially different second ecosystem before stabilizing a generic adapter protocol; then expand adapters, environment providers, mutation testing, fault injection, and additional generators based on demonstrated use cases.

The issue tracker is the source of truth for executable work and priority.

## License

Apache License 2.0. See [LICENSE](LICENSE).
