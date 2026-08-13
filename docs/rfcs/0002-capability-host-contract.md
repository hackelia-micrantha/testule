# RFC 0002: Capability invocation and host integration

- **Status:** Draft
- **Target maturity:** incubating
- **Scope:** host-neutral testing capability invocation, result, authority, and reference-host integration
- **Related:** RFC 0001, #6, #10, `ryjen/dubnium#524`, `ryjen/dubnium#413`, `ryjen/dubnium#677`

## Summary

Testule defines testing-domain capabilities that may be invoked by a CLI, CI system, deterministic automation, or AI-capable host. Testule does not own a general agent runtime, model router, governance engine, or privileged-effect gateway.

This RFC separates three protocol concepts:

1. **CapabilityContract** — the stable definition of a testing operation and its maximum authority;
2. **CapabilityInvocation** — one bounded request to perform that operation;
3. **CapabilityResult** — the bounded outcome, including Testule Evidence and any proposed/generated artifacts.

The protocol is host-neutral. Dubnium is the first reference capability host, but Testule remains independently usable and must not depend on Dubnium for ordinary CLI or CI operation.

A central rule is that a semantic testing capability is not automatically a consequential governed effect. AI participation does not change that classification. A generated test, review, or failure triage result may remain a proposal; a later action that mutates protected state, accesses protected secrets/network targets, or performs another consequential effect may require a separate host-governed operation.

## Motivation

RFC 0001 establishes `CapabilityContract` as the execution/governance boundary for Testule operations. A concrete host integration requires additional precision without turning host implementation details into Testule's public authoring model.

The design must avoid several failure modes:

- allowing an agent or caller to self-assert identity or permissions;
- giving generated code ambient shell, filesystem, network, secret, or device authority;
- conflating a semantic operation with authorization to apply its result;
- making Testule responsible for model selection, agent orchestration, or organization policy;
- forcing every Testule operation through a privileged-effect gateway;
- collapsing testing evidence, runtime evidence, and governance evidence into one ambiguous record;
- coupling Testule to a single host such as Dubnium.

## Terminology and ownership

### CapabilityContract

A `CapabilityContract` defines a testing-domain operation independent of a particular invocation.

It owns at minimum:

- capability identity and version;
- input and output contract;
- supported subject/target classes;
- maximum filesystem/process/network/secret/device authority;
- maximum resource envelope;
- permitted mutation class;
- supported adapters/tools where material;
- required Testule Evidence;
- failure, denial, unsupported, timeout, and partial-result semantics.

A capability contract defines an upper bound. It is not proof that a caller has that authority.

### CapabilityInvocation

A `CapabilityInvocation` represents one request to execute a capability.

It binds:

- capability identity/version;
- exact subject identity and revision where applicable;
- operation-specific input;
- requested authority/resource envelope;
- host-supplied caller/run context reference;
- relevant plan/scenario/environment/data references;
- replay/idempotency identity where required;
- requested evidence/output profile.

The operation payload must not be able to assert its own effective actor identity or authorization. A host may attach caller/run context outside model-controlled fields.

### CapabilityResult

A `CapabilityResult` represents one completed or terminal invocation outcome.

It contains or references:

- invocation identity;
- capability identity/version;
- exact subject revision;
- terminal status;
- normalized Testule Evidence;
- generated/proposed artifacts and their trust state;
- bounded diagnostics;
- explicit denial, unsupported, timeout, partial, or infrastructure-failure classification;
- provenance linking the result to the adapter/tool/runtime identities used.

`CapabilityResult` is not itself authorization to apply or promote an artifact.

### Evidence

Testule `Evidence` remains the authoritative testing-domain record of what was observed or verified. A host may produce separate runtime, policy, approval, or governance evidence and link it by immutable identity/digest. Those records should not be flattened into one ownerless envelope.

## Authority model

Effective authority is fail-closed and intersectional:

```text
effective authority =
  requested authority
  ∩ capability maximum
  ∩ host/principal authority
  ∩ repository policy
  ∩ environment policy
```

Any layer may narrow or reject. No layer may widen an upstream restriction.

The caller may request fewer resources or permissions than a capability supports. It cannot obtain greater authority by putting broader permissions in the invocation payload.

### Default posture

For agent-accessible capabilities, default to:

- network denied unless explicitly required;
- secret access denied unless explicitly required;
- process execution denied or allowlisted to the operation's reviewed tools;
- filesystem writes restricted to explicit workspace/artifact paths;
- device access denied unless explicitly required;
- bounded wall time, CPU, memory, disk, process count, and output size where enforceable;
- failure before execution when a mandatory bound cannot be enforced.

