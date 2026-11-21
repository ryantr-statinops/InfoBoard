# Contributing to InfoBoard

Thank you for your interest in contributing to InfoBoard! We welcome contributions that maintain our core principles: **ultra-fast performance (< 1ms)**, **pure Manifest V3 architecture**, and **zero external bloat**.

---

## 1. Code of Conduct

All contributors and maintainers are expected to adhere to our [Code of Conduct](CODE_OF_CONDUCT.md). Please report unacceptable behavior to repository maintainers.

---

## 2. Git Workflow & Branching Model

- **`main`**: Production release branch. Contains only stable, tagged releases.
- **`dev`**: Active development branch. All feature branches and PRs must target `dev`.
- **Feature Branches**: Branch from latest `dev` using descriptive names:
  - `feat/feature-name`
  - `fix/bug-description`
  - `docs/doc-updates`

### Atomic Commits Rule
InfoBoard strictly enforces **small, atomic commits**:
- Each commit should represent a single cohesive change or fix.
- Follow [Conventional Commits](https://www.conventionalcommits.org):
  - `feat(scope): ...`
  - `fix(scope): ...`
  - `style(scope): ...`
  - `docs(scope): ...`
  - `test(scope): ...`

---

## 3. Development & Testing Standards

1. **Pure MV3 Principle**:
   - Zero external runtime npm dependencies.
   - Zero native hosts or companion daemons.
   - Bundle size uncompressed must remain **< 50 KB**.
2. **TypeScript**:
   - Code must compile with zero errors under `extension/tsconfig.json` (`strict: true`).
3. **Automated Testing**:
   - Every new feature or bug fix must include automated test coverage under `tests/`.
   - Run `npm test` before committing. All tests must pass cleanly.

---

## 4. Submitting a Pull Request (PR)

1. Push your changes to your feature branch or fork.
2. Open a Pull Request targeting the `dev` branch.
3. Fill out the PR template with:
   - Clear summary of changes made.
   - Motivation and context.
   - Verification evidence (`npm test` output, bundle size check, manual testing steps).
4. Maintainers will review your PR and run automated CI checks.
