package clog

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gechr/clog/fx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nilInertTypes are the exported pointer types whose entire exported method set
// must be inert on a nil receiver, per the nil-receiver contract in the package
// documentation. TestNilReceiverInert exercises every method of each.
var nilInertTypes = map[string]reflect.Type{
	"Event":  reflect.TypeFor[*Event](),
	"Logger": reflect.TypeFor[*Logger](),
}

// nilPanicTypes are the exported pointer types deliberately left to panic on a
// nil receiver, with the reason. Nothing in the package returns a nil one, so a
// nil receiver can only come from a caller declaring it and never assigning it
// - and unlike a package-level logger initialised behind a sync.Once, none of
// these is ever held that way: they are per-call values.
var nilPanicTypes = map[string]string{
	"ColorMode": "UnmarshalText's callers (flag, encoding/json) always pass an addressable receiver",
	"Context": "embeds a value-typed field builder, so its promoted field methods dereference the " +
		"receiver before any guard could run; Logger.With never returns nil",
	"DividerBuilder": "only built by Logger.Divider, which renders to io.Discard on a nil Logger",
	"Output":         "only built by NewOutput and friends; Logger.Output never returns nil",
	"Printer":        "only built by Logger.Print, which prints to io.Discard on a nil Logger",
}

// TestNilReceiverContractCoversEveryExportedType enforces the contract by
// enumeration rather than by spot checks: every exported type in the package
// with an exported pointer-receiver method must be classified as either inert
// on nil or deliberately exempt, so a new type cannot quietly arrive without a
// decision. For the inert types it also checks that reflection sees exactly the
// methods the source declares, since reflection is what drives
// TestNilReceiverInert.
func TestNilReceiverContractCoversEveryExportedType(t *testing.T) {
	declared := exportedPointerMethods(t)
	require.NotEmpty(t, declared)

	for _, typeName := range slices.Sorted(maps.Keys(declared)) {
		typ, inert := nilInertTypes[typeName]
		reason, exempt := nilPanicTypes[typeName]

		switch {
		case inert && exempt:
			t.Errorf("type %s is classified as both inert and exempt", typeName)
		case exempt:
			assert.NotEmpty(t, reason, "type %s is exempt without a reason", typeName)
		case !inert:
			t.Errorf("exported type %s has exported pointer-receiver methods but is in neither "+
				"nilInertTypes nor nilPanicTypes; classify it (see the package documentation)",
				typeName)
		default:
			assert.Equal(t, slices.Sorted(slices.Values(declared[typeName])),
				slices.Sorted(maps.Keys(methodSet(typ))),
				"reflection must see exactly the exported methods %s declares", typeName)
		}
	}
}

// exportedPointerMethods parses the package sources - including files excluded
// by build constraints, so the enumeration is platform-independent - and maps
// each exported type to the exported methods declared on its pointer.
func exportedPointerMethods(t *testing.T) map[string][]string {
	t.Helper()

	entries, err := os.ReadDir(".")
	require.NoError(t, err)

	fset := token.NewFileSet()
	methods := make(map[string][]string)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		require.NoError(t, err)

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || !fn.Name.IsExported() {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			if recv, ok := star.X.(*ast.Ident); ok && recv.IsExported() {
				methods[recv.Name] = append(methods[recv.Name], fn.Name.Name)
			}
		}
	}
	return methods
}

// TestNilReceiverInert calls every exported method of every inert type on a nil
// receiver. A method added later without a guard fails here rather than as a
// SIGSEGV in a consumer.
func TestNilReceiverInert(t *testing.T) {
	for _, typeName := range slices.Sorted(maps.Keys(nilInertTypes)) {
		typ := nilInertTypes[typeName]

		t.Run(typeName, func(t *testing.T) {
			methods := methodSet(typ)
			require.NotEmpty(t, methods)

			for name, method := range methods {
				t.Run(name, func(t *testing.T) {
					require.NotPanics(t, func() {
						method.Func.Call(nilReceiverArgs(method.Type, reflect.Zero(typ)))
					})
				})
			}
		})
	}
}

