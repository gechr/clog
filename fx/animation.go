package fx

// Animation is the animation rendering mode for a [Builder].
// It controls how the message text is animated. Symbol animation is
// controlled independently by [Builder.AnimatedSymbol].
type Animation int

const (
	// AnimationNone applies no message effect. This is the default.
	AnimationNone Animation = iota
	// AnimationBar renders a progress bar.
	AnimationBar
	// AnimationPulse renders a pulsing text effect.
	AnimationPulse
	// AnimationShimmer renders a shimmering text effect.
	AnimationShimmer
)
