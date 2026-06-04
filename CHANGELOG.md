# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `--new TITLE` / `--append TITLE` with `--content` or stdin: create or append to
  notes from the command line / pipes, no TUI.
- `--list` prints workspace names, or the tabs of a named workspace (`pad --list work`).
- Each workspace reopens on the tab you last used.
- `^x` toggles a Markdown task checkbox (`- [ ]` ↔ `- [x]`) on the current line.
- Undo/redo (`^z` / `^y`), tracked per tab with debounced checkpoints.
- Live word/character count in the footer while editing.
- Live split preview (`^b`): edit raw Markdown on the left with a rendered view
  updating on the right as you type.
- `alt+1`…`alt+9` jumps straight to a tab.
- Tab bar scrolls when tabs overflow the terminal width, with `‹N` / `N›` counters.
- Empty-note placeholder now hints the core keybindings.

### Changed

- Move tab is now `^→` / `^←` (was `alt+H`/`alt+L`, which is unreliable on macOS);
  the alt chords stay as fallbacks.
- Help overlay key is now `^/` (was `^g`); `F1` still works.

### Fixed

- Split/preview were slow because Glamour re-rendered (and rebuilt its renderer)
  on every keystroke. Rendering is now debounced and the renderer cached.
- Dropped Glamour's `WithAutoStyle` terminal background query (a synchronous OSC
  call that could stall); the preview theme is now fixed (`dark`) and overridable
  via `PAD_THEME`.
- Transient status messages (e.g. "saved", "moved tab") auto-clear after a few seconds.

## [0.1.0] - 2026-06-04

First public release.

### Added

- Tabbed scratchpad TUI built with Bubble Tea — each tab is a quick note.
- Named **workspaces**: independent sets of tabs (`pad`, `pad ideas`, `^e` picker).
- Plain-Markdown file storage (`<dir>/<workspace>/NN-slug.md`), greppable and hand-editable.
- **Find** across tabs by title and body (`^f`).
- **Markdown preview** of the current note, rendered with Glamour (`^o`).
- **Reorder** the active tab with `alt+H` / `alt+L`.
- Rename (`^r`), delete with confirmation (`^d`), and a help overlay (`^g` / `F1`).
- Atomic note writes (temp file + rename) to avoid corruption on crash.
- Autosave on an interval plus on tab switch, rename, workspace switch, and quit.
- `--print` / `--tab` to dump notes to stdout for scripting (no TUI).
- Configurable storage and behavior with precedence **flag → env → config file → default**
  (`--dir`/`PAD_DIR`, `--workspace`/`PAD_WORKSPACE`, `--autosave`/`PAD_AUTOSAVE`,
  `~/.config/pad/config.toml`).
- Tab switching on `^p` / `^n` with `alt+h`/`alt+l`, arrow, and shift-arrow fallbacks.
- Distribution: GitHub Actions CI, GoReleaser cross-platform binaries, and a Homebrew cask.

[Unreleased]: https://github.com/ianaya89/scratchpad/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/ianaya89/scratchpad/releases/tag/v0.1.0
