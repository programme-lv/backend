package subm

import "context"

func (s *SubmissionSrvc) StartStreamingSubmListUpdates(ctx context.Context) {
	sendUpdate := func(update *SubmissionListUpdate) {
		s.updateListenerLock.Lock()
		for _, listener := range s.updateListeners {
			if len(listener) == cap(listener) {
				<-listener
			}
			listener <- update
		}
		s.updateListenerLock.Unlock()
	}

	for {
		select {
		case created := <-s.createdSubmChan:
			// notify all listeners about the new submission
			update := &SubmissionListUpdate{
				SubmCreated: created,
			}
			sendUpdate(update)
		case stateUpdate := <-s.updateSubmStateChan:
			// notify all listeners about the state update
			update := &SubmissionListUpdate{
				StateUpdate: &SubmissionStateUpdate{
					SubmUuid: stateUpdate.SubmUuid,
					EvalUuid: stateUpdate.EvalUuid,
					NewState: stateUpdate.NewState,
				},
			}
			sendUpdate(update)
		case testgroupResUpdate := <-s.updateTestgroupResChan:
			// notify all listeners about the testgroup result update
			update := &SubmissionListUpdate{
				TestgroupResUpdate: &TestgroupScoreUpdate{
					SubmUUID:      testgroupResUpdate.SubmUuid,
					EvalUUID:      testgroupResUpdate.EvalUuid,
					TestGroupID:   testgroupResUpdate.TestgroupId,
					AcceptedTests: testgroupResUpdate.AcceptedTests,
					WrongTests:    testgroupResUpdate.WrongTests,
					UntestedTests: testgroupResUpdate.UntestedTests,
				},
			}

			sendUpdate(update)
		}
	}
}

// StreamSubmissionUpdates implements submissions.Service.
// func (s *SubmissionSrvc) StreamSubmissionUpdates(ctx context.Context, p StreamSubmissionUpdatesServerStream) (err error) {
// 	// register myself as a listener to the submission updates
// 	myChan := make(chan *SubmissionListUpdate, 1000)
// 	s.updateListenerLock.Lock()
// 	s.updateListeners = append(s.updateListeners, myChan)
// 	s.updateListenerLock.Unlock()

// 	defer func() {
// 		// lock listener slice
// 		s.updateListenerLock.Lock()
// 		// remove myself from the listeners slice
// 		for i, listener := range s.updateListeners {
// 			if listener == myChan {
// 				s.updateListeners = append(s.updateListeners[:i], s.updateListeners[i+1:]...)
// 				break
// 			}
// 		}
// 		s.updateListenerLock.Unlock()
// 		close(myChan)
// 	}()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return p.Close()
// 		case update := <-myChan:
// 			err = p.Send(update)
// 			if err != nil {
// 				return p.Close()
// 			}
// 		}
// 	}
// }
