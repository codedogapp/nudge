# AGENTS.md

The canonical rules for everyone who works on nudge: the user and the coding agent. If anything else disagrees with this file, this file wins. It changes only with the user's approval.

- What we build, and in which order: [`docs/plan.md`](docs/plan.md).
- How we build it: this file.

## Working agreement

- **Bit by bit.** A step is one small, focused change: one package or one component, about 150 hand-written lines or fewer, plus its tests. Never generate large batches of code.
- **Scope discipline.** Do exactly the step. No drive-by refactors. Ideas go into `docs/plan.md`, not into the code.
- **Ask, don't assume.** When something isn't decided here or in the plan, ask the user, then record the answer in the right file.
- **Verify facts.** Check versions, APIs and tool options against their source (registry, docs, `--help`) before using them. Never guess.
- **Stay in the repo.** Don't change anything outside it without the user's OK.
- **Spikes live outside the repo**, with the user's standing OK. A spike's code is throwaway: it lives in `../nudge-spikes/<name>/`, next to the repo, and is never committed. Its findings reach the repo through a docs PR on the spike's `step/<id>` branch.

## Pipeline

The main session does all the work itself, in the open, so the user can follow it. No sub-agents. Every step goes through these roles, in order:

1. **Lead**: writes the step brief (goal, scope, out of scope, acceptance criteria). The brief goes into the PR description.
2. **Engineer**: implements the step with unit tests and runs every gate until green.
3. **QA**: checks every acceptance criterion, runs every gate, probes edge cases and failure paths in our code and config, and adds missing tests. It doesn't hunt for ways around the upstream tools.
4. **Reviewer**: reviews the diff against this file (separation of concerns, package boundaries, naming, KISS, YAGNI, error handling, test quality, fit with the architecture) and marks each finding blocking or non-blocking.
5. **Lead**: keeps the docs in sync, commits, opens the PR, and merges after the user's OK.
6. **User**: reviews the PR and approves or rejects it.

- QA and review run once per step. Defects and blocking findings go back to the engineer. A small fix (a few lines of config or docs) then needs only a green `mise run check`; a bigger one goes through QA and review again. After three rounds, stop and ask the user.
- Non-blocking findings go into the PR description, not into another round.
- Known upstream gaps go into plan §5 without asking the user.
- Progress notes to the user are short and factual: what changed, the commands run and their results, open issues.

## Definition of done

1. `mise run check` passes locally.
2. The PR's CI is green, including the SonarQube Cloud quality gate.
3. The user has approved the PR.

Never report a step as done before `mise run check` has passed, and say which gates actually ran.

## Git and pull requests

