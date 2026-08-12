package clog

import (
	"bytes"
	"io"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/fx/spinner"
	"github.com/stretchr/testify/assert"
)

func TestApplyPresetTerseOutput(t *testing.T) {
	l, buf := newTestLogger()

	l.ApplyPreset(TersePreset())

	l.Info().Msg("Starting")
	l.Success().Msg("Deployed")
	l.Notice().Msg("Skipping cache")
	l.Warn().Msg("Degraded upstream")
	l.Error().Msg("Connection refused")
	l.Dry().Msg("rm -rf ./cache")
	l.Info().Str("service", "billing").Msg("Mounted")

	assert.Equal(t,
		"· Starting\n"+
			"✔︎ Deployed\n"+
			"› Skipping cache\n"+
			"! Degraded upstream\n"+
			"✘ Connection refused\n"+
			"$ rm -rf ./cache\n"+
			"· Mounted service=billing\n",
		buf.String(),
	)
}

func TestApplyPresetNilLogger(t *testing.T) {
	var l *Logger

	assert.NotPanics(t, func() {
		l.ApplyPreset(TersePreset())
	})
}

func TestApplyPresetNilPreset(t *testing.T) {
	l, buf := newTestLogger()

	l.ApplyPreset(nil)

	l.Info().Msg("msg")
	assert.Equal(t, "INF ℹ️ msg\n", buf.String())
}

func TestApplyPresetIdempotent(t *testing.T) {
	l, buf := newTestLogger()

	l.ApplyPreset(TersePreset())
	l.ApplyPreset(TersePreset())

	l.Info().Msg("Starting")
	l.Success().Msg("Deployed")

	assert.Equal(t, "· Starting\n✔︎ Deployed\n", buf.String())
}

func TestApplyPresetSparse(t *testing.T) {
	l, buf := newTestLogger()

	l.ApplyPreset(&Preset{Wrap: new(WrapSoft)})

	l.Info().Msg("msg")
	assert.Equal(t, "INF ℹ️ msg\n", buf.String(),
		"a sparse preset should leave unrelated settings unchanged")
}

func TestApplyPresetMergesSymbols(t *testing.T) {
	l, buf := newTestLogger()
	l.SetLevel(LevelDebug)

	l.ApplyPreset(TersePreset())

	l.Debug().Msg("msg")
	assert.Equal(t, "🐞 msg\n", buf.String(),
		"levels absent from the preset should keep their default symbols")
}

func TestApplyPresetSpinnerDefaults(t *testing.T) {
	l, _ := newTestLogger()

	l.ApplyPreset(TersePreset())

	assert.Equal(t, spinner.Apply(spinner.WithConfig(spinner.DotsBounce)), l.resolveSpinnerConfig())
}

func TestApplyPresetFatalMessageStyleIsPlain(t *testing.T) {
	l, _ := newTestLogger()

	l.ApplyPreset(TersePreset())

	l.mu.Lock()
	got := l.styles.Messages[LevelFatal]
	l.mu.Unlock()

	assert.Equal(t, new(lipgloss.NewStyle()), got,
		"the terse preset should override the default fatal message style with a plain style")
}

func TestPackageLevelApplyPreset(t *testing.T) {
	origDefault := Default()
	defer SetDefault(origDefault)

	SetDefault(NewWriter(io.Discard))
	ApplyPreset(&Preset{Parts: []Part{PartMessage}})

	Default().mu.Lock()
	got := Default().parts
	Default().mu.Unlock()

	assert.Equal(t, []Part{PartMessage}, got)
}

func TestPackageLevelApplyPresetTerseOutput(t *testing.T) {
	origDefault := Default()
	defer SetDefault(origDefault)

	var buf bytes.Buffer
	SetDefault(New(TestOutput(&buf)))
	ApplyPreset(TersePreset())

	Success().Msg("Deployed")

	assert.Equal(t, "✔︎ Deployed\n", buf.String())
}
