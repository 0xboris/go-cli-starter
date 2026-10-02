# go-cli-starter

A small, compiling, tested starter for Go command-line tools, modeled on the GitHub CLI (gh).
It implements the contract of the `quality-cli` skill:
thin `main`, `Main()` returning exit codes, typed errors, lazy Factory, IOStreams with
TTY/pipe contracts, Options + `NewCmdX(f, runF)` + `xRun`, `--json/--jq/--template`,
`CanPrompt` (stdin + stderr) + `--yes` + global `--no-input`/`--no-color`, prompts on
stderr, exit codes 0/1/2/4 plus 128+signal (130 Ctrl-C), escaped piped fields,
cell-width tables, help topics, nested typo suggestions, generated docs,
unit + testscript acceptance tests, and a layered structure (domain package, an
external-program adapter, per-noun shared code) whose import rules are enforced by
depguard and by `go test`. Requires Go ≥ 1.22.
Dependencies: cobra, pflag, x/term, go-runewidth, gojq, yaml.v3, go-internal (tests).

## Use it

```sh
git clone --depth 1 https://github.com/0xboris/go-cli-starter mytool && cd mytool
rm -rf .git && git init
# 1. rename module and binary
grep -rl 'example.com/tool' . | xargs sed -i 's#example.com/tool#github.com/me/mytool#g'
grep -rl 'TOOL_' . | xargs sed -i 's/TOOL_/MYTOOL_/g'
mv cmd/tool cmd/mytool
sed -i 's#cmd/tool#cmd/mytool#; s#bin/tool#bin/mytool#' Makefile .goreleaser.yml
# then replace remaining user-facing "tool" strings (Use:, help text, Makefile, .goreleaser.yml)
# 2. verify
go mod tidy && go test ./... && make build && ./bin/mytool --help
```
On macOS use `sed -i ''`.

Then:
1. Replace `internal/api` with your real domain (HTTP client, database or files). Keep its rules:
   typed models with `ExportData`, `ctx` first, typed errors, no cobra/IOStreams/env/printing.
2. Add your nouns under `pkg/cmd/<noun>/<verb>/` by copying `pkg/cmd/item/list` (list),
   `pkg/cmd/item/view` (single item, `--web`, `--json`) and `pkg/cmd/item/delete` (mutating
   command with confirmation); register them in `pkg/cmd/root/root.go`. Code several verbs
   share goes in `pkg/cmd/<noun>/shared`; a verb never imports another verb.
3. Wrap each external program you run (compiler, renderer, git, ...) in its own package shaped
   like `internal/browser`, and inject it through the Factory.
4. Add new domain and adapter packages to the depguard `below-commands` files in
   `.golangci.yml` and to the rules in `internal/archtest`.
5. Delete `pkg/cmd/item` once you have a real noun.
6. Update `pkg/cmd/root/help_topic.go` (`environment`, `exit-codes`) whenever you add env vars or codes.

## Map

| Path | Role |
|---|---|
| `cmd/tool/main.go` | `os.Exit(int(app.Main()))` — nothing else |
| `cmd/gen-docs` | man pages + markdown from the command tree (`make docs`, `make check-docs` in CI) |
| `internal/app` | composition root — `Run(args, ios)`: factory, env/config precedence, signals, error → exit code |
| `internal/api` | domain layer: `Item`, `ItemID` value type, `Client` contract, `NotFoundError`, in-memory impl |
| `internal/browser` | external-program adapter (gh `git.Client` shape): `Browser` interface, `Launcher`, typed errors |
| `internal/archtest` | `go test` check that domain/adapter packages never depend on cobra, `pkg/` or `internal/app` |
| `internal/build` | `Version`/`Date` via `-ldflags`, `ReadBuildInfo` fallback |
| `internal/config` | config dir resolution, options table, atomic writes |
| `internal/prompter` | `Prompter` interface, line-based (accessible) impl, `Mock` |
| `internal/tableprinter` | aligned/colored on TTY; TSV, no header, RFC3339 when piped |
| `pkg/iostreams` | streams, TTY detection + test overrides, color, pager, spinner |
| `pkg/cmdutil` | `Factory`, typed errors, enum flags, `AddJSONFlags` exporter |
| `pkg/cmd/root` | root command, gh-style help, terse usage, help topics |
| `pkg/cmd/item/{list,view,delete}` | sample list, view (`--web`, `--json`) and destructive commands with tests |
| `pkg/cmd/item/shared` | code the item verbs share (ID parsing as a usage error, not-found hint, state color) |
| `acceptance/` | testscript `.txtar` end-to-end scripts running `app.Main` in-process |

