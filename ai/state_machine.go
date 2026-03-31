package ai

import "sync"

// AIState represents the current state of an AI assistant in a pane.
//
//nolint:revive // reason: "ai.AIState" reads naturally; renaming to "State" would be ambiguous.
type AIState string

// AIState constants represent the possible states of an AI assistant.
const (
	AIStateIdle            AIState = "idle"
	AIStateWorking         AIState = "working"
	AIStateWaitingApproval AIState = "waiting_approval"
	AIStateUnknown         AIState = "unknown"
)

// StateChangeHandler is called when a pane's AI state changes.
type StateChangeHandler func(paneKey string, from, to AIState)

// StateMachine tracks AI assistant state per terminal pane.
type StateMachine struct {
	mu       sync.RWMutex
	states   map[string]AIState
	handlers []StateChangeHandler
}

// NewStateMachine creates a new StateMachine.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		states: make(map[string]AIState),
	}
}

// Get returns the current state for a pane, or AIStateUnknown if unset.
func (sm *StateMachine) Get(paneKey string) AIState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if s, ok := sm.states[paneKey]; ok {
		return s
	}
	return AIStateUnknown
}

// Transition updates the state for a pane. Notifies handlers only if state changed.
func (sm *StateMachine) Transition(paneKey string, newState AIState) {
	sm.mu.Lock()
	old, exists := sm.states[paneKey]
	if !exists {
		old = AIStateUnknown
	}
	if old == newState {
		sm.mu.Unlock()
		return
	}
	sm.states[paneKey] = newState
	handlers := make([]StateChangeHandler, len(sm.handlers))
	copy(handlers, sm.handlers)
	sm.mu.Unlock()

	for _, h := range handlers {
		h(paneKey, old, newState)
	}
}

// OnStateChange registers a handler called on every state transition.
func (sm *StateMachine) OnStateChange(handler StateChangeHandler) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.handlers = append(sm.handlers, handler)
}

// AllStates returns a copy of all tracked pane states.
func (sm *StateMachine) AllStates() map[string]AIState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	result := make(map[string]AIState, len(sm.states))
	for k, v := range sm.states {
		result[k] = v
	}
	return result
}