// nilReceiverArgs builds the argument list for calling method sig on recv: the
// receiver followed by a zero value per fixed parameter, with variadic
// parameters left empty. A [context.Context] parameter gets
// [context.Background] instead of the zero value, because a nil context panics
// inside the context package itself and would say nothing about the receiver.
func nilReceiverArgs(sig reflect.Type, recv reflect.Value) []reflect.Value {
	ctxType := reflect.TypeFor[context.Context]()

	args := []reflect.Value{recv}
	for i := 1; i < sig.NumIn(); i++ {
		if sig.IsVariadic() && i == sig.NumIn()-1 {
			break
		}
		if in := sig.In(i); in == ctxType {
			args = append(args, reflect.ValueOf(context.Background()))
		} else {
			args = append(args, reflect.Zero(in))
		}
	}
	return args
}

func TestNilLoggerEventsAreDropped(t *testing.T) {
	var logger *Logger

	require.NotPanics(t, func() {
		logger.Debug().Str("k", "v").Int("n", 1).Msg("dropped")
		logger.Log(LevelWarn).Dict("d", logger.Dict()).Send()
		logger.LogFields(LevelError, time.Now(), "dropped", []Field{{Key: "k", Value: "v"}})
	})

	assert.Nil(t, logger.Info())
	assert.Nil(t, logger.Fatal(), "a nil logger must not exit the process")
	assert.Nil(t, logger.Dict())
}

// TestNilEventChainsAreNoOps checks the other half of the chain: a nil Event,
// whether it came from a nil logger or a disabled level, must survive being
// built up and finalised.
func TestNilEventChainsAreNoOps(t *testing.T) {
	var event *Event

	require.NotPanics(t, func() {
		event.Str("k", "v").Int("n", 1).Dict("d", event).Err(io.EOF).Msg("dropped")
		event.Path("p", "/tmp/x").Link("l", "https://example.com", "text").Send()
		event.Func(func(*Event) { t.Error("Func must not call fn on a nil event") }).Msgf("%d", 1)
		event.Elapsed("took").Deadline("timeout", time.Second).Discard().Send()
	})

	// Several field methods have a second guard of their own (a nil error, a
	// nil dictionary, a line number below one). Feed them arguments that get
	// past it, so the nil-receiver guard is the only thing between the call and
	// a dereference.
	require.NotPanics(t, func() {
		event.AnErr("err", io.EOF).
			Stringer("took", time.Second).
			Dict("d", Default().Dict().Str("a", "b")).
			Line("src", "/tmp/x.go", 12).
			When(true, func(*Event) { t.Error("When must not call fn on a nil event") }).
			Msg("dropped")
	})

	assert.True(t, event.Disabled())
	assert.False(t, event.Enabled())
	assert.Nil(t, event.Str("k", "v"), "field methods must keep returning nil")
}

func TestNilLoggerAccessorsReportDefaults(t *testing.T) {
	var logger *Logger

	assert.Equal(t, LevelInfo, logger.Level())
	assert.False(t, logger.LevelEnabled(LevelFatal))
	assert.Equal(t, DefaultFieldFormats(), logger.FieldFormats())
	assert.Same(t, discardLogger.Output(), logger.Output())
}

func TestNilLoggerMutatorsAreNoOps(t *testing.T) {
	var logger *Logger

	logger.SetLevel(LevelTrace)
	logger.SetPercentPrecision(3)
	logger.SetParts(PartMessage)

	assert.Equal(t, LevelInfo, logger.Level())
	assert.Equal(t, DefaultFieldFormats(), logger.FieldFormats())

	// The shared inert logger backing nil-receiver builders must not pick up
	// configuration aimed at the nil one.
	assert.Equal(t, LevelInfo, discardLogger.Level())
	assert.Equal(t, DefaultFieldFormats(), discardLogger.FieldFormats())
}

