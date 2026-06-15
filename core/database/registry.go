package database

import (
	"fmt"
	"sync"
)

// Registry manages multiple database connections of different types.
type Registry struct {
	mu            sync.RWMutex
	sqlDrivers    map[string]SQLDriver
	redisDrivers  map[string]RedisDriver
	vectorDrivers map[string]VectorDriver
	defaultDB     string
}

// NewRegistry creates a new database registry.
func NewRegistry() *Registry {
	return &Registry{
		sqlDrivers:    make(map[string]SQLDriver),
		redisDrivers:  make(map[string]RedisDriver),
		vectorDrivers: make(map[string]VectorDriver),
	}
}

// SetDefault sets the default database name.
func (r *Registry) SetDefault(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultDB = name
}

// Default returns the default database name.
func (r *Registry) Default() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.defaultDB
}

// RegisterSQL adds a SQL driver.
func (r *Registry) RegisterSQL(name string, driver SQLDriver) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.sqlDrivers[name]; exists {
		return fmt.Errorf("SQL driver %q already registered", name)
	}
	r.sqlDrivers[name] = driver
	return nil
}

// GetSQL returns a SQL driver by name.
func (r *Registry) GetSQL(name string) (SQLDriver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	driver, ok := r.sqlDrivers[name]
	if !ok {
		return nil, fmt.Errorf("SQL driver %q not found", name)
	}
	return driver, nil
}

// RegisterRedis adds a Redis driver.
func (r *Registry) RegisterRedis(name string, driver RedisDriver) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.redisDrivers[name]; exists {
		return fmt.Errorf("redis driver %q already registered", name)
	}
	r.redisDrivers[name] = driver
	return nil
}

// GetRedis returns a Redis driver by name.
func (r *Registry) GetRedis(name string) (RedisDriver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	driver, ok := r.redisDrivers[name]
	if !ok {
		return nil, fmt.Errorf("redis driver %q not found", name)
	}
	return driver, nil
}

// RegisterVector adds a vector driver.
func (r *Registry) RegisterVector(name string, driver VectorDriver) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.vectorDrivers[name]; exists {
		return fmt.Errorf("vector driver %q already registered", name)
	}
	r.vectorDrivers[name] = driver
	return nil
}

// GetVector returns a vector driver by name.
func (r *Registry) GetVector(name string) (VectorDriver, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	driver, ok := r.vectorDrivers[name]
	if !ok {
		return nil, fmt.Errorf("vector driver %q not found", name)
	}
	return driver, nil
}

// List returns all registered database names with their types.
func (r *Registry) List() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string)
	for name := range r.sqlDrivers {
		result[name] = "sql"
	}
	for name := range r.redisDrivers {
		result[name] = "redis"
	}
	for name := range r.vectorDrivers {
		result[name] = "vector"
	}
	return result
}

// Close closes all database connections.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error

	for name, driver := range r.sqlDrivers {
		if err := driver.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing SQL %q: %w", name, err))
		}
	}
	for name, driver := range r.redisDrivers {
		if err := driver.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing Redis %q: %w", name, err))
		}
	}
	for name, driver := range r.vectorDrivers {
		if err := driver.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing Vector %q: %w", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}
