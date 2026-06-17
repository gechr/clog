package fx

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestManualReturnsDrivableUpdate verifies Manual hands back an Update wired to
// the task's atomic state and marks the task started, without running anything.
func TestManualReturnsDrivableUpdate(t *testing.T) {
	g := NewGroup(context.Background(), newStubLogger())
	entry := g.Add(testBar(newStubLogger(), "work", 2))

	update, finish := entry.Manual()
	require.True(t, entry.task.started(), "Manual should mark the task started")
	require.Zero(t, entry.task.finishedAt.Load(), "task is not finished until finish is called")

	update.Msg("halfway").SetProgress(1).Send()
	require.Equal(t, "halfway", *entry.task.msgPtr.Load())
	require.Equal(t, 1, update.Progress())

	finish(nil)
	require.NotZero(t, entry.task.finishedAt.Load(), "finish should mark the task finished")

	select {
	case err := <-entry.task.doneErr:
		require.NoError(t, err)
	default:
		t.Fatal("finish should signal completion on doneErr")
	}
}

// TestManualPropagatesError verifies the error passed to finish reaches doneErr,
// which is what Group.Wait reports for the task.
func TestManualPropagatesError(t *testing.T) {
	g := NewGroup(context.Background(), newStubLogger())
	entry := g.Add(testBar(newStubLogger(), "work", 1))

	_, finish := entry.Manual()
	wantErr := errors.New("boom")
	finish(wantErr)

	require.ErrorIs(t, <-entry.task.doneErr, wantErr)
}

// TestManualFinishIsIdempotent verifies a second finish is a no-op, so a caller
// using `defer finish(err)` after an explicit finish cannot double-send and
// wedge the render loop.
func TestManualFinishIsIdempotent(t *testing.T) {
	g := NewGroup(context.Background(), newStubLogger())
	entry := g.Add(testBar(newStubLogger(), "work", 1))

	_, finish := entry.Manual()
	finish(nil)
	<-entry.task.doneErr // drain the single buffered result

	finish(errors.New("second")) // must not send again
	select {
	case <-entry.task.doneErr:
		t.Fatal("finish sent twice")
	default:
	}
}
