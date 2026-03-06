package duration_test

import (
	"testing"
	"time"

	"github.com/gechr/clog/field/duration"
	"github.com/stretchr/testify/assert"
)

func TestSetFormatFunc(t *testing.T) {
	snap := duration.Save()
	t.Cleanup(func() { duration.Restore(snap) })

	custom := func(_ time.Duration) string { return "custom" }
	duration.SetFormatFunc(custom)
	got := duration.FormatFunc()
	assert.NotNil(t, got, "expected non-nil FormatFunc after setting custom function")
	assert.Equal(t, "custom", got(5*time.Second), "expected custom format function to be used")
}

func TestSetFormatFuncNil(t *testing.T) {
	snap := duration.Save()
	t.Cleanup(func() { duration.Restore(snap) })

	duration.SetFormatFunc(func(_ time.Duration) string { return "x" })
	duration.SetFormatFunc(nil)
	assert.Nil(t, duration.FormatFunc(), "expected nil FormatFunc after setting nil")
}

func TestFormatFuncDefaultNil(t *testing.T) {
	snap := duration.Save()
	t.Cleanup(func() { duration.Restore(snap) })

	duration.SetFormatFunc(nil)
	assert.Nil(t, duration.FormatFunc(), "expected nil FormatFunc when no custom function is set")
}

func TestSaveRestore(t *testing.T) {
	snap := duration.Save()
	t.Cleanup(func() { duration.Restore(snap) })

	duration.SetFormatFunc(func(_ time.Duration) string { return "before" })
	saved := duration.Save()

	duration.SetFormatFunc(func(_ time.Duration) string { return "after" })
	assert.Equal(t, "after", duration.FormatFunc()(time.Second))

	duration.Restore(saved)
	assert.Equal(t, "before", duration.FormatFunc()(time.Second))
}
