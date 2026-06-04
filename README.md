# pad

A tabbed terminal scratchpad. Each **tab** is a quick note; each **workspace** is a named set of tabs. Built with Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea).

Notes are plain Markdown files on disk, so they stay greppable and editable by hand. By default they live under your XDG data dir — but the storage location and other behavior are fully configurable (see [Configuration](#configuration)).

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

## Install

Requires Go 1.26+.

```sh
git clone https://github.com/ianaya89/scratchpad ~/pad
cd ~/pad
go build -o ~/.local/bin/pad .   # ensure ~/.local/bin is on PATH
```

> If your Go toolchain can't reach the checksum database (sandboxed/offline), prefix builds with `GOSUMDB=off`.

## Usage

```sh
pad                  # open the "default" workspace
pad ideas            # open/create the "ideas" workspace
pad --dir ~/notes    # store notes under ~/notes instead of the default
pad --help           # full flag reference
```

### Keybindings

| Key | Action |
| --- | --- |
| `^t` | New tab |
| `^d` (or `^w`) | Delete current note (asks to confirm) |
| `^}` | Next tab |
| `^{` | Previous tab |
| `^r` | Rename current tab |
| `^e` | Workspace picker (switch, or `n` to create new) |
| `^s` | Save now |
| `^q` (or `^c`) | Save everything and quit |

Fallback tab-switch keys are also bound in case your terminal grabs `ctrl+{`/`ctrl+}`: `ctrl+←`/`ctrl+→`, `alt+h`/`alt+l`, `shift+←`/`shift+→`.

> **Terminal note:** `ctrl+{` is `ctrl+shift+[`. A few terminals send `esc` for `ctrl+[` or don't deliver these chords distinctly. If `^{`/`^}` don't move tabs in your terminal, use one of the fallback chords above.

## Configuration

Everything resolves with the precedence **flag → environment variable → default**.

| Flag | Env var | Default | Description |
| --- | --- | --- | --- |
| `--dir PATH` | `PAD_DIR` | `$XDG_DATA_HOME/pad`, else `~/.local/share/pad` | Where notes are stored. `~` is expanded. |
| `--workspace NAME` (or positional arg) | `PAD_WORKSPACE` | `default` | Workspace to open. |
| `--autosave N` | `PAD_AUTOSAVE` | `3` | Autosave interval in seconds (accepts `5` or `5s`). `0` disables periodic autosave. |
| `--version` | — | — | Print version and exit. |

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
go build -o ~/.local/bin/pad .
go test ./...
go vet ./...
```

## License

MIT