// TestNilLoggerChainsStayInert follows each value a nil logger hands back to
// the point where it is next dereferenced. A guard that returns a live object
// wrapping a nil logger would relocate the panic further from its cause, which
// is worse than crashing at the call site.
func TestNilLoggerChainsStayInert(t *testing.T) {
	var logger *Logger

	require.NotPanics(t, func() {
		logger.With().
			Str("component", "auth").
			Path("cfg", "/etc/app.yaml").
			Indent().
			Tree(TreeMiddle).
			Symbol("*").
			Logger().Info().Str("user", "john").Msg("dropped")

		logger.Print().Mode(JSONFlat).JSON(map[string]int{"a": 1})
		logger.Divider().Char('=').Align(AlignCenter).Width(20).Msg("Build Phase")

		out := logger.Output()
		_, _ = out.Width(), out.IsTTY()
		out.WriteLine("dropped")
	})

	assert.NotNil(t, logger.Output().Writer(), "an inert output still needs a writer")
	assert.NotNil(t, logger.With().Logger(), "a sub-logger of a nil logger must not be nil")
}

func TestNilLoggerWithContextStoresNothing(t *testing.T) {
	var logger *Logger

	ctx := context.Background()
	got := logger.WithContext(ctx)

	assert.Equal(t, ctx, got)
	assert.Nil(t, got.Value(ctxKey{}), "a nil logger must not be stored in the context")
}

// TestLoggerAccessorsNeverReturnNil covers the package-level and constructor
// paths that resolve to a logger: none may hand a consumer a nil one, so the
// inert contract stays a safety net rather than a load-bearing part of normal
// use.
func TestLoggerAccessorsNeverReturnNil(t *testing.T) {
	var logger *Logger

	assert.NotNil(t, Default())
	assert.NotNil(t, New(nil))
	assert.NotNil(t, NewWriter(io.Discard))
	assert.NotNil(t, Ctx(context.Background()))
	assert.NotNil(t, Ctx(nil)) //nolint:staticcheck // a nil ctx must fall back to Default
	assert.Same(t, Default(), Ctx(logger.WithContext(context.Background())),
		"a nil logger is not stored, so Ctx must fall back to Default")
}

func TestNilLoggerBuildersRenderNowhere(t *testing.T) {
	var logger *Logger

	require.NotPanics(t, func() {
		logger.Print().JSON(map[string]int{"a": 1})
		logger.Divider().Msg("Build Phase")
		logger.Divider().Send()
	})

	assert.NotNil(t, logger.Spinner("working"))
	assert.NotNil(t, logger.Bar("uploading", 10))
	assert.NotNil(t, logger.Pulse("waiting"))
	assert.NotNil(t, logger.Shimmer("loading"))
	assert.NotNil(t, logger.Group(context.Background()))

	// A non-nil builder proves nothing on its own: it is the render path that
	// reaches back into the logger, so run a task through a group to exercise
	// it.
	group := logger.Group(context.Background())
	result := group.Add(logger.Spinner("working")).Run(func(context.Context) error { return nil })
	group.Wait()

	require.NoError(t, result.Msg("done"))
}

// TestNilLoggerGroupFallbackLoggerIsInert covers the one place a logger is
// stored for later use instead of being used immediately: [fx.Group] hands its
// own logger to any builder added without one, and the render path calls back
// into it for the task's settings. The group must therefore hold an inert
// logger, not the nil one.
func TestNilLoggerGroupFallbackLoggerIsInert(t *testing.T) {
	var logger *Logger

	group := logger.Group(context.Background())
	result := group.Add(fx.NewBuilder(fx.BuilderConfig{
		Level:   LevelInfo,
		Message: "external builder",
		Mode:    fx.AnimationNone,
	})).Run(func(context.Context) error { return nil })
	group.Wait()

	require.NoError(t, result.Msg("done"))
}

func TestNilLoggerInputReturnsEOF(t *testing.T) {
	var logger *Logger

	got, err := logger.Input("Name: ")
	require.ErrorIs(t, err, io.EOF)
	assert.Empty(t, got)

	got, err = logger.Password("Password: ")
	require.ErrorIs(t, err, io.EOF)
	assert.Empty(t, got)
}
