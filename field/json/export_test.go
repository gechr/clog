package json //nolint:revive // export_test must match the package under test

// Test-only exports for external test packages.
var (
	ResolveStringToken = resolveStringToken
	RenderFlat         = renderFlat
	CollectFlatPairs   = collectFlatPairs
	ScanValueEnd       = scanValueEnd
	IsSpace            = isSpace
	EmitStyled         = emitStyled
	ResolveNumberStyle = resolveNumberStyle
	HjsonUnquoteKey    = hjsonUnquoteKey
	HjsonUnquoteValue  = hjsonUnquoteValue
)

// FlatPair is a test-only alias for the unexported flatPair type.
type FlatPair = flatPair
