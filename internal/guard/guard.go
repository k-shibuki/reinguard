package guard

// Guard is a built-in guard implementation registered by ID (ADR-0011). Declarative rules
// under control/guards select when a built-in applies (same priority model as states; ADR-0004).
type Guard interface {
	ID() string
	Eval(signals map[string]any) MergeReadinessResult
}
