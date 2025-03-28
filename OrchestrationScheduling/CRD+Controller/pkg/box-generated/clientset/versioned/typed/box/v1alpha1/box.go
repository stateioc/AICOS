/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	scheme "cncos.io/box-controller/pkg/box-generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// BoxesGetter has a method to return a BoxInterface.
// A group's client should implement this interface.
type BoxesGetter interface {
	Boxes(namespace string) BoxInterface
}

// BoxInterface has methods to work with Box resources.
type BoxInterface interface {
	Create(ctx context.Context, box *v1alpha1.Box, opts v1.CreateOptions) (*v1alpha1.Box, error)
	Update(ctx context.Context, box *v1alpha1.Box, opts v1.UpdateOptions) (*v1alpha1.Box, error)
	UpdateStatus(ctx context.Context, box *v1alpha1.Box, opts v1.UpdateOptions) (*v1alpha1.Box, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Box, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.BoxList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Box, err error)
	BoxExpansion
}

// boxes implements BoxInterface
type boxes struct {
	client rest.Interface
	ns     string
}

// newBoxes returns a Boxes
func newBoxes(c *CncosV1alpha1Client, namespace string) *boxes {
	return &boxes{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the box, and returns the corresponding box object, and an error if there is any.
func (c *boxes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Box, err error) {
	result = &v1alpha1.Box{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("boxes").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Boxes that match those selectors.
func (c *boxes) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.BoxList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.BoxList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("boxes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested boxes.
func (c *boxes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("boxes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a box and creates it.  Returns the server's representation of the box, and an error, if there is any.
func (c *boxes) Create(ctx context.Context, box *v1alpha1.Box, opts v1.CreateOptions) (result *v1alpha1.Box, err error) {
	result = &v1alpha1.Box{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("boxes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(box).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a box and updates it. Returns the server's representation of the box, and an error, if there is any.
func (c *boxes) Update(ctx context.Context, box *v1alpha1.Box, opts v1.UpdateOptions) (result *v1alpha1.Box, err error) {
	result = &v1alpha1.Box{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("boxes").
		Name(box.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(box).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *boxes) UpdateStatus(ctx context.Context, box *v1alpha1.Box, opts v1.UpdateOptions) (result *v1alpha1.Box, err error) {
	result = &v1alpha1.Box{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("boxes").
		Name(box.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(box).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the box and deletes it. Returns an error if one occurs.
func (c *boxes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("boxes").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *boxes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("boxes").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched box.
func (c *boxes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Box, err error) {
	result = &v1alpha1.Box{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("boxes").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
