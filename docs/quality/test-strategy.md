# Test strategy

## Test seams

- Browser fixtures generate tab create/update/move/group/pin/activate/remove events across windows and profiles.
- Projection fixtures compare indexed IDs and fields with the browser-visible eligible snapshot.
- Protocol fixtures exercise framed messages, handshake, revisions, request IDs, limits, timeouts, reconnect, and typed errors.
- Ranking fixtures cover exact fields, prefixes, phrases, typos, duplicate titles, context signals, empty queries, Unicode, long fields, and adversarial input.
- UX fixtures exercise keyboard journeys, stale selections, no results, runtime unavailable, rebuild, reset, and uninstall states.
- Packaging fixtures cover clean install, update, rollback, repair, missing host, wrong origin, permission denial, and idempotent removal.

## Required assertions

Tests should assert observable contracts: deterministic IDs/order/scores, convergence, state transitions, confirmed activation, bounded output, safe error classes, redaction, ownership boundaries, and accessibility semantics. They should not assert private wiring, incidental field copies, or implementation-specific defaults.

## Failure-first coverage

The verification set must include host absence/crash, malformed JSON, oversized payload, interrupted write, persistence failure, out-of-order delta, profile mismatch, protocol mismatch, permission denial, stale result, disappearing tab, and private-context separation.

## Evidence

Record p50/p95/p99 latency and memory at the declared 1,000-tab envelope. Keep browser-family and operating-system results separate. A test result without its product version, protocol version, browser/OS matrix, and fixture revision is not sufficient release evidence.

The [canonical verification contract](../plan/refactor/verification-and-acceptance.md) defines the full acceptance surface.
