package testing

import (
	"github.com/readeck/readeck/internal/bus"
	"github.com/readeck/readeck/pkg/superbus"
)

func startEventManager() {
	bus.LoadCustom("memory", NewEventManager(), superbus.NewMemStore())
}

// Events returns the EventManager
func Events() *EventManager {
	return bus.Events().(*EventManager)
}

// Store return the MemStore
func Store() *superbus.MemStore {
	return bus.Store().(*superbus.MemStore)
}

// EventManager is a in memory event manager that
// stores every event in a list.
type EventManager struct {
	recorder map[string][][]byte
}

// NewEventManager returns a new EventManager instance.
func NewEventManager() *EventManager {
	return &EventManager{
		recorder: make(map[string][][]byte),
	}
}

// Listen does nothing
func (m *EventManager) Listen() {
}

// Stop stops the Event manager. In this case it empties the queue
// of recorded events.
func (m *EventManager) Stop() {
	// Stop is called on a test's teardown, so we clean up everrything there
	m.recorder = make(map[string][][]byte)
}

// Push adds an event to the queue.
func (m *EventManager) Push(name string, value []byte) error {
	m.recorder[name] = append(m.recorder[name], value)
	return nil
}

// On registers an event handler. In this case, it does nothing.
func (m *EventManager) On(_ string, _ superbus.EventHandler) {
}

// Records returns the recorded events for a given event name.
func (m *EventManager) Records(name string) [][]byte {
	return m.recorder[name]
}
