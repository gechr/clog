# Input

Prompt for a line of input from the terminal:

```go
name, err := clog.Input("Name: ")
```

`Input` writes the prompt to the logger's output, then reads a line from its input (`os.Stdin` by default). Reading is a plain newline-delimited read - the terminal's own canonical line editing already provides backspace, Ctrl-U, and Ctrl-W for free, and Ctrl-C terminates the process exactly as it would at any other prompt, so `Input` never hangs waiting for input that isn't coming.

Prompt from one goroutine at a time: `Input` and `Password` are not safe for concurrent use, and log lines emitted from other goroutines while a prompt is pending may interleave with it.

## Password

`Password` is `Input` with typed characters not echoed to the screen, via [`term.ReadPassword`](https://pkg.go.dev/golang.org/x/term#ReadPassword):

```go
pass, err := clog.Password("Password: ")
```

`term.ReadPassword` only disables local echo; it leaves the terminal's own Ctrl-C/Ctrl-Z handling intact, so `Password` deliberately defers to it rather than a bespoke raw-mode reader. Outside of a real terminal (piped input, tests), `Password` behaves exactly like `Input` - there's no echo to suppress.

## Options

Both `Input` and `Password` accept `InputOption` values. `Password` is exactly `Input` with `clog.WithSensitive(true)` applied:

```go
secret, err := clog.Input("Token: ", clog.WithSensitive(true))
```

Use `WithClearOnSuccess`, `WithClearOnError`, or `WithClearOnDone` to erase the prompt/input line after the read completes:

```go
name, err := clog.Input("Name: ", clog.WithClearOnDone())
```

Clearing only applies when the logger's output is a terminal. Redirected output keeps the prompt text and never receives cursor-control escape sequences.

## Prompt marker

`SetPromptMarker` sets a leading marker prepended to every prompt, so all prompts on a logger share one visual identity without threading the marker through each call:

```go
clog.SetPromptMarker("❯ ")
name, err := clog.Input("Name: ") // "❯ Name: "
```

The marker is rendered with the `Prompt` style (see [Styles](styles.md)), letting it carry its own color independent of the prompt message:

```go
markerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4"))
clog.SetStyles(&style.Config{Prompt: &markerStyle})
```

Pass `""` to disable the marker. It is dropped, along with all other colors, when the logger's output has color disabled.

## Testing

`SetInput` overrides the reader `Input`/`Password` read from, so prompts can be driven in tests without a real terminal:

```go
clog.SetInput(strings.NewReader("alice\n"))
name, _ := clog.Input("Name: ") // "alice", no TTY required
```

## Per-logger usage

`Input` and `Password` are also available on any `*clog.Logger`:

```go
logger := clog.NewWriter(os.Stdout)
name, err := logger.Input("Name: ")
```