## Capability families

Initial candidate operations remain those established in RFC 0001:

- `testing.validate-plan`;
- `testing.inspect-gaps`;
- `testing.review-testability`;
- `testing.generate-tests`;
- `testing.generate-properties`;
- `testing.generate-data`;
- `testing.generate-fuzz-harness`;
- `testing.execute-plan`;
- `testing.execute-fuzz`;
- `testing.triage-failure`;
- `testing.minimize-reproducer`;
- `testing.propose-regression`.

The capability namespace identifies testing semantics, not the host implementation or model backend. Names such as `openai.generate-tests`, `dubnium.fuzz`, or `llama.execute-plan` are therefore inappropriate as stable Testule capability identities.

## Semantic operations versus governed effects

Testule does not define a universal privileged-effect taxonomy.

A host should distinguish semantic/domain operations from consequential effects according to the concrete authority and side effects involved.

Typical semantic operations include:

- plan validation and gap inspection;
- testability review;
- generation of tests, properties, data, or fuzz harnesses as proposals;
- failure triage;
- reproducer minimization;
- regression proposal generation.

Execution-oriented operations such as `testing.execute-plan` and `testing.execute-fuzz` require explicit sandbox/resource policy. They are not automatically privileged effects merely because they execute code; classification depends on what authority the host grants and what state can be affected.

Consequential follow-on operations may include:

- applying generated changes to protected or canonical source state;
- pushing, opening/merging a PR, releasing, or deploying;
- accessing protected credentials, production systems, or restricted network targets;
- provisioning or mutating privileged environments;
- changing organization policy or authorization state.

Those operations should cross the host's effect/governance boundary rather than being hidden inside a Testule semantic capability.

## Generated artifact trust states

Generated artifacts are proposals, not trusted repository state.

A host and Testule implementation should be able to distinguish at least:

```text
proposed
  -> structurally validated
  -> compile/type-check/static-analysis validated where applicable
  -> bounded-execution validated
  -> eligible for separately authorized mutation/promotion
```

No trust transition automatically grants mutation authority.

A generation capability may emit a proposed patch, file set, fuzz corpus, fixture, property, or harness. The result should preserve provenance and exact subject revision so stale proposals cannot silently apply to a different source state.

Before any mutation or promotion consumes a proposal, the mutation boundary must compare the current subject identity/revision or equivalent content digest with the revision bound into the `CapabilityResult`. A mismatch must fail closed: the proposal is rejected as stale or explicitly revalidated/regenerated against the new subject and produces new attributable evidence. A caller cannot rebind an old result to newer source state by assertion.

## Host-neutral provider seam

The first integration seam should remain deliberately small.

A host needs to be able to:

1. discover supported capability identities/versions and their maximum requirement profile;
2. construct a typed `CapabilityInvocation` with trusted host context bound outside caller-controlled fields;
3. invoke Testule through a stable machine-readable local boundary;
4. receive and validate a typed `CapabilityResult`;
5. link Testule Evidence into host runtime/governance lineage without changing Testule evidence semantics.

The first implementation may use a CLI/subprocess JSON interface. A permanent RPC/plugin protocol is deferred until at least one real host integration proves the requirements.

## Reference integration: Dubnium

Dubnium is the first reference host because it already separates platform capabilities, semantic capabilities, bounded agent execution, and governed effects.

Reference architecture:

```text
opencode / Codex / Supervisor / CI
  -> Dubnium testing semantic capability adapter
       -> Testule CapabilityInvocation
       -> Testule domain operation
       -> Testule adapter/native tool
       -> CapabilityResult + Evidence + proposed artifacts
```

Where mutable generated-test execution is needed, Dubnium should compose with its bounded reversible agent-run execution boundary (`ryjen/dubnium#413`) rather than Testule inventing another general worktree/sandbox lifecycle.

Where a consequential effect is required:

```text
Testule proposal/result
  -> typed governed-effect request
  -> Dubnium Capability Gateway
  -> Anthesis exact decision
  -> provider / narrow worker
```

This preserves Dubnium's invariant that a semantic capability does not imply governed-effect authority.

### Dubnium owns

- caller/transport/run identity;
- model selection and AI entrypoint routing;
- Supervisor/specialist orchestration where used;
- execution sandbox and host resource enforcement;
- exact repository/source binding at the runtime boundary;
- integration with bounded reversible runs;
- links to runtime/governance evidence;
- classification and routing of consequential follow-on effects.

