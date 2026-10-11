package hybrid

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

type LifecycleRecord struct {
	Metadata ComponentMetadata `json:"metadata"`
	State    LifecycleState    `json:"state"`
	Error    string            `json:"error,omitempty"`
}

type componentEntry struct {
	component ManagedComponent
	metadata  ComponentMetadata
	state     LifecycleState
	err       error
}

// Registry is an in-process lifecycle registry. Replacements wait for active
// branch invocations because invocation holds a read lock through the call.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*componentEntry
}

func NewRegistry() *Registry { return &Registry{entries: make(map[string]*componentEntry)} }

func (r *Registry) Register(ctx context.Context, component ManagedComponent) error {
	if r == nil || component == nil {
		return fmt.Errorf("%w: nil registry or component", ErrIncompatible)
	}
	metadata := component.Metadata()
	if err := metadata.Validate(); err != nil {
		return err
	}
	if err := checkComponentInterface(metadata.Role, component); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[metadata.ID]; exists {
		return fmt.Errorf("%w: component ID %q already registered", ErrIncompatible, metadata.ID)
	}
	entry := &componentEntry{component: component, metadata: metadata, state: LifecycleInitializing}
	r.entries[metadata.ID] = entry
	if err := component.Initialize(ctx); err != nil {
		entry.err = err
		if errors.Is(err, ErrComponentUnavailable) {
			entry.state = LifecycleUnavailable
		} else {
			entry.state = LifecycleFailed
		}
		return err
	}
	entry.state = LifecycleReady
	return nil
}

// Replace initializes the replacement before swapping it into service. If
// initialization fails the existing component remains registered and ready.
func (r *Registry) Replace(ctx context.Context, id string, replacement ManagedComponent) error {
	if r == nil || replacement == nil {
		return fmt.Errorf("%w: nil registry or replacement", ErrIncompatible)
	}
	metadata := replacement.Metadata()
	if metadata.ID != id {
		return fmt.Errorf("%w: replacement ID %q does not match %q", ErrIncompatible, metadata.ID, id)
	}
	if err := metadata.Validate(); err != nil {
		return err
	}
	if err := checkComponentInterface(metadata.Role, replacement); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	old, exists := r.entries[id]
	if !exists {
		return fmt.Errorf("%w: component %q", ErrNotRegistered, id)
	}
	if old.metadata.Role != metadata.Role || old.metadata.Head != metadata.Head {
		return fmt.Errorf("%w: replacement changes component role or head kind", ErrIncompatible)
	}
	if err := replacement.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize replacement %q: %w", id, err)
	}
	r.entries[id] = &componentEntry{component: replacement, metadata: metadata, state: LifecycleReady}
	if err := old.component.Shutdown(ctx); err != nil {
		return fmt.Errorf("replacement active but previous component shutdown failed: %w", err)
	}
	return nil
}

func checkComponentInterface(role ComponentRole, component ManagedComponent) error {
	valid := false
	switch role {
	case RoleEvent:
		_, valid = component.(EventBranch)
	case RoleTopoTemporal:
		_, valid = component.(TopoTemporalBranch)
	case RoleHomeAdaptation:
		_, valid = component.(HomeAdaptationBranch)
	case RoleCognitiveHead:
		_, valid = component.(CognitiveHead)
	}
	if !valid {
		return fmt.Errorf("%w: component does not implement the %q contract", ErrIncompatible, role)
	}
	return nil
}

func (r *Registry) Records() []LifecycleRecord {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0, len(r.entries))
	for id := range r.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]LifecycleRecord, 0, len(ids))
	for _, id := range ids {
		entry := r.entries[id]
		record := LifecycleRecord{Metadata: entry.metadata, State: entry.state}
		if entry.err != nil {
			record.Error = entry.err.Error()
		}
		result = append(result, record)
	}
	return result
}

func (r *Registry) Describe(role ComponentRole, head HeadKind) (ComponentMetadata, LifecycleState, bool, error) {
	if r == nil {
		return ComponentMetadata{}, LifecycleUnavailable, false, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	var selected *componentEntry
	for _, entry := range r.entries {
		if entry.metadata.Role == role && (head == "" || entry.metadata.Head == head) {
			if selected != nil {
				return ComponentMetadata{}, LifecycleFailed, false, fmt.Errorf("%w: multiple %s components", ErrIncompatible, role)
			}
			selected = entry
		}
	}
	if selected == nil {
		return ComponentMetadata{}, LifecycleUnavailable, false, nil
	}
	return selected.metadata, selected.state, true, nil
}

func (r *Registry) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]string, 0, len(r.entries))
	for id := range r.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var failures []error
	for _, id := range ids {
		entry := r.entries[id]
		if entry.state == LifecycleStopped {
			continue
		}
		if err := entry.component.Shutdown(ctx); err != nil {
			failures = append(failures, fmt.Errorf("shutdown %q: %w", id, err))
		}
		entry.state = LifecycleStopped
	}
	return errors.Join(failures...)
}

func (r *Registry) invoke(role ComponentRole, head HeadKind, call func(ManagedComponent) error) (ComponentMetadata, LifecycleState, error) {
	if r == nil {
		return ComponentMetadata{}, LifecycleUnavailable, ErrNotRegistered
	}
	r.mu.RLock()
	var selected *componentEntry
	for _, entry := range r.entries {
		if entry.metadata.Role == role && (head == "" || entry.metadata.Head == head) {
			if selected != nil {
				r.mu.RUnlock()
				return ComponentMetadata{}, LifecycleFailed, fmt.Errorf("%w: multiple %s components", ErrIncompatible, role)
			}
			selected = entry
		}
	}
	if selected == nil {
		r.mu.RUnlock()
		return ComponentMetadata{}, LifecycleUnavailable, fmt.Errorf("%w: role %q", ErrNotRegistered, role)
	}
	metadata, state, component := selected.metadata, selected.state, selected.component
	if state != LifecycleReady {
		err := selected.err
		r.mu.RUnlock()
		if err != nil {
			if state == LifecycleFailed {
				return metadata, state, fmt.Errorf("%w: %v", ErrComponentFailed, err)
			}
			return metadata, state, fmt.Errorf("%w: %v", ErrComponentUnavailable, err)
		}
		return metadata, state, fmt.Errorf("%w: component %q is %s", ErrComponentUnavailable, metadata.ID, state)
	}
	if component.Metadata() != metadata {
		r.mu.RUnlock()
		err := fmt.Errorf("%w: component metadata changed after registration", ErrIncompatible)
		r.markFailure(metadata.ID, selected, err)
		return metadata, LifecycleFailed, fmt.Errorf("%w: %v", ErrComponentFailed, err)
	}
	err := call(component)
	r.mu.RUnlock()
	if err != nil {
		if errors.Is(err, ErrExecutionDisabled) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return metadata, state, fmt.Errorf("%w: %v", ErrComponentUnavailable, err)
		}
		r.markFailure(metadata.ID, selected, err)
		if errors.Is(err, ErrComponentUnavailable) {
			return metadata, LifecycleUnavailable, err
		}
		return metadata, LifecycleFailed, fmt.Errorf("%w: %v", ErrComponentFailed, err)
	}
	return metadata, LifecycleReady, nil
}

func (r *Registry) markFailure(id string, expected *componentEntry, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if entry := r.entries[id]; entry == expected {
		entry.err = err
		if errors.Is(err, ErrComponentUnavailable) {
			entry.state = LifecycleUnavailable
		} else {
			entry.state = LifecycleFailed
		}
	}
}