## Layers

```
cmd/tool → internal/app → pkg/cmd/root → pkg/cmd/item/<verb> → pkg/cmd/item/shared
                                                │
                    ┌───────────────────────────┴─────────────────────┐
                    ▼                                                 ▼
          internal/api (domain)                         internal/browser (adapter)
```
Imports point down only. Commands parse flags, call the domain, and render; the domain
and adapters return data and typed errors and never touch cobra, IOStreams, env or
`os.Exit`. `make lint` (depguard + forbidigo) and `go test ./...` (`internal/archtest`)
both fail on a violation. See `references/layers.md` in the quality-cli skill.

## Try it

```sh
go run ./cmd/tool item list                 # TTY: header, colors, relative time
go run ./cmd/tool item list | cat           # piped: TSV, no header, RFC3339
go run ./cmd/tool item list --json          # lists available fields
go run ./cmd/tool item list --json id,title --jq '.[].title'
go run ./cmd/tool item view 3                # piped: key<TAB>value lines
go run ./cmd/tool item view 3 --web          # opens the URL via TOOL_BROWSER / BROWSER / OS default
go run ./cmd/tool item delete 3 </dev/null  # refuses: --yes required when not interactive
go run ./cmd/tool item delete 3 --no-input  # same, even on a terminal (CI, agents)
go run ./cmd/tool item delete 3 > out.txt   # still prompts (on stderr); out.txt only gets data
go run ./cmd/tool item lsit                 # Did you mean this? list
go run ./cmd/tool help environment
```

## Releasing

Label the pull request: `release:patch`, `release:minor` or `release:major` (the
labels are created when `.github/workflows/release.yml` lands on main). When the PR
is merged, the `release` workflow bumps the latest `vX.Y.Z` tag accordingly, runs
`go mod tidy -diff`, `go vet`, `go test -race` and golangci-lint on the merge commit,
pushes an annotated tag on it and comments the version on the PR.

The label can go on before or after the merge. Labeling an already-merged PR
releases its merge commit, unless a newer release already exists. A PR with two
release labels, or one that is already tagged, is refused or skipped. Users install
with `go install example.com/tool/cmd/tool@<version>`.

`.goreleaser.yml` is not wired into the workflow yet. Before adding a GoReleaser step,
add a `LICENSE` file (the archives include it) and set `nfpms.maintainer`, then check
with `goreleaser release --snapshot --clean`.

## Upgrades when you need them

- Richer terminal support (Windows VT, 256/truecolor, themes, `GH_FORCE_TTY %`): `github.com/cli/go-gh/v2/pkg/term`,
  `pkg/tableprinter`, `pkg/jq`, `pkg/template`, `pkg/markdown` (requires a newer Go).
- Rich prompts: `github.com/charmbracelet/huh` behind the same `Prompter` interface; keep the line-based one as the accessible mode.
- HTTP stubs: write a self-verifying `httpmock.Registry` (unused stubs fail the test; see gh's `pkg/httpmock`) once you have a real HTTP client.
- More subprocesses: copy `internal/browser`'s shape (one type per program, ctx first, injected
  streams, typed errors with stderr, an unexported `commandContext` test seam).
- Keyring: `github.com/zalando/go-keyring`, with a timeout on every call.
