# review-DocExampleTests v2.0 — plan

- Created / last modified: 2026-07-26 12:42
- Target release: 2.0.4
- Plan version: v2.0 (aligned with feature docs baseline `-v2.0`)

## Documentation references

Companion feature docs created / maintained under this plan (Keytiles docs rules: `FeatureName-vMajor.Minor.md`, self-contained, no patch):

- [docs/Fault-v2.0.md](../../docs/Fault-v2.0.md)
- [docs/PublicFaultConversion-v2.0.md](../../docs/PublicFaultConversion-v2.0.md)
- [docs/StatusCodeMapping-v2.0.md](../../docs/StatusCodeMapping-v2.0.md)
- [docs/FaultSerialization-v2.0.md](../../docs/FaultSerialization-v2.0.md)

Also updated:

- [README.md](../../README.md) — links to the four feature docs

## Why are we doing this?

- The library had useful README + code comments but no `/docs` feature documentation matching Keytiles standards.
- We need clear, user-facing feature boundaries so maintainers can evolve the lib in isolation later (docs + plans per feature).
- Release 2.0.4 is a good moment to baseline documentation at **v2.0** (major.minor, no patch) before further enhancements.
- The broader review also aims to improve **examples** and **tests** alongside docs (this plan name: Doc / Example / Tests).
- Code review of `pkg/` + `tests/` found panic risks, one confirmed retryable logic bug, enhancement opportunities, and missing sane tests — these must be fixed for 2.0.4 quality.

## What will be changed?

- Introduce `/docs` with four v2.0 feature documents covering the current public surface of `pkg/kt_errors`.
- Cross-link README to those docs.
- Fix panic risks and the `Build()` retryable override bug (priority order below).
- Add missing tests (TDD: red first), then production fixes to green — only after user confirms each test batch.
- (Later) Review / add examples that match the documented workflows.
- Related 2.0.4 work done in parallel (not a public API break): upgrade internal deps to `lib-sets-golang/v2` and `lib-utils-golang/v2`, adapt call sites (`pkg/kt_sets` import path, `*Set`, `AsSlice`).

## Decisions we made

- **Feature granularity = four docs** (not one mega-doc, not one doc per file):
  - `Fault` — core model + builder + kinds + error codes + enrichment (builder is not separate: it is the only public construction path).
  - `PublicFaultConversion` — boundary conversion + `ConversionOption`s (already evolved independently in 2.0.0/2.0.1).
  - `StatusCodeMapping` — HTTP / gRPC mapping.
  - `FaultSerialization` — natural / full JSON + `SerializationOption`s.
- **Not separate docs:** FaultBuilder alone, error-code catalog alone, `IsFault`, `LIB_NAME` / `PACKAGE_NAME`.
- **Doc versioning:** first baseline is `-v2.0.md`. Grow in place during 2.0.4; do not invent `-v2.1` unless a new plan cycle or explicit bump is requested.
- **SemVer for dep upgrades:** switching to `lib-sets-golang/v2` / `lib-utils-golang/v2` under the hood does **not** require this lib to go to v3 — public API of `kt_errors` does not re-export those types; Go module paths for major versions coexist. Stay on **2.0.4**.
- **TDD for review fixes:** for each implementation step below:
  1. Write the new/extended test cases first (they must be **red** / fail or panic as expected).
  2. **Stop and wait for user confirmation** that the test cases make sense.
  3. Only after confirmation: change production code until those tests are **green**.
  4. Do not sneak in unrelated production changes in the same step.

## Code review findings (pkg/ + tests/)

Source: review of `pkg/kt_errors` and `tests/` (2026-07-26). Several items reproduced with a small harness.

### Priority 1 — Panic risks

1. **`ToFullJSON(nil, ResolveMessages)` panics** (`fault.go`)  
   Nil path sets empty template, then `resolveMessages` still dereferences nil `fault` fields.  
   Repro: `GetFaultAsFullJSON(nil, ResolveMessages)` → nil pointer panic. Existing nil test does not pass `ResolveMessages`.

2. **`NewPublicFaultFromAnyError` + `OptionWhitelistedFaultKinds(true, …)` on a plain `error` panics** (`util_funcs.go`)  
   `inheritErrorCodes` calls `fault.GetErrorCodes()` even when `isFault == false`.

3. **`Error()` / `String()` are not nil-safe** (`fault.go`)  
   Other methods guard nil; these do not. Typed-nil `Fault` → panic on `.Error()` / `.String()` / `%s`.

4. **Nil element in `options ...ConversionOption`** (`util_funcs.go`)  
   `opt.getOptionId()` on nil option panics. Less common; harden while touching conversion.

5. **Nil `*FaultBuilder` method calls**  
   Misuse case; lower priority than 1–3.

### Priority 1b — Confirmed logic bug

6. **`Build()` retryable override does not affect the returned Fault** (`fault_builder.go`)  
   Copies `_fault := builder.fault`, then sets `builder.fault.Retryable = false` for certain auth/authz codes, but returns `&_fault` with the old flag.  
   Repro: `AuthenticationFault` + `AUTHENTICATION_ERRCODE_MISSING` + `WithIsRetryable(true)` → `IsRetryable()` stays **true** (should be false).

### Priority 2 — Enhancements / other issues

