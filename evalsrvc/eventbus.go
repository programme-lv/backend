package evalsrvc

import (
	"sync"

	"github.com/google/uuid"
)

type MockEvalEventBus struct {
	lock sync.Mutex
	subs []chan uuid.UUID
}

// receive stream of ids of evaluations when all tester results are received
func (bus *MockEvalEventBus) AwaitReceivedAllFromTester() <-chan uuid.UUID {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	ch := make(chan uuid.UUID, 1)
	bus.subs = append(bus.subs, ch)
	return ch
}

// notify all subscribers that all tester results are received for a given evaluation
func (bus *MockEvalEventBus) BroadcastReceivedAllFromTester(id uuid.UUID) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	for _, sub := range bus.subs {
		sub <- id
	}
}

func NewMockEvalEventBus() *MockEvalEventBus {
	return &MockEvalEventBus{
		subs: []chan uuid.UUID{},
	}
}
