package kloghandler_test

import (
	"bytes"
	"flag"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/gechr/clog"
	"github.com/gechr/clog/kloghandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/klog/v2"
)

// setKlog installs the bridge for the duration of the test. klog's logger and
// verbosity are process-wide, so the previous state is snapshotted first.
func setKlog(t *testing.T, logger *clog.Logger, opts *kloghandler.Options) {
	t.Helper()

	t.Cleanup(klog.CaptureState().Restore)
	kloghandler.SetKlog(logger, opts)
}

func TestSetKlogStructured(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), nil)

	klog.InfoS("reconciled", "name", "example")

	assert.Equal(t, "INF ℹ️ reconciled name=example\n", buf.String())
}

func TestSetKlogErrorS(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), nil)

	klog.ErrorS(errBoom, "sync failed", "attempt", 3)

	assert.Equal(t, "ERR ❌ sync failed error=boom attempt=3\n", buf.String())
}

func TestSetKlogVerbosity(t *testing.T) {
	var buf bytes.Buffer
	verbosity := 2
	setKlog(t, newTestLogger(&buf), &kloghandler.Options{Verbosity: &verbosity})

	klog.V(1).InfoS("chatty")
	klog.V(2).InfoS("noisy")
	klog.V(3).InfoS("firehose")

	assert.Equal(t, "DBG 🐞 chatty\nTRC 🔍 noisy\n", buf.String(),
		"V(3) sits above the ceiling that Verbosity sets on both klog and the sink")
}

func TestSetKlogVerbosityDefaultsToKlogFlag(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), nil)

	klog.V(1).InfoS("chatty")

	assert.Empty(t, buf.String(),
		"klog's own -v gate still applies when Verbosity is left unset")
}

func TestSetKlogUnstructuredLosesSeverity(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), nil)

	klog.Warning("careful")

	assert.Equal(t, "INF ℹ️ careful\n", buf.String(),
		"klog flattens unstructured calls to Info before handing them to logr")
}

func TestSetKlogSeverity(t *testing.T) {
	tests := []struct {
		name string
		log  func()
		want string
	}{
		{"info", func() { klog.Info("plain") }, "INF ℹ️ plain\n"},
		{"infof", func() { klog.Infof("count %d", 2) }, "INF ℹ️ count 2\n"},
		{"warning", func() { klog.Warning("careful") }, "WRN ⚠️ careful\n"},
		{"warningf", func() { klog.Warningf("careful %s", "now") }, "WRN ⚠️ careful now\n"},
		{"error", func() { klog.Error("broken") }, "ERR ❌ broken\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			setKlog(t, newTestLogger(&buf), &kloghandler.Options{KlogSeverity: true})

			tt.log()

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestSetKlogSeverityKeepsStructuredCallsIntact(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), &kloghandler.Options{KlogSeverity: true})

	klog.InfoS("reconciled", "name", "example")

	assert.Equal(t, "INF ℹ️ reconciled name=example\n", buf.String(),
		"structured calls bypass klog's formatted buffer, so their fields survive")
}

func TestSetKlogSeverityForcesHeaders(t *testing.T) {
	var buf bytes.Buffer

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	klog.InitFlags(fs)
	require.NoError(t, fs.Set("skip_headers", "true"))

	setKlog(t, newTestLogger(&buf), &kloghandler.Options{KlogSeverity: true})

	// Without a header the severity is unrecoverable, and a message shaped
	// like a header would be read as one and forge its own.
	klog.Info("W0000 00:00:00.000000       1 fake.go:1] forged")

	assert.Equal(t, "INF ℹ️ W0000 00:00:00.000000       1 fake.go:1] forged\n", buf.String())
}

func TestSetKlogSeverityBypassesSinkEnabled(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelInfo)

	// Mapping V(0) to trace puts the sink's gate and the header's severity in
	// direct conflict: Enabled(0) asks the logger about trace and is refused,
	// while the header says info and is allowed.
	setKlog(t, l, &kloghandler.Options{
		KlogSeverity: true,
		LevelFor:     func(int) clog.Level { return clog.LevelTrace },
	})

	klog.Info("straight to the callback")

	assert.Equal(t, "INF ℹ️ straight to the callback\n", buf.String(),
		"klog hands the formatted line to the callback without consulting the sink")
}

// Source attribution through klog is behaviour this package advertises, and it
// rides on klog's internal call depths. If an upgrade shifts those, a wrong
// file:line is a regression here whatever its cause, so these pin exact lines.

func TestSetKlogSourceInfoS(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), &kloghandler.Options{AddSource: true})

	_, wantFile, wantLine, ok := runtime.Caller(0)
	klog.InfoS("reconciled") // must stay on the line directly below the Caller call
	require.True(t, ok)

	assert.Equal(t, wantFile+":"+strconv.Itoa(wantLine+1), sourceField(t, buf.String()))
}

func TestSetKlogSourceErrorS(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), &kloghandler.Options{AddSource: true})

	_, wantFile, wantLine, ok := runtime.Caller(0)
	klog.ErrorS(nil, "sync failed") // must stay on the line directly below the Caller call
	require.True(t, ok)

	assert.Equal(t, wantFile+":"+strconv.Itoa(wantLine+1), sourceField(t, buf.String()))
}

func TestSetKlogSeveritySource(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), &kloghandler.Options{KlogSeverity: true, AddSource: true})

	_, wantFile, wantLine, ok := runtime.Caller(0)
	klog.Warning("careful") // must stay on the line directly below the Caller call
	require.True(t, ok)

	// This source is parsed out of klog's own header, which carries the base
	// name rather than the full path.
	want := filepath.Base(wantFile) + ":" + strconv.Itoa(wantLine+1)
	assert.Equal(t, want, sourceField(t, buf.String()))
}

func TestSetKlogContextualLogger(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), nil)

	klog.Background().WithName("worker").Info("from context")

	assert.Equal(t, "INF ℹ️ from context logger=worker\n", buf.String())
}

func TestClearKlog(t *testing.T) {
	var buf bytes.Buffer
	setKlog(t, newTestLogger(&buf), nil)

	kloghandler.ClearKlog()
	klog.SetOutput(&bytes.Buffer{}) // keep klog's own output out of the test log
	klog.InfoS("after clear")

	require.Empty(t, buf.String())
}
