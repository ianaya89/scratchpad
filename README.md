# pad

[![CI](https://github.com/ianaya89/scratchpad/actions/workflows/ci.yml/badge.svg)](https://github.com/ianaya89/scratchpad/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ianaya89/scratchpad?sort=semver)](https://github.com/ianaya89/scratchpad/releases)
![Go](https://img.shields.io/github/go-mod/go-version/ianaya89/scratchpad)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A tabbed terminal scratchpad. Each **tab** is a quick note; each **workspace** is a named set of tabs. Built with Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![demo](demo.gif)

Notes are plain Markdown files on disk, so they stay greppable and editable by hand. By default they live under your XDG data dir — but the storage location and other behavior are fully configurable (see [Configuration](#configuration)).

**Features:** tabbed notes · named workspaces · fuzzy find across tabs (`^f`) · live Markdown preview (`^o`) · reorder tabs · atomic saves · plain-Markdown storage · `--print` for scripting · config via flags, env, or file.

## What is a "workspace"?

A workspace is just a **named drawer of tabs**. Think of it as one project's scratch area.

- Open `pad` with no argument → the `default` workspace.
- Open `pad ideas` → a workspace called `ideas` (created on first use).
- Each workspace has its own independent set of tabs. Switching workspaces swaps the whole tab bar.

Use them however you like: one per project, one per context (`work`, `personal`, `meeting`), or just live in `default` forever. You don't have to think about workspaces at all if you don't want to — the default one is always there.

```
<data-dir>/
├── default/
│   ├── 1-untitled.md
│   └── 2-todo.md
└── ideas/
    └── 1-feature-brainstorm.md
```

## Installation

| Platform | Recommended |
| --- | --- |
| Linux | install script or prebuilt binary |
| macOS | Homebrew or install script |
| Any (with Go) | from source |

### Install script (Linux / macOS)

Downloads the right binary for your OS/arch, verifies its checksum, and installs it:

```sh
curl -fsSL https://raw.githubusercontent.com/ianaya89/scratchpad/main/install.sh | sh
```

Installs to `~/.local/bin` by default. Override with env vars:

```sh
PAD_INSTALL_DIR=/usr/local/bin PAD_VERSION=v0.1.0 \
  sh -c "$(curl -fsSL https://raw.githubusercontent.com/ianaya89/scratchpad/main/install.sh)"
```

### Homebrew (macOS)

```sh
brew install ianaya89/tap/pad
```

> The Homebrew cask is macOS-only. On Linux, use the install script or a prebuilt binary.

### Prebuilt binary

Grab a tarball for your OS/arch from the [releases page](https://github.com/ianaya89/scratchpad/releases), then:

```sh
tar -xzf pad_*_linux_amd64.tar.gz
install -m 0755 pad ~/.local/bin/pad
```

### From source (requires Go 1.26+)

```sh
git clone https://github.com/ianaya89/scratchpad ~/pad
cd ~/pad
make install          # builds with version info into ~/.local/bin/pad
# or: go build -o ~/.local/bin/pad .
```

Ensure `~/.local/bin` is on your `PATH` (or set `PREFIX=/usr/local make install`).

> If your Go toolchain can't reach the checksum database (sandboxed/offline), prefix builds with `GOSUMDB=off`.

## Usage

```sh
pad                  # open the "default" workspace
pad ideas            # open/create the "ideas" workspace
pad --dir ~/notes    # store notes under ~/notes instead of the default
pad --print          # dump the workspace's notes to stdout (no TUI)
pad --print --tab todo
pad --new "meeting" --content "kickoff notes"   # create a note, no TUI
echo "remember this" | pad --append todo        # append to a note from stdin
pad --help           # full flag reference
```

### Scripting (no TUI)

```sh
# create a note (content from --content or stdin)
pad --new "Standup" --content "- shipped X"
git log --oneline -5 | pad --new "recent commits"

# append to a note (matched by title; created if absent)
echo "buy milk" | pad --append shopping

# read notes back out
pad --print --tab standup
```

### Keybindings

| Key | Action |
| --- | --- |
| `^t` | New tab |
| `^d` (or `^w`) | Delete current note (asks to confirm) |
| `^n` | Next tab |
| `^p` | Previous tab |
| `alt+1`…`alt+9` | Jump straight to tab N |
| `alt+L` / `alt+H` | Move current tab right / left |
| `^f` | Find across tabs (title + body) in the workspace |
| `^o` | Markdown preview of the current note |
| `^r` | Rename current tab |
| `^e` | Workspace picker (switch, or `n` to create new) |
| `^/` (or `F1`) | Help overlay |
| `^s` | Save now |
| `^q` (or `^c`) | Save everything and quit |

When tabs exceed the terminal width, the bar scrolls to keep the active tab
visible and shows `‹N` / `N›` counters for hidden tabs on each side.

Fallback tab-switch keys are also bound: `alt+h`/`alt+l`, `ctrl+←`/`ctrl+→`, `shift+←`/`shift+→`.

> **Terminal note:** `^p`/`^n` (plain `ctrl`+letter) are delivered by essentially every terminal. They replace `ctrl+n`/`ctrl+p` line movement inside a note — use the arrow keys for that. `ctrl`-arrows are also bound but some terminals grab them for pane navigation.

## Configuration

Everything resolves with the precedence **flag → environment variable → config file → default**.

| Flag | Env var | Default | Description |
| --- | --- | --- | --- |
| `--dir PATH` | `PAD_DIR` | `$XDG_DATA_HOME/pad`, else `~/.local/share/pad` | Where notes are stored. `~` is expanded. |
| `--workspace NAME` (or positional arg) | `PAD_WORKSPACE` | `default` | Workspace to open. |
| `--autosave N` | `PAD_AUTOSAVE` | `3` | Autosave interval in seconds (accepts `5` or `5s`). `0` disables periodic autosave. |
| `--config PATH` | `PAD_CONFIG` | `$XDG_CONFIG_HOME/pad/config.toml` | Config file location. |
| `--print` / `--tab TITLE` | — | — | Dump notes to stdout and exit (no TUI). |
| `--new TITLE` / `--append TITLE` / `--content TEXT` | — | — | Create or append to a note from CLI/stdin and exit (no TUI). |
| `--version` | — | — | Print version and exit. |

### Config file

A simple `key = value` file (TOML-ish). Default location `~/.config/pad/config.toml`:

```toml
dir = "~/notes"
workspace = "work"
autosave = 5
```

Set defaults once in your shell profile, e.g.:

```sh
# put notes wherever you want
export PAD_DIR="$HOME/Documents/scratch"
export PAD_WORKSPACE="work"
export PAD_AUTOSAVE="5"
```

For example, to keep notes inside an existing [`nb`](https://github.com/xwmx/nb) setup:

```sh
export PAD_DIR="$HOME/.nb/scratchpad"
```

## How it works

- **Storage** — each tab is a file `<data-dir>/<workspace>/NN-slug.md`. The numeric prefix `NN` fixes tab order; the slug is derived from the tab title.
- **Saving** — autosaves on the configured interval, and also on tab switch, rename, workspace switch, and quit. A `•` next to a tab title means it has unsaved changes in the buffer.
- **Renaming** — renames the underlying file (keeps its numeric prefix, so order is preserved).
- **Deleting** — removes the file from disk after a `y`/`n` confirm. If you delete the last tab, a fresh empty `untitled` tab is created so there's always somewhere to type.

## Layout

```
pad/
├── main.go         # TUI model, update loop, keybindings
├── view.go         # rendering (tab bar, footer, overlays)
├── store.go        # file storage: workspaces, notes, slugs, paths
├── config.go       # flag/env/default resolution
├── *_test.go       # unit tests
└── go.mod
```

## Development

```sh
make build      # build ./pad with version stamped from git
make install    # build into $PREFIX/bin (default ~/.local)
make test       # go test ./...
make check      # fmt + vet + test
make demo       # re-render demo.gif (needs vhs)
```

CI (build, vet, gofmt, race tests) runs on every push and PR.

### Releasing

Tag and push; the release workflow runs [GoReleaser](https://goreleaser.com) to
build cross-platform binaries, publish a GitHub release, and update the Homebrew
cask in [`ianaya89/homebrew-tap`](https://github.com/ianaya89/homebrew-tap):

```sh
git tag v0.1.0
git push origin v0.1.0
```

The binary's `--version` is stamped from the tag. Update [`CHANGELOG.md`](CHANGELOG.md) before tagging a release.

**One-time setup for the Homebrew step** — the default `GITHUB_TOKEN` can't write
to the tap repo, so add a Personal Access Token with write access to it:

```sh
# create a fine-grained PAT with "Contents: read & write" on ianaya89/homebrew-tap,
# then store it as a secret on this repo:
gh secret set HOMEBREW_TAP_TOKEN --repo ianaya89/scratchpad
```

Without that secret the release still succeeds, but the cask update step fails.

## License

MIT
