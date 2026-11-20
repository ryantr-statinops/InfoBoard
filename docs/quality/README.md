# Quality Assurance & Testing

This directory defines the verification strategies, compatibility boundaries, privacy guarantees, and acceptance criteria for InfoBoard.

---

## Quality Documents

1. [**Test Strategy**](test-strategy.md): Automated unit test suites (32 tests in ~75ms) testing MRU management, scoring, surface interaction, and bundle packaging.
2. [**Acceptance Matrix**](acceptance-matrix.md): Production release criteria and validation matrix across all feature gates.
3. [**Compatibility and Performance**](compatibility-and-performance.md): Platform support (Chrome, Edge on Linux, macOS, Windows) and performance benchmarks (< 1ms search, ~18KB bundle).
4. [**Privacy and Security**](privacy-and-security.md): Offline execution, least-privilege permissions, zero network egress, and session isolation.
