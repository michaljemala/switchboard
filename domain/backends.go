package domain

import (
	"sync"

	"github.com/pivotal-cf-experimental/switchboard/config"
	"github.com/pivotal-golang/lager"
)

var BackendProvider = NewBackend

type Backends interface {
	All() <-chan Backend
	Any() Backend
	Active() Backend
	SetHealthy(backend Backend)
	SetUnhealthy(backend Backend)
	Healthy() <-chan Backend
	AsJSON() []BackendJSON
}

type backends struct {
	mutex  sync.RWMutex
	all    map[Backend]bool
	active Backend
	logger lager.Logger
}

func NewBackends(backendConfigs []config.Backend, logger lager.Logger) Backends {
	b := &backends{
		logger: logger,
		all:    make(map[Backend]bool),
	}

	for _, bc := range backendConfigs {
		backend := BackendProvider(
			bc.Name,
			bc.Host,
			bc.Port,
			bc.HealthcheckPort,
			logger,
		)

		b.all[backend] = true
	}

	b.active = b.unsafeNextHealthy()

	return b
}

// Maintains a lock while iterating over all backends
func (b *backends) All() <-chan Backend {
	b.mutex.RLock()

	ch := make(chan Backend, len(b.all))
	go func() {
		defer b.mutex.RUnlock()

		for backend := range b.all {
			ch <- backend
		}
		close(ch)
	}()

	return ch
}

// Maintains a lock while iterating over healthy backends
func (b *backends) Healthy() <-chan Backend {
	b.mutex.RLock()

	c := make(chan Backend, len(b.all))
	go func() {
		defer b.mutex.RUnlock()

		for backend, healthy := range b.all {
			if healthy {
				c <- backend
			}
		}

		close(c)
	}()

	return c
}

func (b *backends) Any() Backend {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for backend := range b.all {
		return backend
	}

	return nil
}

func (b *backends) Active() Backend {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.active
}

func (b *backends) SetHealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.all[backend] = true
	b.logger.Info("Backend became healthy again.")
	if b.active == nil {
		b.active = backend
	}
}

func (b *backends) SetUnhealthy(backend Backend) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.all[backend] = false
	if b.active == backend {
		b.logger.Info("Active backend became unhealthy. Switching over to next available...")
		backend.SeverConnections()
		b.active = b.unsafeNextHealthy()
	}
}

func (b *backends) AsJSON() []BackendJSON {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	backendsJSON := []BackendJSON{}
	for backend, healthy := range b.all {
		backendJSON := backend.AsJSON()
		backendJSON.Healthy = healthy
		backendJSON.Active = (b.active == backend)
		backendsJSON = append(backendsJSON, backendJSON)
	}

	return backendsJSON
}

func (b *backends) unsafeNextHealthy() Backend {
	for backend, healthy := range b.all {
		if healthy {
			return backend
		}
	}
	return nil
}
