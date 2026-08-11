package clog

import (
	"io"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
)

func TestFieldShapesTokens(t *testing.T) {
	tests := []struct {
		name   string
		shapes FieldShapeMap
		fields []Field
		want   string
	}{
		{
			name:   "no_shape_renders_key_and_value",
			shapes: FieldShapeMap{"other": {OmitKey: true}},
			fields: []Field{{Key: "region", Value: "emea"}},
			want:   " region=emea",
		},
		{
			name:   "omit_key_drops_key_and_separator",
			shapes: FieldShapeMap{"region": {OmitKey: true}},
			fields: []Field{{Key: "region", Value: "emea"}},
			want:   " emea",
		},
		{
			name:   "affixes_keep_the_key",
			shapes: FieldShapeMap{"region": {Prefix: "(", Suffix: ")"}},
			fields: []Field{{Key: "region", Value: "emea"}},
			want:   " region=(emea)",
		},
		{
			name:   "omit_key_with_affixes_renders_a_badge",
			shapes: FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}},
			fields: []Field{{Key: "region", Value: "emea"}},
			want:   " (emea)",
		},
		{
			name:   "affixes_wrap_outside_quotes",
			shapes: FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}},
			fields: []Field{{Key: "region", Value: "emea north"}},
			want:   ` ("emea north")`,
		},
		{
			name:   "affixes_wrap_outside_slice_brackets",
			shapes: FieldShapeMap{"zones": {OmitKey: true, Prefix: "⟨", Suffix: "⟩"}},
			fields: []Field{{Key: "zones", Value: []string{"a", "b"}}},
			want:   " ⟨[a, b]⟩",
		},
		{
			name:   "empty_value_renders_affixes_alone",
			shapes: FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}},
			fields: []Field{{Key: "region", Value: ""}},
			want:   " ()",
		},
		{
			name:   "prefix_only",
			shapes: FieldShapeMap{"branch": {OmitKey: true, Prefix: "@"}},
			fields: []Field{{Key: "branch", Value: "topic"}},
			want:   " @topic",
		},
		{
			name: "shapes_apply_per_key",
			shapes: FieldShapeMap{
				"region": {OmitKey: true, Prefix: "(", Suffix: ")"},
			},
			fields: []Field{
				{Key: "region", Value: "emea"},
				{Key: "queue", Value: "ingest"},
			},
			want: " (emea) queue=ingest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := formatFieldsOpts{fieldShapes: tt.shapes, noColor: true}
			assert.Equal(t, tt.want, formatFields(tt.fields, opts))
		})
	}
}

func TestFieldShapesSortByRealKey(t *testing.T) {
	// A hidden key still decides where its field sorts.
	opts := formatFieldsOpts{
		fieldShapes: FieldShapeMap{"beta": {OmitKey: true, Prefix: "(", Suffix: ")"}},
		fieldSort:   SortAscending,
		noColor:     true,
	}
	fields := []Field{
		{Key: "gamma", Value: "c"},
		{Key: "beta", Value: "b"},
		{Key: "alpha", Value: "a"},
	}

	assert.Equal(t, " alpha=a (b) gamma=c", formatFields(fields, opts))
}

func TestFieldShapesAffixesTakeKeyValueStyle(t *testing.T) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles := DefaultStyles()
	styles.KeyValues = style.KeyValueMap{
		"region": {Values: style.ValueMap{"emea": &green}},
	}

	opts := formatFieldsOpts{
		fieldShapes: FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}},
		level:       LevelInfo,
		styles:      styles,
	}
	got := formatFields([]Field{{Key: "region", Value: "emea"}}, opts)

	// The whole token reads as one unit: affixes share the value's color.
	want := " " + green.Render("(") + green.Render("emea") + green.Render(")")
	assert.Equal(t, want, got)
}

func TestFieldShapesAffixesTakeTypeStyle(t *testing.T) {
	styles := DefaultStyles()

	opts := formatFieldsOpts{
		fieldShapes: FieldShapeMap{"count": {OmitKey: true, Prefix: "(", Suffix: ")"}},
		level:       LevelInfo,
		styles:      styles,
	}
	got := formatFields([]Field{{Key: "count", Value: 42}}, opts)

	num := styles.FieldNumber
	want := " " + num.Render("(") + num.Render("42") + num.Render(")")
	assert.Equal(t, want, got)
}

