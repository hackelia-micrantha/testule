# CLI and Unix composability contract

Status: incubating `v1alpha1` contract

Tracking epic: #25

## Purpose

Testule is a CLI-first capability layer for portable testing contracts, adapters, and normalized evidence. Its command-line interface must behave as a composable Unix tool rather than as an interactive application or a collection of file-only commands.

This document defines the process boundary shared by shell users, CI systems, and future agent/supervisor transports. Transport-specific bindings may differ, but the underlying Testule operation, inputs, outputs, evidence semantics, and capability boundaries should not.

## Design principles

1. **One semantic operation, multiple transports.** CLI, CI, and agent/supervisor invocation are bindings over the same Testule domain contracts. A transport must not silently change the meaning of validation, evidence, gaps, or execution outcomes.
2. **Streams are first-class.** Commands should accept stdin and emit stdout where their semantics permit sequential processing. Files remain supported, but should not be the only automation path.
3. **Data and diagnostics are separate.** stdout is reserved for requested command output. Human diagnostics, warnings, progress, and errors belong on stderr.
4. **Machine output is contractual.** Automation consumes explicitly selected structured formats, never human-oriented text scraping.
5. **Mutation is explicit.** Commands that alter files, corpora, environments, or external state must make that behavior explicit and remain bounded by capability contracts.
6. **Non-interactive by default.** Normal Testule operations must be safe for pipes, CI, and agent invocation without a controlling terminal.
7. **Unix failure semantics matter.** Exit status, EPIPE/SIGPIPE behavior, and partial-input handling are part of the public CLI contract.

## Standard streams

### stdin

Where a positional path denotes a read-only input document or stream, `-` should denote stdin when the command can consume that input unambiguously.

Target examples:

```sh
cat plan.yaml | testule validate --format json -

testule fingerprint - < plan.yaml
```

Commands accepting multiple heterogeneous inputs must define which option or position owns stdin. A command must reject ambiguous attempts to bind stdin to more than one logical input rather than guessing.

For example, a future gaps surface may use explicit options when both plan and evidence can be streamed:

```sh
testule gaps --plan plan.yaml --evidence - --subject-revision "$REV" --format json
```

The precise syntax remains versioned with the CLI; the invariant is that stdin binding is explicit and deterministic.

### stdout

stdout contains only the requested primary result.

Examples include:

- validation results;
- fingerprints;
- gap reports;
- normalized Evidence records;
- explicitly requested human text output.

When `--format json` or another machine format is selected, stdout must remain parseable even when warnings or execution diagnostics occur.

### stderr

stderr is reserved for diagnostics and operator-facing context that is not part of the selected result format, including:

- invalid invocation details;
- warnings;
- adapter/runtime diagnostics;
- progress, if progress is ever emitted;
- internal failure information.

Scripts must not need to parse stderr to determine a normal domain result.

## Formats

### Text

`text` is a human-oriented presentation format. Its wording and layout may evolve and must not be treated as an automation API.

### JSON

`json` is the structured document format for finite command results such as validation and gap reports.

Machine-readable fields and meanings require compatibility discipline within the declared CLI/schema version. Additive fields are preferred to incompatible reinterpretation. Breaking changes require an explicit version boundary.

### JSONL

JSON Lines (`jsonl`) is the preferred format for streaming independent normalized records, particularly Evidence, once a command has a proven streaming use case.

Each line must be a complete independently parseable JSON object. Stream contracts must define:

- record kind/schema version;
- ordering guarantees, if any;
- whether duplicate records are permitted;
- malformed-record behavior;
- partial final-line behavior;
- provenance and identity preservation.

JSONL should not be added merely for symmetry. It should be introduced where record-at-a-time composition is useful and semantically sound.

## Exit status

Testule should define stable exit-status classes rather than requiring callers to infer state from text.

The exact numeric allocation is tracked by #26, but the semantic classes are:

