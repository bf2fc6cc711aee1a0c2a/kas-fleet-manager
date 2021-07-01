// Aquires and exclusive lock against the test suite so that it is the only scenario executing until the secnario
// finishes executing.
//    Given LOCK
// Releases the exclusive lock previously acquired.  Not required, any aquired lock is automatically released at the
// end of scenario.
//    Given UNLOCK
// Sleeps for the given number of seconds.
//    And I sleep for 0.5 second
package cucumber

import (
	"sync"
	"time"

	"github.com/cucumber/godog"
)

var testCaseLock sync.RWMutex

func init() {
	StepModules = append(StepModules, func(ctx *godog.ScenarioContext, s *TestScenario) {
		ctx.Step(`^LOCK-*$`, s.lock)
		ctx.Step(`^UNLOCK-*$`, s.unlock)
		ctx.Step(`^I sleep for (\d+(.\d+)?) seconds?$`, s.iSleepForSecond)

		ctx.BeforeScenario(func(sc *godog.Scenario) {
			testCaseLock.RLock()
		})
		ctx.AfterScenario(func(sc *godog.Scenario, err error) {
			if s.hasTestCaseLock {
				testCaseLock.Unlock()
			} else {
				testCaseLock.RUnlock()
			}
		})
	})
}

func (s *TestScenario) lock() error {
	// User might have already locked the scenario.
	if !s.hasTestCaseLock {
		// Convert to write lock to be the only executing scenario.
		testCaseLock.RUnlock()
		testCaseLock.Lock()
		s.hasTestCaseLock = true
	}
	return nil
}

func (s *TestScenario) unlock() error {
	// User might have already unlocked the scenario.
	if s.hasTestCaseLock {
		// Convert to read lock to to allow other scenarios to keep running.
		testCaseLock.Unlock()
		testCaseLock.RLock()
		s.hasTestCaseLock = false
	}
	return nil
}

func (s *TestScenario) iSleepForSecond(seconds float64) error {
	time.Sleep(time.Duration(seconds * float64(time.Second)))
	return nil
}
