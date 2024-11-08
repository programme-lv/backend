package submsrvc

import "context"

func (s *SubmissionSrvc) StartStreamingSubmListUpdates(ctx context.Context) {
	sendUpdate := func(update *SubmissionListUpdate) {
		s.listenerLock.Lock()
		for _, listener := range s.listeners {
			if len(listener) == cap(listener) {
				<-listener
			}
			listener <- update
		}
		s.listenerLock.Unlock()
	}

	for {
		select {
		case created := <-s.submCreated:
			// notify all listeners about the new submission
			update := &SubmissionListUpdate{
				SubmCreated: created,
			}
			sendUpdate(update)
		case stateUpdate := <-s.evalStageUpd:
			// notify all listeners about the state update
			update := &SubmissionListUpdate{
				StateUpdate: &EvalStageUpd{
					SubmUuid: stateUpdate.SubmUuid,
					EvalUuid: stateUpdate.EvalUuid,
					NewStage: stateUpdate.NewStage,
				},
			}
			sendUpdate(update)
		case testgroupScoringResUpdate := <-s.tGroupScoreUpd:
			// notify all listeners about the testgroup result update
			update := &SubmissionListUpdate{
				TestgroupResUpdate: &TGroupScoreUpd{
					SubmUUID:      testgroupScoringResUpdate.SubmUUID,
					EvalUUID:      testgroupScoringResUpdate.EvalUUID,
					TestGroupID:   testgroupScoringResUpdate.TestGroupID,
					AcceptedTests: testgroupScoringResUpdate.AcceptedTests,
					WrongTests:    testgroupScoringResUpdate.WrongTests,
					UntestedTests: testgroupScoringResUpdate.UntestedTests,
				},
			}

			sendUpdate(update)
		case atomicTestsScoringResUpdate := <-s.tSetScoreUpd:
			update := &SubmissionListUpdate{
				TestsResUpdate: atomicTestsScoringResUpdate,
			}

			sendUpdate(update)
		}
	}
}

type Streamee interface {
	Send(*SubmissionListUpdate) error
	Close() error
}

// StreamSubmissionUpdates implements submissions.Service.
func (s *SubmissionSrvc) StreamSubmissionUpdates(ctx context.Context, stream Streamee) (err error) {
	// register myself as a listener to the submission updates
	myChan := make(chan *SubmissionListUpdate, 10000)
	s.listenerLock.Lock()
	s.listeners = append(s.listeners, myChan)
	s.listenerLock.Unlock()

	defer func() {
		// lock listener slice
		s.listenerLock.Lock()
		// remove myself from the listeners slice
		for i, listener := range s.listeners {
			if listener == myChan {
				s.listeners = append(s.listeners[:i], s.listeners[i+1:]...)
				break
			}
		}
		s.listenerLock.Unlock()
		close(myChan)
	}()

	for {
		select {
		case <-ctx.Done():
			return stream.Close()
		case update := <-myChan:
			err = stream.Send(update)
			if err != nil {
				return stream.Close()
			}
		}
	}
}
