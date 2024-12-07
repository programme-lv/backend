package evalsrvc

import (
	"log"
	"sync"
)

type ResultProcessorImpl struct {
	lock sync.Mutex
}

func (e *ResultProcessorImpl) Handle(msg Msg) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	log.Printf("processing tester result: %v", msg)
	return nil
}
