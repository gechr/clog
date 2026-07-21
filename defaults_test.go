package clog

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func TestLoggerDefaultWrapperSync(t *testing.T) {
	loggerMethods := methodSet(reflect.TypeFor[*Logger]())
	wrappers := defaultWrapperFuncs()

	for name, loggerMethod := range loggerMethods {
		if !isDefaultWrapperMethod(name) {
			continue
		}
		wrapper, ok := wrappers[name]
		if !ok {
			t.Errorf("Logger has method %s but defaults.go has no package-level wrapper", name)
			continue
		}
		compareDefaultWrapperSignature(t, name, loggerMethod.Type, reflect.TypeOf(wrapper))
	}

	for name, wrapper := range wrappers {
		loggerMethod, ok := loggerMethods[name]
		if !ok {
			t.Errorf("defaults.go wrapper %s has no Logger method", name)
			continue
		}
		compareDefaultWrapperSignature(t, name, loggerMethod.Type, reflect.TypeOf(wrapper))
	}

	checkDefaultWrapperInventory(t, wrappers)
}

func defaultWrapperFuncs() map[string]any {
	return map[string]any{
		"AddHook":                         AddHook,
		"ClearAllHooks":                   ClearAllHooks,
		"ClearHooks":                      ClearHooks,
		"SetAnimationInterval":            SetAnimationInterval,
		"SetColorMode":                    SetColorMode,
		"SetDurationFormat":               SetDurationFormat,
		"SetDurationGradientMax":          SetDurationGradientMax,
		"SetDurationMinimum":              SetDurationMinimum,
		"SetDurationScale":                SetDurationScale,
		"SetElapsedFormat":                SetElapsedFormat,
		"SetElapsedGradientMax":           SetElapsedGradientMax,
		"SetElapsedMinimum":               SetElapsedMinimum,
		"SetElapsedScale":                 SetElapsedScale,
		"SetExitCode":                     SetExitCode,
		"SetExitFunc":                     SetExitFunc,
		"SetFieldFormats":                 SetFieldFormats,
		"SetHyperlinkColumnFormat":        SetHyperlinkColumnFormat,
		"SetHyperlinkDirFormat":           SetHyperlinkDirFormat,
		"SetHyperlinkEnabled":             SetHyperlinkEnabled,
		"SetHyperlinkFileFormat":          SetHyperlinkFileFormat,
		"SetHyperlinkLineFormat":          SetHyperlinkLineFormat,
		"SetHyperlinkPathFormat":          SetHyperlinkPathFormat,
		"SetPercentFormat":                SetPercentFormat,
		"SetPercentMaximum":               SetPercentMaximum,
		"SetPercentPrecision":             SetPercentPrecision,
		"SetPercentReverseGradient":       SetPercentReverseGradient,
		"SetQuantityUnitsIgnoreCase":      SetQuantityUnitsIgnoreCase,
		"SetTimeGradientMax":              SetTimeGradientMax,
		"SetTimeScale":                    SetTimeScale,
		"SetFieldSort":                    SetFieldSort,
		"SetFieldStyleLevel":              SetFieldStyleLevel,
		"SetFieldTimeFormat":              SetFieldTimeFormat,
		"SetHandler":                      SetHandler,
		"SetIndent":                       SetIndent,
		"SetIndentPrefixes":               SetIndentPrefixes,
		"SetIndentPrefixSeparator":        SetIndentPrefixSeparator,
		"SetIndentWidth":                  SetIndentWidth,
		"SetJSONIndent":                   SetJSONIndent,
		"SetJSONPrintMode":                SetJSONPrintMode,
		"SetLabelWidth":                   SetLabelWidth,
		"SetLabels":                       SetLabels,
		"SetLevel":                        SetLevel,
		"SetLevelAlign":                   SetLevelAlign,
		"SetNonTTYLevel":                  SetNonTTYLevel,
		"SetNumberFormat":                 SetNumberFormat,
		"SetFractionFormat":               SetFractionFormat,
		"SetNumberGroupSeparator":         SetNumberGroupSeparator,
		"SetNumberCompactMinimum":         SetNumberCompactMinimum,
		"SetNumberCompactFallback":        SetNumberCompactFallback,
		"SetOmitEmpty":                    SetOmitEmpty,
		"SetOmitZero":                     SetOmitZero,
		"SetInput":                        SetInput,
		"SetOutput":                       SetOutput,
		"SetOutputWriter":                 SetOutputWriter,
		"SetParts":                        SetParts,
		"SetPrintIndent":                  SetPrintIndent,
		"SetPromptMarker":                 SetPromptMarker,
		"SetQuote":                        SetQuote,
		"SetQuoteChars":                   SetQuoteChars,
		"SetReportTimestamp":              SetReportTimestamp,
		"SetSeparatorText":                SetSeparatorText,
		"SetSliceBrackets":                SetSliceBrackets,
		"SetSliceSeparator":               SetSliceSeparator,
		"SetSmartQuoteChars":              SetSmartQuoteChars,
		"SetSmartQuotes":                  SetSmartQuotes,
		"SetSpinnerDefaults":              SetSpinnerDefaults,
		"SetStyles":                       SetStyles,
		"SetSuppressEchoDuringAnimations": SetSuppressEchoDuringAnimations,
		"SetSymbols":                      SetSymbols,
		"SetTheme":                        SetTheme,
		"SetTimeFormat":                   SetTimeFormat,
		"SetTimeLocation":                 SetTimeLocation,
		"SetTreeChars":                    SetTreeChars,
		"SetWrap":                         SetWrap,
		"SetYAMLIndent":                   SetYAMLIndent,
		"SetYAMLIndentSequence":           SetYAMLIndentSequence,
	}
}

