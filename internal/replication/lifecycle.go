package replication

// LifecycleStore provides DB operations for child lifecycle (TS-aligned).
type LifecycleStore interface {
	InsertChildLifecycleEvent(childID, fromState, toState, reason, metadata string) error
	GetLatestChildState(childID string) (string, error)
	UpdateChildStatus(id, status string) error
}

// ChildLifecycle manages child state transitions with audit trail.
type ChildLifecycle interface {
	// Transition records a state change if valid; updates children.status and child_lifecycle_events.
	Transition(childID, toState, reason string) error
	// Current returns the latest lifecycle state for a child (from events or children.status).
	Current(childID string) (string, error)
}

// DBChildLifecycle is a DB-backed ChildLifecycle implementation.
type DBChildLifecycle struct {
	Store LifecycleStore
}

// Transition implements ChildLifecycle.
func (l *DBChildLifecycle) Transition(childID, toState, reason string) error {
	if l.Store == nil {
		return nil
	}
	current, err := l.Store.GetLatestChildState(childID)
	if err != nil {
		return err
	}
	if current == "" {
		current = StateRequested
	}
	allowed := ValidTransitions[current]
	valid := false
	for _, a := range allowed {
		if a == toState {
			valid = true
			break
		}
	}
	if !valid {
		return nil // invalid transition: no-op (TS: fail silently or return error per design)
	}
	if err := l.Store.InsertChildLifecycleEvent(childID, current, toState, reason, "{}"); err != nil {
		return err
	}
	return l.Store.UpdateChildStatus(childID, toState)
}

// Current implements ChildLifecycle.
func (l *DBChildLifecycle) Current(childID string) (string, error) {
	if l.Store == nil {
		return "", nil
	}
	return l.Store.GetLatestChildState(childID)
}