7. Shared map/slice aliases in serialization (`ErrorCodes` / `Labels`) when not copying — prefer always copy for marshal safety.
8. Package-level `_EMPTY_*` / `_NONPUBLIC_*` templates share map/slice headers across calls — fragile if later mutated.
9. Circular `cause` in `String()` can stack-overflow — document or depth-limit.
10. HTTP vs gRPC asymmetry for `ILLEGALSTATE_ERRCODE_EXHAUSTED` (503 vs `ResourceExhausted`) — document + lock with tests.
11. Unused `properties` field on `defaultFault` — remove or use.
12. Typos / naming (`EXCPECTATION_FAILED`, comment typos) — comment fixes free; constant rename would be breaking.
13. `Fault` mutation not concurrency-safe — document caller responsibility.
14. `inheritErrorCodes` without kind kept can attach original codes onto `RuntimeFault` + internal — clarify docs or restrict.
15. `WithSource` appends on repeated calls — document intended behavior.

### Missing sane test cases (catalog)

Panic / nil / conversion:

- `GetFaultAsFullJSON(nil, ResolveMessages)` (and PrettyPrint combo) must not panic
- `NewPublicFaultFromAnyError(plainError, …)` with/without transactionId
- `NewPublicFaultFromAnyError(nil)` → nil
- `OptionWhitelistedFaultKinds(true, …)` on non-Fault (currently panics)
- Whitelist kind kept + `inheritErrorCodes=false` → no `ERRCODE_INTERNAL_ERROR`, codes dropped
- Typed-nil / nil-safe `Error()` / `String()`
- Nil option in conversion `options`

Builder / retryable:

- `WithIsRetryable(true)` ignored for Validation / NotImplemented / ResourceNotFound
- Auth `MISSING` / `NOT_SUPPORTED` and authz `NO_PERMISSION` force non-retryable on `Build()`
- `WithExactLabels` / `WithExactMessageTemplatesByAudience` / `WithoutMessageTemplateForAudiences`
- `AddLabels` / empty context add no-ops

Status mapping (kind defaults exist; **error-code overrides missing**):

- Constraint → 409 / `AlreadyExists` for already-taken / already-exist
- Constraint → 404 / `NotFound` for does-not-exist
- IllegalState → 503 / `Unavailable` for dependency unavailable / timed out
- IllegalState → gRPC `ResourceExhausted` for exhausted
- IllegalState → 412 / `FailedPrecondition` for expectation failed
- Method wrappers `fault.GetHttpStatusCode()` / `GetGrpcStatusCode()`

Serialization / misc:

- Non-public `ToFullJSON` defensive blank form
- `PrettyPrint`
- `IsFault` for nil, plain error, Fault
- `GetSource()` vs call-stack deepest hop

## Implementation steps

### Done earlier in this plan

1. **Decide feature granularity for `/docs`** — **implemented**
2. **Author v2.0 feature docs** — **implemented**
3. **Adapt code for sets/utils v2 (2.0.4 upgrades)** — **implemented**
4. **Code review of `pkg/` + `tests/`** — **implemented** (findings recorded above)

### TDD fix stream (priority order)

Process note: early steps used strict red→confirm→green. From Priority 1 panics onward (user request), tests + fixes may land together for review.

5. **P1 — Fix `ToFullJSON` nil + `ResolveMessages` panic** — **implemented**
   - Tests: extended `TestGetFaultAsFullJSON_NilFaultIsSafe` with `ResolveMessages`.
   - Code: resolve path only runs when serializing real content (`serializeContent`); avoids nil deref and avoids resolving into blanked non-public form.
6. **P1 — Fix `inheritErrorCodes` panic on non-Fault** — **implemented**
   - Tests: `TestNewPublicFaultFromAnyError_PlainErrorAndNilOptionsAreSafe`.
   - Code: inherit codes only when `isFault`; nil original already returned nil.
7. **P1 — Nil-safe `Error()` / `String()`** — **implemented**
   - Tests: `TestTypedNilFault_ErrorAndStringAreSafe` via `VisibleForTesting_NilFault()`.
   - Code: nil guards; `Error()` → `""`, `String()` → `"Fault{nil}"`.
8. **P1b — Fix `Build()` retryable override** — **implemented**
   - Tests: `TestBuild_RetryableOverridesAndInherentNonRetryableKinds` in `tests/fault_test.go` (kept as agreed).
   - Code: `Build()` now applies auth/authz non-retryable overrides to `_fault.Retryable` (the returned instance), not `builder.fault` alone.
9. **P1 — Harden nil `ConversionOption`** — **implemented** (merged with step 6)
   - Nil options skipped in `NewPublicFaultFromAnyError` options loop.
10. **P2 — Status-code error-code override tests (+ wrappers)** — **planned**
    - Constraint / IllegalState override matrix for HTTP and gRPC
    - `fault.GetHttpStatusCode()` / `GetGrpcStatusCode()` match package funcs
11. **P2 — Remaining missing tests (builder / conversion / serialization / IsFault)** — **planned**
    - Catalog items not covered in steps 5–10.
12. **P2 — Soft enhancements** — **planned** (after panic/bug stream)
    - Serialization always-copy labels/errorCodes; document concurrency + exhausted HTTP/gRPC asymmetry; unused `properties`; comment typos.
13. **Examples aligned with docs** — **planned**
14. **Final pass** — **planned**
    - `go test ./...` green; CHANGELOG kept current; mark remaining steps as they land.

## Notes

- Docs are the **current** v2.0 companions for this release plan; update them in place if 2.0.4 doc fixes are needed.
- Highest fix order matches Priority 1 panics → retryable bug → conversion nil option → status override tests → remaining gaps → soft enhancements.
- **Gate:** no production code for a TDD step until the user confirms that step’s red tests look right.