var defaultWrapperFuncExclusions = map[string]string{
	"Configure":        "startup helper that applies Config, not a Logger method",
	"DefaultLabels":    "default-value helper, not a default-logger wrapper",
	"DefaultParts":     "default-value helper, not a default-logger wrapper",
	"DefaultStyles":    "default-value helper, not a default-logger wrapper",
	"DefaultSymbols":   "default-value helper, not a default-logger wrapper",
	"DefaultTreeChars": "default-value helper, not a default-logger wrapper",
	"SetVerbose":       "composite helper that changes level and timestamp together",
}

func isDefaultWrapperMethod(name string) bool {
	return strings.HasPrefix(name, "Set") ||
		name == "AddHook" ||
		name == "ClearAllHooks" ||
		name == "ClearHooks"
}

func compareDefaultWrapperSignature(
	t *testing.T,
	name string,
	methodType, wrapperType reflect.Type,
) {
	t.Helper()

	if wrapperType.Kind() != reflect.Func {
		t.Fatalf("wrapper %s has kind %s, want func", name, wrapperType.Kind())
	}
	if methodType.IsVariadic() != wrapperType.IsVariadic() {
		t.Errorf("wrapper %s variadic mismatch: Logger=%t wrapper=%t",
			name, methodType.IsVariadic(), wrapperType.IsVariadic())
	}

	methodParams := methodType.NumIn() - 1 // drop receiver
	if methodParams != wrapperType.NumIn() {
		t.Errorf("wrapper %s parameter count mismatch: Logger=%d wrapper=%d",
			name, methodParams, wrapperType.NumIn())
		return
	}
	for i := range methodParams {
		methodParam := methodType.In(i + 1)
		wrapperParam := wrapperType.In(i)
		if methodParam != wrapperParam {
			t.Errorf("wrapper %s parameter %d mismatch: Logger=%v wrapper=%v",
				name, i, methodParam, wrapperParam)
		}
	}

	if methodType.NumOut() != wrapperType.NumOut() {
		t.Errorf("wrapper %s return count mismatch: Logger=%d wrapper=%d",
			name, methodType.NumOut(), wrapperType.NumOut())
		return
	}
	for i := range methodType.NumOut() {
		methodOut := methodType.Out(i)
		wrapperOut := wrapperType.Out(i)
		if methodOut != wrapperOut {
			t.Errorf("wrapper %s return %d mismatch: Logger=%v wrapper=%v",
				name, i, methodOut, wrapperOut)
		}
	}
}

func checkDefaultWrapperInventory(t *testing.T, wrappers map[string]any) {
	t.Helper()

	funcs := defaultFuncs(t)
	for name := range wrappers {
		if !funcs[name] {
			t.Errorf("default wrapper table contains %s, but defaults.go does not declare it", name)
		}
	}
	for name := range funcs {
		if _, ok := wrappers[name]; ok {
			continue
		}
		if _, ok := defaultWrapperFuncExclusions[name]; ok {
			continue
		}
		t.Errorf(
			"defaults.go declares exported function %s; add it to defaultWrapperFuncs or defaultWrapperFuncExclusions",
			name,
		)
	}
}

func defaultFuncs(t *testing.T) map[string]bool {
	t.Helper()

	funcs := make(map[string]bool)
	for _, name := range []string{"defaults.go", "fieldformats_setters_default.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			funcs[fn.Name.Name] = true
		}
	}
	return funcs
}
