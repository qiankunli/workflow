/*
Copyright 2021 workflow authors. All rights reserved.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/qiankunli/workflow/pkg/apis/workflow/v1alpha1"
	scheme "github.com/qiankunli/workflow/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// StepsGetter has a method to return a StepInterface.
// A group's client should implement this interface.
type StepsGetter interface {
	Steps(namespace string) StepInterface
}

// StepInterface has methods to work with Step resources.
type StepInterface interface {
	Create(ctx context.Context, step *v1alpha1.Step, opts v1.CreateOptions) (*v1alpha1.Step, error)
	Update(ctx context.Context, step *v1alpha1.Step, opts v1.UpdateOptions) (*v1alpha1.Step, error)
	UpdateStatus(ctx context.Context, step *v1alpha1.Step, opts v1.UpdateOptions) (*v1alpha1.Step, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Step, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.StepList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Step, err error)
	StepExpansion
}

// steps implements StepInterface
type steps struct {
	client rest.Interface
	ns     string
}

// newSteps returns a Steps
func newSteps(c *WorkflowV1alpha1Client, namespace string) *steps {
	return &steps{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the step, and returns the corresponding step object, and an error if there is any.
func (c *steps) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Step, err error) {
	result = &v1alpha1.Step{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("steps").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Steps that match those selectors.
func (c *steps) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.StepList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.StepList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("steps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested steps.
func (c *steps) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("steps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a step and creates it.  Returns the server's representation of the step, and an error, if there is any.
func (c *steps) Create(ctx context.Context, step *v1alpha1.Step, opts v1.CreateOptions) (result *v1alpha1.Step, err error) {
	result = &v1alpha1.Step{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("steps").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(step).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a step and updates it. Returns the server's representation of the step, and an error, if there is any.
func (c *steps) Update(ctx context.Context, step *v1alpha1.Step, opts v1.UpdateOptions) (result *v1alpha1.Step, err error) {
	result = &v1alpha1.Step{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("steps").
		Name(step.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(step).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *steps) UpdateStatus(ctx context.Context, step *v1alpha1.Step, opts v1.UpdateOptions) (result *v1alpha1.Step, err error) {
	result = &v1alpha1.Step{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("steps").
		Name(step.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(step).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the step and deletes it. Returns an error if one occurs.
func (c *steps) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("steps").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *steps) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("steps").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched step.
func (c *steps) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Step, err error) {
	result = &v1alpha1.Step{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("steps").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
