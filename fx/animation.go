package fx

// Animation is the animation rendering mode for a [Builder].
type Animation int

const (
	// AnimationBar renders a progress bar.
	AnimationBar Animation = iota
	// AnimationPulse renders a pulsing text effect.
	AnimationPulse
	// AnimationShimmer renders a shimmering text effect.
	AnimationShimmer
	// AnimationSpinner renders a spinning character.
	AnimationSpinner
)
