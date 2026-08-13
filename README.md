# Testule

Reusable testability contracts, test plans, generated data, environments, fuzzing, and agent-accessible testing capabilities.

> **Status:** incubating. The core model is being defined before executable interfaces are treated as stable.

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

The initial domain model contains:

- **TestPlan** — declares required validation and quality gates.
- **TestabilityContract** — declares controllable seams and observable properties exposed by a system under test.
- **TestDataSpec** — declares fixtures, generators, provenance, constraints, and reproducibility requirements.
- **TestEnvironmentSpec** — declares runtime, services, network, filesystem, clock, randomness, secrets, resource limits, and reset behavior.
- **TestScenario** — composes plan, data, environment, faults, actions, and expected properties into an executable outcome.
- **CapabilityContract** — exposes bounded testing operations to humans, automation, and agents.
- **Evidence** — normalizes what ran, under which inputs and environment, with which result and artifacts.

See [RFC 0001](docs/rfcs/0001-core-model.md) for the proposed model and open decisions.

## Principles

### Evidence over test count

Coverage percentages and test counts are supporting signals. The primary question is whether declared behavior, failure modes, boundaries, and safety properties have trustworthy evidence.

### Full pyramid, risk-based

Testule can model unit/component, contract, integration, system, and end-to-end validation, plus negative, security, compatibility, migration, performance, and operational testing. Layers are required by risk and boundaries, not mechanically.

### Fuzzing is first-class

Fuzzing is a generation/execution strategy that may apply at multiple layers: functions, parsers, protocols, APIs, state machines, integration boundaries, event sequences, and fault conditions. A fuzz failure should preserve its seed and environment, minimize the reproducer when possible, and be promotable into a permanent regression case.

### Data and environments are inputs

Generated data, fixtures, environment configuration, clocks, randomness, service dependencies, feature flags, resource limits, and fault injection are modeled explicitly so failures can be reproduced rather than inferred from hidden CI state.

### Native tools remain native

Adapters should integrate tools such as `go test`, native Go fuzzing, cargo test/cargo-fuzz, JUnit/Jazzer, pytest/Hypothesis/Atheris, Vitest/Playwright/fast-check, and comparable ecosystem tools instead of replacing them.

### Agents receive capabilities, not ambient authority

AI-assisted testing should operate through explicit contracts such as generating candidate tests, producing fuzz harnesses, analyzing gaps, minimizing failures, or proposing regression cases. Capability contracts must bound filesystem, network, secret, process, and mutation access and record resulting evidence.

### Determinism where possible

Time, randomness, generated data, service state, and environment configuration should be reproducible. When nondeterminism is intentional, it should be declared and enough evidence retained to investigate failures.

## Example direction

```yaml
apiVersion: testule.dev/v1alpha1
kind: TestPlan
metadata:
  name: parser

subject:
  component: parser

requirements:
  levels:
    unit: required
    integration: required
  behaviors:
    positive: required
    negative: required
    boundary: required
  generation:
    propertyBased: required
    fuzz:
      required: true
      minimumDuration: 60s

qualityGates:
  fuzz:
    crashes: 0
    sanitizerFailures: 0
```

The exact schema remains draft until the core model RFC is accepted.

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

1. Define and version the core specification and conformance rules.
2. Bootstrap the Go CLI and canonical validation/static-analysis entry points.
3. Implement plan validation and a test-coverage/gap matrix.
4. Deliver the first adapter vertical slice using Go tests and native fuzzing.
5. Add normalized evidence, deterministic replay, and regression promotion.
6. Define and harden agent-accessible capability contracts.
7. Expand adapters, templates, environment providers, mutation testing, and fault injection based on demonstrated use cases.

The issue tracker is the source of truth for executable work and priority.

## License

Apache License 2.0. See [LICENSE](LICENSE).
