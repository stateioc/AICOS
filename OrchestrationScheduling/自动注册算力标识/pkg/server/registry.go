package server

import (
	"register-power-resources/pkg/apis"
	"sync"
)

type Registry struct {
	sync.RWMutex
	Resources map[string]*apis.NodeResourceInfo
}

func (r *Registry) AddResource(resource *apis.NodeResourceInfo) {
	r.Lock()
	defer r.Unlock()
	r.Resources[resource.ID] = resource
}

func (r *Registry) DeleteResource(id string) {
	r.Lock()
	defer r.Unlock()
	delete(r.Resources, id)
}

func (r *Registry) GetResources(offset, limit int) []*apis.NodeResourceInfo {
	r.RLock()
	defer r.RUnlock()

	resources := make([]*apis.NodeResourceInfo, 0, 1000)
	for _, resource := range r.Resources {
		resources = append(resources, resource)
	}
	return resources
}

var registry = &Registry{
	Resources: make(map[string]*apis.NodeResourceInfo),
}
