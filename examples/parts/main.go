package main

import (
	"strconv"

	"github.com/gechr/clog"
	"github.com/gechr/clog/style"
)

// Custom parts take any value outside the built-in range (0-4).
const (
	PartWorker clog.Part = 100
	PartAlert  clog.Part = 101
)

func main() {
	worker := 3

	// A renderer is handed the completed entry, the logger's styles, and
	// whether color is off.
	clog.RegisterPart(PartWorker, func(_ clog.Entry, styles *style.Config, noColor bool) string {
		label := "worker-" + strconv.Itoa(worker)
		if noColor || styles.KeyDefault == nil {
			return label
		}
		return styles.KeyDefault.Render(label)
	})

	// Returning "" omits the part for that line, so a part can appear only
	// when it has something to say.
	clog.RegisterPart(PartAlert, func(e clog.Entry, _ *style.Config, _ bool) string {
		if e.Level < clog.LevelWarn {
			return ""
		}
		return "‼️"
	})

	clog.SetParts(clog.PartLevel, PartWorker, PartAlert, clog.PartMessage, clog.PartFields)

	clog.Info().Str("queue", "ingest").Msg("Draining queue")
	clog.Warn().Int("depth", 4_200).Msg("Queue is backing up")

	// Custom parts reorder and hide like built-in ones.
	clog.SetParts(clog.PartMessage, PartWorker)
	clog.Info().Msg("Worker last")

	// A single event can override the order without touching the logger.
	clog.SetParts(clog.DefaultParts()...)
	clog.Info().Parts(PartWorker, clog.PartMessage).Msg("Tagged line")
	clog.Info().Msg("Back to the logger's parts")

	// Unregistering leaves the part in the order but renders nothing.
	clog.UnregisterPart(PartWorker)
	clog.SetParts(clog.PartLevel, PartWorker, clog.PartMessage)
	clog.Info().Msg("Worker part is gone")
}
