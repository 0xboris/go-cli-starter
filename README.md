# go-cli-starter

A small, compiling, tested starter for Go command-line tools, modeled on the GitHub CLI (gh).
It implements the contract of the `quality-cli` skill:
thin `main`, `Main()` returning exit codes, typed errors, lazy Factory, IOStreams with
TTY/pipe contracts, Options + `NewCmdX(f, runF)` + `xRun`, `--json/--jq/--template`,
`CanPrompt` + `--yes`, help topics, nested typo suggestions, generated docs,
unit + testscript acceptance tests. Requires Go ≥ 1.22.
Dependencies: cobra, pflag, x/term, gojq, yaml.v3, go-internal (tests).

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
1. Replace `internal/api` with your real client (keep the `Client` interface + `ExportData`).
2. Add your nouns under `pkg/cmd/<noun>/<verb>/` by copying `pkg/cmd/item/list` (read command)
   and `pkg/cmd/item/delete` (mutating command with confirmation); register them in `pkg/cmd/root/root.go`.
3. Delete `pkg/cmd/item` once you have a real noun.
4. Update `pkg/cmd/root/help_topic.go` (`environment`, `exit-codes`) whenever you add env vars or codes.

## Map

| Path | Role |
|---|---|
| `cmd/tool/main.go` | `os.Exit(int(app.Main()))` — nothing else |
| `cmd/gen-docs` | man pages + markdown from the command tree (`make docs`, `make check-docs` in CI) |
| `internal/app` | `Run(args, ios)`: factory, env/config precedence, signals, error → exit code |
| `internal/build` | `Version`/`Date` via `-ldflags`, `ReadBuildInfo` fallback |
| `internal/config` | config dir resolution, options table, atomic writes |
| `internal/prompter` | `Prompter` interface, line-based (accessible) impl, `Mock` |
| `internal/tableprinter` | aligned/colored on TTY; TSV, no header, RFC3339 when piped |
| `pkg/iostreams` | streams, TTY detection + test overrides, color, pager, spinner |
| `pkg/cmdutil` | `Factory`, typed errors, enum flags, `AddJSONFlags` exporter |
| `pkg/cmd/root` | root command, gh-style help, terse usage, help topics |
| `pkg/cmd/item/{list,delete}` | sample read and destructive commands with tests |
| `acceptance/` | testscript `.txtar` end-to-end scripts running `app.Main` in-process |

## Try it

```sh
go run ./cmd/tool item list                 # TTY: header, colors, relative time
go run ./cmd/tool item list | cat           # piped: TSV, no header, RFC3339
go run ./cmd/tool item list --json          # lists available fields
go run ./cmd/tool item list --json id,title --jq '.[].title'
go run ./cmd/tool item delete 3 </dev/null  # refuses: --yes required when not interactive
go run ./cmd/tool item lsit                 # Did you mean this? list
go run ./cmd/tool help environment
```

## Upgrades when you need them

- Richer terminal support (Windows VT, 256/truecolor, themes, `GH_FORCE_TTY %`): `github.com/cli/go-gh/v2/pkg/term`,
  `pkg/tableprinter`, `pkg/jq`, `pkg/template`, `pkg/markdown` (requires a newer Go).
- Rich prompts: `github.com/charmbracelet/huh` behind the same `Prompter` interface; keep the line-based one as the accessible mode.
- HTTP stubs: write a self-verifying `httpmock.Registry` (unused stubs fail the test; see gh's `pkg/httpmock`) once you have a real HTTP client.
- Subprocesses: add an `internal/run` seam before the first `exec.Command`.
- Keyring: `github.com/zalando/go-keyring`, with a timeout on every call.
