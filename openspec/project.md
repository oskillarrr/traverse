# Project Context — Traverse (Offline Architecture Analyzer)

## Purpose
Offline, deterministic (no AI) static architecture analysis of portfolios of applications stored in many repositories (GitLab, Subversion, others), with output that is traceable to evidence (`file:line`, versioned rule) and reusable in official architecture documentation (C4).

## Constraints (non-negotiable)
- No AI, no network access, no telemetry, no auto-update.
- Read-only on analyzed repositories; outputs only under `--out`, which must be outside them.
- Never compile or execute analyzed code; never invoke build tools or VCS clients.
- Portable: one self-contained executable per OS/arch, no installation, no admin rights, no runtime (JDK, Python, .NET) required.
- Technology-agnostic: inventory and rules for any language; deep extractors added per technology.
- Never copy code lines, property values, secrets or full URLs into outputs.
- Open source (Apache-2.0); no third-party code.

## Architecture decisions

### ADR-001 — Implementation language: Go, standard library only (2026-11-03)
- **Context:** the PoC was a PowerShell 5.1 script (Windows-only, regex per line). Requirements changed to portable, local, OS/technology agnostic, public open source.
- **Options:** PowerShell 7 (needs runtime install on Linux/macOS, slow, weak parsing); Java CLI (needs JDK); Python (needs interpreter); Go (single static binary, cross-compiles to 6 targets, fast, strong stdlib for XML/JSON/regex/templates).
- **Decision:** Go with **zero third-party modules**. Go is needed only to build; users run a binary.
- **Consequences:** trivial supply-chain review; reproducible builds; organisations can rebuild in their own CI and sign. Deep Java parsing is structural (custom tokenizer), not semantic: acceptable trade-off documented in limitations.

### ADR-002 — Pipeline with a stable facts schema
`scan (per repo) → facts.json (schemaVersion semver) → portfolio aggregation → renderers`. Extractors can be added or replaced (e.g. a future bytecode/jdeps extractor) without touching aggregation or reports.

### ADR-003 — Rules as data
Rules are JSON with RE2 patterns and mandatory positive/negative examples executed by `traverse rules test` and CI. RE2 guarantees linear-time matching on untrusted input. A literal prefilter derived from each pattern keeps scans fast.

### ADR-004 — Evidence over inference
Every relationship carries kind, keys, confidence and locations. Generic matches (HTTP paths with fewer than two literal segments) are never linked automatically; the most specific match wins.

## Conventions
- Reports in English and Spanish (`--lang`); identifiers, JSON keys and rule ids in English.
- Severity: `info | low | medium | high`; confidence: `low | medium | high`.
- Validation states used by architects: `confirmado | probable | descartado | no-verificable`.
- Semver for tool, rules catalog and facts schema, independently.