- **success** — command completed and its success condition is satisfied;
- **domain-negative** — command completed correctly but the requested test/validation/gap condition is not satisfied;
- **usage/configuration error** — invocation or declarative input is invalid for the requested operation;
- **execution failure** — an adapter, dependency, environment, or bounded test execution failed in a way distinct from a normal negative test result;
- **internal failure** — Testule itself could not complete due to an unexpected defect or invariant violation.

A command-specific domain-negative result must remain representable in structured stdout when useful; exit status is a control-flow signal, not a replacement for Evidence.

## Signals and broken pipes

A downstream consumer may terminate early:

```sh
testule gaps --format json ... | head -c 200
```

EPIPE/SIGPIPE from a closed downstream pipe is ordinary Unix pipeline behavior. Testule should terminate without reporting a misleading internal error or stack trace.

Signal handling must not corrupt persistent state. Commands with explicit mutation semantics must either complete their atomic unit or leave recoverable, documented state.

## Interaction

Commands are non-interactive by default.

A normal Testule command must not unexpectedly:

- prompt for confirmation;
- open an editor;
- launch a browser;
- request credentials from a terminal;
- depend on TTY-only formatting.

If an interactive workflow is ever added, it must be explicitly selected and have a non-interactive equivalent where the underlying operation is suitable for automation.

## Composition examples

### Shell filtering

```sh
testule validate --format json plan.yaml | jq -e '.valid'
```

### Gap inspection

```sh
testule gaps \
  --subject-revision "$REV" \
  --format json \
  plan.yaml evidence/*.yaml \
  | jq '.requirements[] | select(.state != "satisfied")'
```

### Fan-in with xargs

Where a command accepts multiple evidence paths, standard shell discovery should remain usable:

```sh
find evidence -name '*.yaml' -print0 \
  | xargs -0 testule gaps --subject-revision "$REV" --format json plan.yaml
```

### Preserve and consume a normalized stream

Target shape after #27 proves the producer/consumer semantics:

```sh
testule <evidence-producer> --format jsonl ... \
  | tee evidence.jsonl \
  | testule gaps --plan plan.yaml --evidence - --subject-revision "$REV" --format json
```

The producer command must arise from a real adapter/domain requirement; Testule should not invent a universal runner solely to make this pipeline possible.

## CLI and agent/supervisor parity

Unix composability and agent orchestration should reuse the same semantic contracts.

A useful mental model is:

```text
native tools / declared resources
            |
            v
       Testule operation
   (validate, execute, evaluate)
            |
      normalized result
       /      |       \
      v       v        v
   shell      CI    agent/supervisor
```

The transport binding may add metadata such as actor identity, approval context, capability grants, or invocation provenance. It must not weaken Testule's declared filesystem, network, secret, process, mutation, or evidence rules.

This keeps the CLI useful outside an orchestrator while allowing the same operations to be safely exposed as governed capabilities.

## Man pages

Testule should ship operator documentation suitable for `man`.

At minimum `testule(1)` must document:

- synopsis and command tree;
- stdin/stdout/stderr conventions;
- formats;
- exit status;
- environment variables;
- files and state locations, if any;
- signal behavior;
- mutation/capability notes;
- examples.

Subcommands may use dedicated pages or generated sections, but man pages and `--help` must derive from or be validated against a canonical command description so they do not drift.

Tracked by #28.

## Conformance expectations

CI should eventually assert at least:

- stdin works for supported read-only inputs;
- machine stdout remains parseable while diagnostics are written to stderr;
- text output is not used as an interchange dependency;
- exit-code classes are stable;
- no normal command prompts unexpectedly;
- EPIPE/SIGPIPE does not surface as an internal error;
- one Testule-to-Testule pipeline works without temporary interchange files;
- man pages and CLI help agree on shipped commands and options.

## Delivery plan

The work is split under epic #25:

- #26 — stdin/stdout, exit-code, and signal semantics;
- #27 — streaming normalized Evidence and Testule-to-Testule composition;
- #28 — man pages and Unix CLI conformance tests.

The contract should harden incrementally with the `v1alpha1` CLI. It should not force premature stabilization of the generic adapter protocol described in issue #16.
