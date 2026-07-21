package clog

import (
	"reflect"
	"testing"
)

// TestEventFieldBuilderMethodSync ensures that *Event and FieldBuilder[Context]
// (embedded in Context) expose the same set of field-appending methods.
// This catches drift when a new field type is added to one but not the other.
func TestEventFieldBuilderMethodSync(t *testing.T) {
	// Event-only methods that are NOT expected on FieldBuilder.
	// These are either finalizers, event lifecycle methods, or methods
	// that require logger/output state not available in FieldBuilder.
	eventOnly := map[string]bool{
		"Deadline":     true, // event lifecycle
		"Discard":      true, // event lifecycle
		"Disabled":     true, // event lifecycle
		"Elapsed":      true, // event lifecycle
		"Enabled":      true, // event lifecycle
		"Err":          true, // event lifecycle (different signature)
		"ExitCode":     true, // event lifecycle (Fatal exit code)
		"Func":         true, // takes func(*Event)
		"MessageStyle": true, // event-only override
		"Msg":          true, // finalizer
		"Msgf":         true, // finalizer
		"MsgFunc":      true, // finalizer
		"OmitEmpty":    true, // event-only override
		"OmitZero":     true, // event-only override
		"Parts":        true, // event-only override
		"Send":         true, // finalizer
		"Sort":         true, // event-only override
		"Symbol":       true, // event-only override
	}

	// FieldBuilder-only methods that are NOT expected on Event.
	// These are Context-specific methods for tree/indent/logger management,
	// or internal constructor helpers.
	fbOnly := map[string]bool{
		"Dedent":   true, // Context tree management
		"Depth":    true, // Context tree management
		"Indent":   true, // Context tree management
		"InitSelf": true, // constructor helper
		"Logger":   true, // Context logger accessor
		"Tree":     true, // Context tree management
	}

	eventType := reflect.TypeFor[*Event]()
	ctxType := reflect.TypeFor[*Context]()

	eventMethods := methodSet(eventType)
	fbMethods := methodSet(ctxType)

	// Check that every Event field method also exists on FieldBuilder (via Context).
	for name, eventMethod := range eventMethods {
		if eventOnly[name] {
			continue
		}
		fbMethod, ok := fbMethods[name]
		if !ok {
			t.Errorf("Event has method %s but Context (FieldBuilder) does not", name)
			continue
		}
		// Compare parameter counts (NumIn includes the receiver).
		if eventMethod.Type.NumIn() != fbMethod.Type.NumIn() {
			t.Errorf("method %s: Event has %d params, Context has %d params",
				name, eventMethod.Type.NumIn(), fbMethod.Type.NumIn())
		}
	}

	// Check that every FieldBuilder field method also exists on Event.
	for name, fbMethod := range fbMethods {
		if fbOnly[name] {
			continue
		}
		eventMethod, ok := eventMethods[name]
		if !ok {
			t.Errorf("Context (FieldBuilder) has method %s but Event does not", name)
			continue
		}
		if fbMethod.Type.NumIn() != eventMethod.Type.NumIn() {
			t.Errorf("method %s: Context has %d params, Event has %d params",
				name, fbMethod.Type.NumIn(), eventMethod.Type.NumIn())
		}
	}
}

// methodSet returns a map of exported method names to their types for a pointer type.
func methodSet(t reflect.Type) map[string]reflect.Method {
	m := make(map[string]reflect.Method, t.NumMethod())
	for method := range t.Methods() {
		if method.IsExported() {
			m[method.Name] = method
		}
	}
	return m
}
