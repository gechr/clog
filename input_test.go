package clog

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputCookedReadsLine(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("hello\n"))

	got, err := l.Input("Name: ")

	require.NoError(t, err)
	assert.Equal(t, "hello", got)
	assert.Equal(t, "Name: ", out.String())
}

func TestInputCookedTrimsCRLF(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("hello\r\n"))

	got, err := l.Input("Name: ")

	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestInputCookedEOFWithoutNewline(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("hello"))

	got, err := l.Input("Name: ")

	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestInputCookedEmptyEOF(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader(""))

	got, err := l.Input("Name: ")

	require.ErrorIs(t, err, io.EOF)
	assert.Empty(t, got)
}

func TestPasswordCookedReadsLine(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("hunter2\n"))

	got, err := l.Password("Password: ")

	require.NoError(t, err)
	assert.Equal(t, "hunter2", got)
	assert.Equal(t, "Password: ", out.String())
}

func TestPackageLevelInputUsesDefault(t *testing.T) {
	orig := Default
	t.Cleanup(func() { Default = orig })

	var out bytes.Buffer
	Default = New(TestOutput(&out))
	Default.SetInput(strings.NewReader("value\n"))

	got, err := Input("Prompt: ")

	require.NoError(t, err)
	assert.Equal(t, "value", got)
}

func TestPackageLevelPasswordUsesDefault(t *testing.T) {
	orig := Default
	t.Cleanup(func() { Default = orig })

	var out bytes.Buffer
	Default = New(TestOutput(&out))
	Default.SetInput(strings.NewReader("value\n"))

	got, err := Password("Prompt: ")

	require.NoError(t, err)
	assert.Equal(t, "value", got)
}

func TestReadLineCookedTrimsCRLF(t *testing.T) {
	got, err := readLineCooked(bufio.NewReader(strings.NewReader("ab\r\n")))

	require.NoError(t, err)
	assert.Equal(t, "ab", got)
}

func TestReadLineCookedEOFReturnsPartialLine(t *testing.T) {
	got, err := readLineCooked(bufio.NewReader(strings.NewReader("ab")))

	require.NoError(t, err)
	assert.Equal(t, "ab", got)
}

func TestReadLineCookedEOFEmptyReturnsError(t *testing.T) {
	got, err := readLineCooked(bufio.NewReader(strings.NewReader("")))

	require.ErrorIs(t, err, io.EOF)
	assert.Empty(t, got)
}

// Regression test: a fresh bufio.Reader per call would read ahead into the
// underlying reader and discard whatever it buffered past the first line,
// silently losing every prompt after the first when several lines are
// available up front (e.g. piped input, as here). The cached inputSource
// must persist its bufio.Reader across calls.
func TestInputMultiplePromptsShareBufferedReader(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("Alice\nhunter2\nsecret-token\n"))

	name, err := l.Input("Name: ")
	require.NoError(t, err)
	assert.Equal(t, "Alice", name)

	pass, err := l.Password("Password: ")
	require.NoError(t, err)
	assert.Equal(t, "hunter2", pass)

	token, err := l.Input("Token: ")
	require.NoError(t, err)
	assert.Equal(t, "secret-token", token)
}

// Sensitive input over a non-TTY reader (a pipe, or this test's in-memory
// reader) has no terminal to mask, so it falls back to the same plain read
// as non-sensitive input. Masking via [term.ReadPassword] only kicks in for
// a real *os.File terminal, which isn't something a unit test can fake.
func TestInputAppliesSensitiveOption(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("secret\n"))

	got, err := l.Input("Value: ", WithSensitive(true))

	require.NoError(t, err)
	assert.Equal(t, "secret", got)
}

func TestInputContextCancelledWhileBlocked(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	l.SetInput(r) // nothing ever written: the read blocks

	ctx, cancel := context.WithCancel(t.Context())
	go cancel()

	got, err := l.InputContext(ctx, "Name: ")

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, got)
}

func TestInputContextReadsLine(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	r, w := io.Pipe()
	defer r.Close()
	l.SetInput(r)
	go func() {
		_, writeErr := io.WriteString(w, "hello\n")
		_ = writeErr
	}()

	got, err := l.InputContext(t.Context(), "Name: ")

	require.NoError(t, err)
	assert.Equal(t, "hello", got)
}

func TestPasswordContextCancelledWhileBlocked(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	l.SetInput(r)

	ctx, cancel := context.WithCancel(t.Context())
	go cancel()

	got, err := l.PasswordContext(ctx, "Password: ")

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, got)
}

func TestInputWithFieldsRendersPrompt(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("hunter2\n"))

	got, err := l.Password("Enter passphrase", WithFields(func(e *Event) {
		e.Str("user", "alice").Int("attempts", 3)
	}))

	require.NoError(t, err)
	assert.Equal(t, "hunter2", got)
	assert.Equal(t, "Enter passphrase user=alice attempts=3: ", out.String())
}

func TestInputWithFieldsEmptyFields(t *testing.T) {
	var out bytes.Buffer
	l := New(TestOutput(&out))
	l.SetInput(strings.NewReader("x\n"))

	got, err := l.Input("Name", WithFields(func(*Event) {}))

	require.NoError(t, err)
	assert.Equal(t, "x", got)
	assert.Equal(t, "Name: ", out.String())
}
