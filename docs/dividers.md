# Dividers

Print horizontal rules to visually separate sections of CLI output:

```go
clog.Divider().Send()
// ────────────────────────────────────────────────────────────────────────────────
```

## Title

Add a title to label the section:

```go
clog.Divider().Msg("Build Phase")
// ─── Build Phase ────────────────────────────────────────────────────────────────
```

## Custom Character

Change the line character:

```go
clog.Divider().Char('═').Send()
// ════════════════════════════════════════════════════════════════════════════════

clog.Divider().Char('═').Msg("Deployment")
// ═══ Deployment ═════════════════════════════════════════════════════════════════
```

## Title Alignment

Control where the title sits within the line:

```go
// Left-aligned (default)
clog.Divider().Msg("Left")
// ─── Left ───────────────────────────────────────────────────────────────────────

// Centered
clog.Divider().Align(clog.AlignCenter).Msg("Center")
// ───────────────────────────────── Center ───────────────────────────────────────

// Right-aligned
clog.Divider().Align(clog.AlignRight).Msg("Right")
// ────────────────────────────────────────────────────────────────────── Right ───
```

## Width

The divider fills the full terminal width. When the terminal width cannot be detected (non-TTY output, piped commands), it defaults to 80 columns.

Override the width manually with `Width`:

```go
clog.Divider().Width(40).Send()
// ────────────────────────────────────────

clog.Divider().Width(40).Msg("Short")
// ─── Short ──────────────────────────────
```

## Styling

Divider styles are configured through the `Styles` struct:

```go
styles := clog.DefaultStyles()
styles.DividerLine = new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))   // blue line
styles.DividerTitle = new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))) // bold yellow title
clog.SetStyles(styles)
```

| Style Field    | Default | Description                   |
| -------------- | ------- | ----------------------------- |
| `DividerLine`  | Faint   | Style for the line characters |
| `DividerTitle` | Bold    | Style for the title text      |