func TestFieldShapesAffixesOutsideStyledQuotes(t *testing.T) {
	styles := DefaultStyles()
	quote := lipgloss.NewStyle().Bold(true)
	styles.FieldQuote = &style.QuoteStyle{Style: quote}
	styles.FieldString = nil

	opts := formatFieldsOpts{
		fieldShapes: FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}},
		level:       LevelInfo,
		styles:      styles,
	}
	got := formatFields([]Field{{Key: "region", Value: "emea north"}}, opts)

	want := " (" + quote.Render(`"`) + "emea north" + quote.Render(`"`) + ")"
	assert.Equal(t, want, got)
}

func TestFieldShapesUnstyledValueKeepsPlainAffixes(t *testing.T) {
	// Durations style per segment, so their affixes render plain.
	styles := DefaultStyles()
	styles.FieldDuration = style.SegmentStyle{}

	opts := formatFieldsOpts{
		fieldShapes: FieldShapeMap{"took": {OmitKey: true, Prefix: "(", Suffix: ")"}},
		level:       LevelInfo,
		styles:      styles,
	}
	got := formatFields([]Field{{Key: "took", Value: 1500 * time.Millisecond}}, opts)

	assert.Equal(t, " (1.5s)", got)
}

func TestFieldShapesNoColorRendersPlainAffixes(t *testing.T) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles := DefaultStyles()
	styles.KeyValues = style.KeyValueMap{
		"region": {Values: style.ValueMap{"emea": &green}},
	}

	opts := formatFieldsOpts{
		fieldShapes: FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}},
		level:       LevelInfo,
		noColor:     true,
		styles:      styles,
	}

	assert.Equal(t, " (emea)", formatFields([]Field{{Key: "region", Value: "emea"}}, opts))
}

func TestFieldShapesBelowStyleLevelRendersPlainAffixes(t *testing.T) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles := DefaultStyles()
	styles.KeyValues = style.KeyValueMap{
		"region": {Values: style.ValueMap{"emea": &green}},
	}

	opts := formatFieldsOpts{
		fieldShapes:     FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}},
		fieldStyleLevel: LevelWarn,
		level:           LevelInfo,
		styles:          styles,
	}

	assert.Equal(t, " (emea)", formatFields([]Field{{Key: "region", Value: "emea"}}, opts))
}

func TestSetFieldShapes(t *testing.T) {
	l, buf := newTestLogger()
	l.SetFieldShapes(FieldShapeMap{"region": {OmitKey: true, Prefix: "(", Suffix: ")"}})

	l.Info().Str("region", "emea").Str("queue", "ingest").Msg("draining")

	assert.Equal(t, "INF ℹ️ draining (emea) queue=ingest\n", buf.String())
}

func TestSetFieldShapesReplacesAndClears(t *testing.T) {
	l, buf := newTestLogger()

	l.SetFieldShapes(FieldShapeMap{"region": {OmitKey: true}})
	l.SetFieldShapes(FieldShapeMap{"queue": {OmitKey: true}})
	l.Info().Str("region", "emea").Str("queue", "ingest").Msg("msg")
	assert.Equal(t, "INF ℹ️ msg region=emea ingest\n", buf.String())

	buf.Reset()
	l.SetFieldShapes(nil)
	l.Info().Str("region", "emea").Str("queue", "ingest").Msg("msg")
	assert.Equal(t, "INF ℹ️ msg region=emea queue=ingest\n", buf.String())
}

func TestSetFieldShapesCopiesTheMap(t *testing.T) {
	l, buf := newTestLogger()

	shapes := FieldShapeMap{"region": {OmitKey: true}}
	l.SetFieldShapes(shapes)
	shapes["region"] = FieldShape{Prefix: "!"}

	l.Info().Str("region", "emea").Msg("msg")
	assert.Equal(t, "INF ℹ️ msg emea\n", buf.String())
}

func TestPackageLevelSetFieldShapes(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	SetDefault(NewWriter(io.Discard))
	SetFieldShapes(FieldShapeMap{"region": {OmitKey: true}})

	Default().mu.Lock()
	got := Default().fieldShapes
	Default().mu.Unlock()

	assert.Equal(t, FieldShapeMap{"region": {OmitKey: true}}, got)
}