- Repo: [`codedogapp/nudge`](https://github.com/codedogapp/nudge), public. Default branch `main`.
- One step, one branch (`step/<id>`, ids from plan §0) and one PR. The lead squash-merges after the user approves, with an explicit subject and body: `gh pr merge --squash --subject "…" --body "…"`.
- Commit messages and PR titles follow Conventional Commits (`feat(ask): …`, `fix: …`, `build: …`, `ci: …`, `docs: …`): an imperative subject of 72 characters or fewer, and a body that says why when it isn't obvious.
- **No AI attribution.** Never add `Co-authored-by` trailers for Copilot or any other AI agent.
- Never force-push `main` or rewrite pushed history.
- Secrets never go into git or chat. The server's secrets live in the login Keychain on the Mac (`docs/setup.md`); other local settings in `mise.local.toml` (gitignored); CI secrets are GitHub organisation secrets.
- The repo is public: no personal identifiers, work-project names or third-party material in tracked files.
- A PR that changes what the user sees includes screenshots: light and dark, desktop and phone where they differ. PR attachments are public too, so take them with made-up questions and a time zone other than the host's.

## Engineering principles

- Clean code, KISS, YAGNI and separation of concerns.
- **One package, one job.** The package name says what it does. No `util`, `common`, `helpers` or `misc` packages; no god packages or god structs.
- **One concern per file**, and files stay short (see the gates).
- Small functions, early returns, shallow nesting.
- Domain states and enums get their own types, never bare strings.
- Prefer the standard library; never re-implement it.
- No speculative abstractions: an interface needs two implementations or a test seam; shared code needs two users.
- Pass dependencies explicitly. No package-level mutable state.
- Wrap errors with context (`fmt.Errorf("load config: %w", err)`) and handle each error once.
- Comment only where the code needs clarification. No heavy commenting.
- Tests check behaviour, not implementation details. Table-driven where it helps; no sleeps.

## Architecture

- **One binary, `nudge`**, with a subcommand per job: `nudge serve` runs the server, `nudge ask` asks a question and prints the answer. Plan §3 has the design.
- **One Go module** at the root: `github.com/codedogapp/nudge`. The `main` package stays thin wiring; everything else lives in `internal/` packages.
- **Two doors, two locks.** Agents reach the server only through a Unix socket that only the user can open; the phone reaches it only through `tailscale serve`, and only with the user's Tailscale login. Neither side uses the other's door.
- **Nothing on disk that the server can keep in memory.** A question lives as long as the request that asks it.

## Toolchain

- **mise** manages every tool and task, except tools that macOS and the CI runners already ship (curl, jq); those run at the OS's version. Every task has a description.
- **Latest versions.** Everything we install runs on its newest release.
  - mise tools use `latest` in `mise.toml`. mise skips releases younger than 24 h (its `minimum_release_age` default), and mise-action does the same for mise itself. CI installs mise and the tools afresh on every run, with mise-action's cache off. Update local tools with `mise upgrade --local`; a plain `mise upgrade` also changes the user's global tools.
  - The rest is pinned exactly at its newest release, checked against its registry: GitHub Actions by commit SHA with a `# vX.Y.Z` comment, Go modules in `go.mod`. Renovate keeps these current.
- **Nothing deprecated.** Never use a deprecated tool, API, endpoint, parameter, field or flag; use its replacement. Check the source's deprecation notes before picking one.
- **Go**: the latest release. Before editing Go code, run the `use-modern-go` skill's `list` for the file and follow its guidelines.
- **Go lint and format**: golangci-lint, based on the user's labrys config; gofumpt and golines at 100 columns.
- **Web**: pages rendered by the server with `html/template`, plain CSS, and only the JavaScript that Web Push needs (the service worker and the subscription). Its gates arrive with the first JavaScript file (plan §6).
- **Mirror the user's repos** before inventing a convention: Bakhyard (this file, CI, gates), labrys (Go lint) and jirascrap (CI, Sonar, Renovate).

## Quality gates

Measure everything we can with lightweight static tools. Thresholds change only with the user's OK. Every suppression (`//nolint`, a Sonar "won't fix") needs a reason next to it and the user's OK.

| Measure | Limit | Tool |
| --- | --- | --- |
| File length | ≤ 200 lines, not counting blank lines and comments | revive `file-length-limit` |
| Function length | ≤ 50 lines and ≤ 30 statements; tests exempt | `funlen` |
| Cognitive complexity | ≤ 15 per function | `gocognit` |
| Cyclomatic complexity | ≤ 10 per function | `gocyclo` |
| Nesting | ≤ 3 levels of control flow | revive `max-control-nesting` |
| Parameters, results | ≤ 4 parameters, ≤ 3 results | revive `argument-limit`, `function-result-limit` |
| Duplication | no clone of 100 tokens or more; tests exempt | `dupl` |
| Test coverage | ≥ 80% total, ≥ 70% per package; `main` package wiring excluded | go-test-coverage |
| Security | no findings | `gosec` in golangci-lint |
| Vulnerabilities | none known | `govulncheck` |
| Formatting | no diff | gofumpt, golines |
| Suppressions | every `//nolint` names its linters and every `//revive:disable` its rules, each with a reason; `#nosec` is ignored; every `coverage-ignore` gives a reason | `nolintlint`, revive directives, gosec, go-test-coverage |
| Package names | no `util`, `common`, `helpers`, `misc` or similar names (`api` is allowed), and no clash with a common standard-library package | revive `package-naming` |
| GitHub workflows | no findings | jactionlint (with shellcheck for `run:` scripts) |
| Dependencies | current | Renovate (GitHub App) |
| New code overall | quality gate passes | SonarQube Cloud |

The local gates run through `mise run check`. CI runs the same task and, in a parallel job, SonarQube Cloud.

### SonarQube Cloud

- It runs only in CI, on every PR and every push to `main`, and fails the build when the quality gate is red (`-Dsonar.qualitygate.wait=true`). Its verdict is final: `mise run check` is necessary but not sufficient.
- Never run a scanner locally. Scanning an uncommitted tree publishes it as a `main` analysis and corrupts the new-code baseline.
- Fix a red gate. Never explain it away, and never widen the exclusions in `sonar-project.properties` to turn it green.
- `mise run sonar-issues [pr]` lists the open issues, on `main` or on a pull request. Security hotspots show only in the SonarQube Cloud UI, because their API is deprecated.

## Communication

- With the user: terse (caveman style). Keep the review load small: what changed, where and why, in a few lines.
- Code, docs, commits and PRs: plain, normal prose.
- Docs live in `docs/`; diagrams use Mermaid. Refer to issues in other repos as `owner/repo#123`.
