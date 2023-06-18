package kube

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Helper is a utility for ensuring the proper patching of objects.
type Helper struct {
	client       client.Client
	beforeObject client.Object
	before       *unstructured.Unstructured
	after        *unstructured.Unstructured
	changes      map[string]bool
}

// NewHelper returns an initialized Helper
func NewHelper(obj client.Object, crClient client.Client) (*Helper, error) {
	if obj == nil || (reflect.ValueOf(obj).IsValid() && reflect.ValueOf(obj).IsNil()) {
		return nil, errors.Errorf("expected non-nil object")
	}

	// Convert the object to unstructured to compare against our before copy.
	unstructuredObj, err := ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &Helper{
		client:       crClient,
		before:       unstructuredObj,
		beforeObject: obj.DeepCopyObject().(client.Object),
	}, nil
}

// Patch will attempt to patch the given object, including its status.
func (h *Helper) Patch(ctx context.Context, obj client.Object, opts ...Option) error {
	if obj == nil {
		return errors.Errorf("expected non-nil object")
	}

	// Calculate the options.
	options := &HelperOptions{}
	for _, opt := range opts {
		opt.ApplyToHelper(options)
	}

	// Convert the object to unstructured to compare against our before copy.
	var err error
	h.after, err = ToUnstructured(obj)
	if err != nil {
		return err
	}

	// Determine if the object has status.
	if unstructuredHasStatus(h.after) && options.IncludeStatusObservedGeneration {
		// Set status.observedGeneration if we're asked to do so.
		if err := unstructured.SetNestedField(h.after.Object, h.after.GetGeneration(), "status", "observedGeneration"); err != nil {
			return err
		}

		// Restore the changes back to the original object.
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(h.after.Object, obj); err != nil {
			return err
		}
	}

	// Calculate and store the top-level field changes (e.g. "metadata", "spec", "status") we have before/after.
	h.changes, err = h.calculateChanges(obj)
	if err != nil {
		return err
	}

	specPatchCtx, specCancelFunc := context.WithTimeout(ctx, 3*time.Second)
	defer specCancelFunc()
	if err = h.patch(specPatchCtx, obj); err != nil {
		return errors.Wrapf(err, "patch spec failed")
	}

	statusPatchCtx, statusPatchCancelFunc := context.WithTimeout(ctx, 3*time.Second)
	defer statusPatchCancelFunc()
	if err = h.patchStatus(statusPatchCtx, obj); err != nil {
		return errors.Wrapf(err, "patch status failed")
	}

	return nil
}

// patch issues a patch for metadata and spec.
func (h *Helper) patch(ctx context.Context, obj client.Object) error {
	if !h.shouldPatch("metadata") && !h.shouldPatch("spec") {
		return nil
	}
	beforeObject, afterObject, err := h.calculatePatch(obj)
	if err != nil {
		return err
	}
	return h.client.Patch(ctx, afterObject, client.MergeFrom(beforeObject))
}

// patchStatus issues a patch if the status has changed.
func (h *Helper) patchStatus(ctx context.Context, obj client.Object) error {
	if !h.shouldPatch("status") {
		return nil
	}
	beforeObject, afterObject, err := h.calculatePatch(obj)
	if err != nil {
		return err
	}
	return h.client.Status().Patch(ctx, afterObject, client.MergeFrom(beforeObject))
}

// calculatePatch returns the before/after objects to be given in a controller-runtime patch, scoped down to the absolute necessary.
func (h *Helper) calculatePatch(afterObj client.Object) (client.Object, client.Object, error) {
	// Get a shallow unsafe copy of the before/after object in unstructured form.
	before := unsafeUnstructuredCopy(h.before)
	after := unsafeUnstructuredCopy(h.after)

	// We've now applied all modifications to local unstructured objects,
	// make copies of the original objects and convert them back.
	beforeObj := h.beforeObject.DeepCopyObject().(client.Object)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(before.Object, beforeObj); err != nil {
		return nil, nil, err
	}
	afterObj = afterObj.DeepCopyObject().(client.Object)
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(after.Object, afterObj); err != nil {
		return nil, nil, err
	}
	return beforeObj, afterObj, nil
}

func (h *Helper) shouldPatch(in string) bool {
	return h.changes[in]
}

// calculate changes tries to build a patch from the before/after objects we have
// and store in a map which top-level fields (e.g. `metadata`, `spec`, `status`, etc.) have changed.
func (h *Helper) calculateChanges(after client.Object) (map[string]bool, error) {
	// Calculate patch data.
	patch := client.MergeFrom(h.beforeObject)
	diff, err := patch.Data(after)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to calculate patch data")
	}

	// Unmarshal patch data into a local map.
	patchDiff := map[string]interface{}{}
	if err := json.Unmarshal(diff, &patchDiff); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal patch data into a map")
	}

	// Return the map.
	res := make(map[string]bool, len(patchDiff))
	for key := range patchDiff {
		res[key] = true
	}
	return res, nil
}
