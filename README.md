# pad

A tabbed terminal scratchpad. Each **tab** is a quick note; each **workspace** is a named set of tabs. Built with Go + [Bubble Tea](https://github.com/charmbracelet/bubbletea).

It is intentionally separate from any other note tool — it never touches your `nb` notebook or `scratch.md`. Notes are plain Markdown files on disk, so they stay greppable and editable by hand.

## What is a "workspace"?

A workspace is just a **named drawer of tabs**. Think of it as one project's scratch area.

- Open `pad` with no argument → the `default` workspace.
- Open `pad ideas` → a workspace called `ideas` (created on first use).
- Each workspace has its own independent set of tabs. Switching workspaces swaps the whole tab bar.

Use them however you like: one per project, one per context (`work`, `personal`, `meeting`), or just live in `default` forever. You don't have to think about workspaces at all if you don't want to — the default one is always there.

```
~/.nb/scratchpad/
├── default/
│   ├── 1-untitled.md
│   └── 2-todo.md
└── ideas/
    └── 1-feature-brainstorm.md
```

## Install

Requires Go 1.26+.

```sh
git clone <repo> ~/pad
cd ~/pad
GOSUMDB=off go build -o ~/.local/bin/pad .   # ensure ~/.local/bin is on PATH
```

> `GOSUMDB=off` is only needed in environments where the Go checksum database is unreachable. Drop it if your `go` toolchain has normal network access.

## Usage

```sh
pad            # open the "default" workspace
pad ideas      # open/create the "ideas" workspace
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

## How it works

- **Storage** — each tab is a file `~/.nb/scratchpad/<workspace>/NN-slug.md`. The numeric prefix `NN` fixes tab order; the slug is derived from the tab title.
- **Saving** — autosaves every 3 seconds, and also on tab switch, rename, workspace switch, and quit. A `•` next to a tab title means it has unsaved changes in the buffer.
- **Renaming** — renames the underlying file (keeps its numeric prefix, so order is preserved).
- **Deleting** — removes the file from disk after a `y`/`n` confirm. If you delete the last tab, a fresh empty `untitled` tab is created so there's always somewhere to type.

## Layout

```
pad/
├── main.go        # TUI model, update loop, keybindings
├── view.go        # rendering (tab bar, footer, overlays)
├── store.go       # file storage: workspaces, notes, slugs, paths
├── store_test.go  # unit tests for the storage layer
└── go.mod
```

## Development

```sh
GOSUMDB=off go build -o ~/.local/bin/pad .
GOSUMDB=off go test ./...
go vet ./...
```
