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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "cncos.io/box-controller/pkg/apis/box/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// BoxLister helps list Boxes.
// All objects returned here must be treated as read-only.
type BoxLister interface {
	// List lists all Boxes in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Box, err error)
	// Boxes returns an object that can list and get Boxes.
	Boxes(namespace string) BoxNamespaceLister
	BoxListerExpansion
}

// boxLister implements the BoxLister interface.
type boxLister struct {
	indexer cache.Indexer
}

// NewBoxLister returns a new BoxLister.
func NewBoxLister(indexer cache.Indexer) BoxLister {
	return &boxLister{indexer: indexer}
}

// List lists all Boxes in the indexer.
func (s *boxLister) List(selector labels.Selector) (ret []*v1alpha1.Box, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Box))
	})
	return ret, err
}

// Boxes returns an object that can list and get Boxes.
func (s *boxLister) Boxes(namespace string) BoxNamespaceLister {
	return boxNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// BoxNamespaceLister helps list and get Boxes.
// All objects returned here must be treated as read-only.
type BoxNamespaceLister interface {
	// List lists all Boxes in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Box, err error)
	// Get retrieves the Box from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Box, error)
	BoxNamespaceListerExpansion
}

// boxNamespaceLister implements the BoxNamespaceLister
// interface.
type boxNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Boxes in the indexer for a given namespace.
func (s boxNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Box, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Box))
	})
	return ret, err
}

// Get retrieves the Box from the indexer for a given namespace and name.
func (s boxNamespaceLister) Get(name string) (*v1alpha1.Box, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("box"), name)
	}
	return obj.(*v1alpha1.Box), nil
}
