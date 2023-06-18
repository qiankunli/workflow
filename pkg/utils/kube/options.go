package kube

// Option is some configuration that modifies options for a patch request.
type Option interface {
	// ApplyToHelper applies this configuration to the given Helper options.
	ApplyToHelper(*HelperOptions)
}

// HelperOptions contains options for patch options.
type HelperOptions struct {
	// IncludeStatusObservedGeneration sets the status.observedGeneration field
	// on the incoming object to match metadata.generation, only if there is a change.
	IncludeStatusObservedGeneration bool

	// OwnedConditions defines condition types owned by the controller.
	OwnedConditions []string
}

// WithForceOverwriteConditions allows the patch helper to overwrite conditions in case of conflicts.
// This option should only ever be set in controller managing the object being patched.
type WithForceOverwriteConditions struct{}

// WithStatusObservedGeneration sets the status.observedGeneration field
// on the incoming object to match metadata.generation, only if there is a change.
type WithStatusObservedGeneration struct{}

// ApplyToHelper applies this configuration to the given HelperOptions.
func (w WithStatusObservedGeneration) ApplyToHelper(in *HelperOptions) {
	in.IncludeStatusObservedGeneration = true
}

// WithOwnedConditions allows to define condition types owned by the controller.
// In case of conflicts for the owned conditions, the patch helper will always use the value provided by the controller.
type WithOwnedConditions struct {
	Conditions []string
}

// ApplyToHelper applies this configuration to the given HelperOptions.
func (w WithOwnedConditions) ApplyToHelper(in *HelperOptions) {
	in.OwnedConditions = w.Conditions
}