### Testule owns

- testing plans, testability, scenarios, data, and environment semantics;
- testing capability semantics;
- native test/fuzz/static-analysis adapters;
- testing-specific validation, replay, and minimization;
- normalized Testule Evidence;
- generated testing artifacts and their testing-domain provenance.

### Anthesis / Capability Gateway own when applicable

- exact authorization for consequential governed effects;
- approvals, constraints, expiry, and governance evidence requirements;
- governed-effect request identity, canonicalization, idempotency, and dispatch.

Testule must not become a second governance authority, and Dubnium must not copy Testule's testing-domain model into its Supervisor or Capability Gateway.

## Security requirements

- Caller/model output cannot assert effective actor identity.
- Caller/model output cannot widen filesystem, process, network, secret, device, mutation, or resource authority.
- Repository content, test data, templates, tool output, logs, and generated artifacts are untrusted inputs.
- Prompt/tool-output injection cannot add capabilities, destinations, credentials, or mutation authority.
- Generated code is not executed merely because generation succeeded.
- Required compile/static-analysis/bounded-execution validation fails closed.
- A proposal cannot be promoted or applied when its bound subject revision no longer matches current source state unless it is explicitly revalidated/regenerated and new evidence is produced.
- Path traversal, symlink escape, workspace escape, command injection, and output-size/resource exhaustion are explicit negative-test targets.
- Secret values are not copied into plans, capability payloads, evidence, or diagnostics; references and redaction are used instead.
- Unsupported mandatory isolation/resource constraints cause rejection before execution.
- Testule Evidence success never authorizes an unrelated or consequential host effect.

## Compatibility and serialization

This RFC does not require `CapabilityInvocation` or `CapabilityResult` to become ordinary `testule.dev/v1alpha1` authoring resources.

The first machine-readable protocol should be explicitly versioned independently enough that host integrations can detect incompatible changes. The exact envelope/schema mechanism is deferred to #6/#10 implementation work.

The following are compatibility-sensitive and must be versioned deliberately:

- capability identity and operation input semantics;
- terminal result/failure classifications;
- authority/resource requirement fields;
- subject/revision binding semantics;
- artifact trust-state semantics;
- evidence linkage semantics.

Host-specific identities, model names, service names, policy documents, or privileged-effect schemas must not leak into the stable Testule capability contract.

## Validation strategy

Before a live AI or privileged integration, provide deterministic fixtures for:

- a read-only operation such as `testing.validate-plan` or `testing.inspect-gaps`;
- caller identity spoofing rejection;
- requested authority wider than host/capability policy;
- denied filesystem/network/process/secret/device access;
- unsupported mandatory resource limit;
- stale subject revision;
- stale proposal promotion after subject revision changes, including reject and explicit revalidation paths;
- generated artifact returned as proposal without canonical mutation;
- malformed/oversized result/evidence;
- explicit denied/unsupported/timeout/partial status;
- evidence linkage without collapsing host/runtime/governance ownership.

A Dubnium reference fixture must require no model, GPU, external network, real Anthesis service, or privileged host mutation.

## Implementation sequence

1. Define the minimal versioned invocation/result envelopes in #6/#10.
2. Implement deterministic contract fixtures and a fake host/provider seam.
3. Land Evidence and native adapter prerequisites from #4/#5.
4. Expose a stable local machine-readable Testule invocation interface.
5. Integrate one read-only Testule operation in Dubnium #677.
6. Add one AI-assisted proposal operation without canonical mutation.
7. Compose bounded generated-test/fuzz execution with the host runtime boundary.
8. Add a separate governed-effect bridge only after a concrete consequential mutation requires it.

## Decision summary

- Testule owns testing-domain capabilities; it does not own a general AI runtime.
- AI is a caller/producer within the same typed capability boundary, not a separate capability architecture.
- `CapabilityContract`, `CapabilityInvocation`, and `CapabilityResult` are distinct concepts.
- Callers cannot self-authorize; effective authority is externally bounded and fail-closed.
- Generated artifacts are proposals with explicit trust states.
- Semantic testing capabilities do not automatically become governed effects.
- Dubnium is the first reference host, not a dependency.
- Testule Evidence remains separately owned and attributable from host runtime/governance evidence.
- A simple local machine-readable provider seam should be proven before standardizing a broader protocol.