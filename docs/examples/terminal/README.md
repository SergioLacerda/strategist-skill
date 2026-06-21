# Strategist · Terminal Loaders

Loading animations for the **strategist** CLI, in the dungeon/grimoire register —
delivered in two interchangeable targets:

| File | Target | Dependencies |
|------|--------|--------------|
| `phosphor-type.sh` | Bash (any POSIX shell) | none |
| `spell-charge.sh`  | Bash (any POSIX shell) | `awk` (token meter only) |
| `phosphor_type.go` | Go — Bubble Tea + Lip Gloss | `bubbletea`, `lipgloss` |
| `spell_charge.go`  | Go — Bubble Tea + Lip Gloss | `bubbletea`, `lipgloss` |

Two loaders are included:

- **Phosphor Type** — a typewriter stream. Text appears character by character in
  phosphor green with a blinking block cursor (`█`). Use while the agent is
  "thinking" / streaming, when you have no measurable progress.
- **Spell Charge** — a determinate progress bar (`█ ▓ ░`) with a percentage and a
  live token meter (`↑ 12.3k tokens`). Use when you *can* measure progress
  (steps done / total, bytes, tokens).

---

## How it works (the mechanism)

Both targets paint the **same** picture; only the rendering host differs. The core
ideas are plain terminal primitives — no framebuffer, no canvas.

### 1. ANSI escape codes do everything

A terminal is a stream of bytes. Special **escape sequences** (starting with the
`ESC` byte, written `\e` / `\033` / `\x1b`) are interpreted as commands instead of
printed as text:

| Sequence | Meaning |
|----------|---------|
| `\e[38;5;Nm` | set foreground to 256-color index `N` |
| `\e[0m` | reset all attributes |
| `\e[?25l` / `\e[?25h` | hide / show the cursor |
| `\r` (carriage return) | move the write head to **column 0 of the current line** |

That's the whole toolbox.

### 2. Animation = rewrite the same line

There is no "update element". To animate a single line you:

1. Print the frame.
2. Emit `\r` to jump back to the start of the line (**without** a newline, so you
   stay on the same row).
3. Print the next frame over the top.
4. Sleep a few tens of milliseconds. Repeat.

Because every frame is the same width, each redraw fully overwrites the previous
one. This is why the progress bar appears to "fill in place".

> The cursor is hidden (`\e[?25l`) during animation so it doesn't flicker at the
> end of the line, and restored (`\e[?25h`) on exit — including on Ctrl-C, via a
> `trap` (bash) or normal program teardown (Go).

### 3. Phosphor Type — typing + blinking

- Keep a growing substring of the message; each frame appends one more character.
  The bash version reads the string **one character at a time** (`read -n1`) and
  the Go version slices a `[]rune` — both so the multibyte `❯` glyph never gets cut
  in half.
- A block `█` is printed after the text as a fake cursor.
- Once the full string is typed, the cursor alternates between `█` and a space on a
  timer to **blink**.

### 4. Spell Charge — the bar + meter

For a bar of `width` cells and progress `p`:

```
cell i < p   → █   (filled,  amber)
cell i == p  → ▓   (leading,  ember  — the "charging" edge)
cell i > p   → ░   (track,    dim)
```

- Percentage is `p * 100 / width`.
- The **token meter** after the percentage is a counter that grows with progress
  (`p * 4400` here as a stand-in), formatted to `k` past 1000. **Swap this for your
  real token count** when wiring it into strategist (see below).

### 5. Color palette (matches the design tokens)

256-color indices were chosen to match the Strategist grimoire palette:

| Role | ANSI 256 | Hex (design) |
|------|----------|--------------|
| phosphor green | `114` | `#74cf8e` |
| amber (resin)  | `179` | `#e8c25a` |
| ember (edge)   | `173` | `#cf7a2c` |
| track / dim    | `94`  | `#5a4a2a` |

---

## Running the previews

Bash (no build step):

```bash
chmod +x phosphor-type.sh spell-charge.sh
./phosphor-type.sh
./spell-charge.sh
```

Go (each file is its own `package main` — run from separate folders, or merge):

```bash
go get github.com/charmbracelet/bubbletea github.com/charmbracelet/lipgloss
go run ./phosphor_type.go     # any key quits
go run ./spell_charge.go      # any key quits
```

---

## Integrating into the strategist CLI

The demos loop forever for preview purposes. In production you drive them from
**real** state.

### Determinate work (Spell Charge)

Render the bar from your actual progress instead of a frame counter:

```go
// p := currentStep ; width := totalSteps
// tokens := llmClient.TokensUsed()
```

Pattern: run the work, and after each unit completes, recompute `p`/`tokens` and
redraw. With Bubble Tea, send a custom `progressMsg{done, total, tokens}` from your
worker goroutine via `program.Send(...)` and update the model from it — drop the
synthetic `tickMsg` entirely.

### Indeterminate work (Phosphor Type)

Keep the `tickMsg` timer (you have no progress to show), run the loader on the main
loop, do the real work on a goroutine, and call `tea.Quit` (or `program.Quit()`)
when it finishes:

```go
go func() {
    doRealWork()
    program.Send(doneMsg{})   // model returns tea.Quit on doneMsg
}()
```

### Where the host's spinner wins

If strategist runs **inside** an IDE/agent host (Cursor, Claude Code, Copilot), the
host draws its own "thinking" spinner and a skill cannot replace it. These loaders
apply to **strategist's own stdout** — i.e. when the Go binary prints progress
itself. That's the surface you control.

### Plain-Go alternative (no Bubble Tea)

If you don't want the Bubble Tea dependency, the bash logic ports directly: a
`for` loop that builds the frame string, prints it after `\r`, and `time.Sleep`s.
Use `fmt.Print("\033[?25l")` / `"\033[?25h"` for the cursor and `\033[38;5;Nm` for
color. Bubble Tea is recommended only because it handles resize, input, and clean
teardown for you.

---

## Files

```
terminal/
├── README.md          ← this file
├── phosphor-type.sh   ← typewriter stream  (bash)
├── spell-charge.sh    ← progress + meter   (bash)
├── phosphor_type.go   ← typewriter stream  (Go / Bubble Tea)
└── spell_charge.go    ← progress + meter   (Go / Bubble Tea)
```

A live HTML preview of both (faithful to the terminal output) lives in
`Terminal Loaders.dc.html` at the project root.
