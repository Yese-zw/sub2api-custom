package judge

import (
	"fmt"
	"math"
)

// Rules the v6 reference bank was built with (judge_v6.py): at least two
// reference artifacts per side, and the degraded mean must sit at least
// MinReferenceGap above the normal mean, otherwise the prompt gets no
// same-prompt reference.
const (
	MinReferencesPerSide = 2
	MinReferenceGap      = 1.5
)

// Reference is a same-prompt baseline: the mean V5b margin of known-normal
// artifacts (Low) and of known-degraded artifacts (High) for one prompt.
type Reference struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

// NewReference builds a Reference from margins (Judger.Margin) of artifacts
// generated for one prompt by a known-normal and a known-degraded source,
// using the same rules as the built-in bank.
func NewReference(normalMargins, degradedMargins []float64) (Reference, error) {
	if len(normalMargins) < MinReferencesPerSide || len(degradedMargins) < MinReferencesPerSide {
		return Reference{}, fmt.Errorf("%w: need at least %d margins per side", ErrBadReference, MinReferencesPerSide)
	}
	ref := Reference{Low: mean(normalMargins), High: mean(degradedMargins)}
	if err := ref.validate(); err != nil {
		return Reference{}, err
	}
	return ref, nil
}

func (r Reference) validate() error {
	if math.IsNaN(r.Low) || math.IsNaN(r.High) || math.IsInf(r.Low, 0) || math.IsInf(r.High, 0) {
		return fmt.Errorf("%w: margins must be finite", ErrBadReference)
	}
	if r.High-r.Low < MinReferenceGap {
		return fmt.Errorf("%w: degraded mean %.3f is less than %.1f above normal mean %.3f",
			ErrBadReference, r.High, MinReferenceGap, r.Low)
	}
	return nil
}

func mean(values []float64) float64 {
	var sum neumaierSum
	for _, v := range values {
		sum.add(v)
	}
	return sum.value() / float64(len(values))
}

// WithReference adds a same-prompt reference for a prompt outside the
// built-in bank. Such prompts are judged with MethodCustom using the v6
// threshold and borderline band. The built-in bank cannot be overridden.
//
// The v6 threshold was calibrated for the built-in normal/degraded sources;
// a custom reference is only meaningful if its two sides show the same kind
// of gap. See docs/assets.md.
func WithReference(prompt string, ref Reference) Option {
	return func(c *loadConfig) error {
		if err := ref.validate(); err != nil {
			return err
		}
		if c.custom == nil {
			c.custom = map[string]Reference{}
		}
		c.custom[PromptKey(prompt)] = ref
		return nil
	}
}
