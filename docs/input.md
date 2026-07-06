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
