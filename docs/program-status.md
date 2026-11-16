# Program status

`SetProgramStatus` reports what a command is doing through the [Program Status Protocol (OSC 7501)](https://www.superlogical.com/rex/docs/build/program-status). Supporting terminals can display progress, completion, failures, and requests for user action outside the terminal's text area.

```go
clog.SetProgramStatus(clog.ProgramStatus{
    State:       clog.ProgramStateWorking,
    App:         "example",
    Message:     "Processing items",
    Progress:    40,
    HasProgress: true,
})

clog.SetProgramStatus(clog.ProgramStatus{
    State:   clog.ProgramStateBlocked,
    Kind:    clog.ProgramStatusKindPermission,
    App:     "example",
    Message: "Waiting for approval",
})

clog.SetProgramStatus(clog.ProgramStatus{
    State:   clog.ProgramStateDone,
    App:     "example",
    Message: "Processing complete",
})
```

Use `ProgramStateError` when work fails and `ProgramStateIdle` when it is cancelled. The application owns these transitions; completing a spinner or logging an error does not change program status automatically. Omit `HasProgress` for indeterminate work. Each report replaces its record completely, so repeat `App`, `Title`, and any other fields that should remain present.

An empty `ID` addresses the root record. Use IDs such as `batch/worker` to report separate activities. `ProgramStateClear` removes the addressed record and its descendants; an empty ID clears every record on the terminal.

Reports use the logger's output and synchronize with live animations without inserting newlines or moving the cursor. They bypass log levels and handlers. Redirected output receives no reports, including with `ColorAlways`; `ColorNever` only disables colors and does not suppress status. Unsupported terminals ignore the sequence. The encoder sanitizes text and enforces protocol limits, but callers must keep credentials and other sensitive data out of reports.

The same API is available on a logger (`logger.SetProgramStatus`) or an output (`output.SetProgramStatus`). The status types and constants alias `github.com/charmbracelet/x/ansi`.
