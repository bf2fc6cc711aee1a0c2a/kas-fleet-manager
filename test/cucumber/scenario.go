// Aquires and exclusive lock against the test suite so that it is the only scenario executing until the secnario
// finishes executing.
//    Given this is the only scenario running
package cucumber

import (
	"github.com/cucumber/godog"
	"sync"
)

var testCaseLock sync.RWMutex

func init() {
	StepModules = append(StepModules, func(ctx *godog.ScenarioContext, s *TestScenario) {
		ctx.Step(`^this is the only scenario running$`, s.thisIsTheOnlyScenarioRunning)
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

func (s *TestScenario) thisIsTheOnlyScenarioRunning() error {
	// Convert to write lock to be the only executing scenario.
	testCaseLock.RUnlock()
	testCaseLock.Lock()
	s.hasTestCaseLock = true
	return nil
}
