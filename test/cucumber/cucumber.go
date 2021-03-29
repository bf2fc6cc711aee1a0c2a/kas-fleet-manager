// Package cucumber allows you to use cucumber to execute Gherkin based
// BDD test scenarios with some helpful API testing step implementations.
//
// Some steps allow you store variables or use those variables.  The variables
// are scoped to the Scenario.  The http response state is stored in the users
// session.  Switching users will switch the session.  Scenarios are executed
// concurrently.  The same user can be logged into two scenarios, but each scenario
// has a different session.
//
// Note: be careful using the same user/organization across different scenarios since
// they will likely see unexpected API mutations done in the other scenarios.
//
// Using in a test
//  func TestMain(m *testing.M) {
//
//	// Startup all the services and mocks that are needed to test the
//	// connector features.
//	t := &testing.T{}
//
//	connectorTypeService := mocks.NewConnectorTypeMock(t)
//	defer connectorTypeService.Close()
//	environments.Environment().Config.ConnectorsConfig.Enabled = true
//	environments.Environment().Config.ConnectorsConfig.ConnectorTypesDir = ""
//	environments.Environment().Config.ConnectorsConfig.ConnectorTypeSvcUrls = []string{connectorTypeService.URL}
//
//	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
//	defer ocmServer.Close()
//
//	h, _, teardown := test.RegisterIntegration(t, ocmServer)
//	defer teardown()
//
//	cucumber.TestMain(m, h)
//
//}

package cucumber

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/itchyny/gojq"
)

// TestSuite holds the sate global to all the test scenarios.
// It is accessed concurrently from all test scenarios.
type TestSuite struct {
	ApiURL    string
	Helper    *test.Helper
	Mu        sync.Mutex
	users     map[string]*TestUser
	nextOrgId uint32
}

// TestUser represents a user that can login to the system.  The same users are shared by
// the different test scenarios.
type TestUser struct {
	Name  string
	Token string
	Ctx   context.Context
	Mu    sync.Mutex
}

// TestScenario holds that state of single scenario.  It is not accessed
// concurrently.
type TestScenario struct {
	Suite       *TestSuite
	CurrentUser string
	PathPrefix  string
	sessions    map[string]*TestSession
	Variables   map[string]string
}

func (s *TestScenario) User() *TestUser {
	s.Suite.Mu.Lock()
	defer s.Suite.Mu.Unlock()
	return s.Suite.users[s.CurrentUser]
}

func (s *TestScenario) Session() *TestSession {
	result := s.sessions[s.CurrentUser]
	if result == nil {
		result = &TestSession{
			TestUser: s.User(),
			Client:   &http.Client{},
		}
		s.sessions[s.CurrentUser] = result
	}
	return result
}

// Expand replaces ${var} or $var in the string based on saved Variables in the session/test scenario.
func (s *TestScenario) Expand(value string) string {
	session := s.Session()
	return os.Expand(value, func(name string) string {
		if strings.HasPrefix(name, "response.") {

			selector := strings.TrimPrefix(name, "response")
			query, err := gojq.Parse(selector)
			if err != nil {
				return s.Variables[name]
			}

			j, err := session.RespJson()
			if err != nil {
				return s.Variables[name]
			}

			iter := query.Run(j)
			if next, found := iter.Next(); found {
				switch next := next.(type) {
				case string:
					return next
				case float64:
					return fmt.Sprintf("%f", next)
				case float32:
					return fmt.Sprintf("%f", next)
				default:
					return fmt.Sprintf("%s", next)
				}
			}
		}
		return s.Variables[name]
	})
}

// TestSession holds the http context for a user kinda like a browser.  Each scenario
// had a different session even if using the same user.
type TestSession struct {
	TestUser            *TestUser
	Client              *http.Client
	Resp                *http.Response
	Ctx                 context.Context
	RespBytes           []byte
	respJson            interface{}
	AuthorizationHeader string
	EventStream         bool
	EventStreamEvents   chan interface{}
}

// RespJson returns the last http response body as json
func (s *TestSession) RespJson() (interface{}, error) {
	if s.respJson == nil {
		if err := json.Unmarshal(s.RespBytes, &s.respJson); err != nil {
			return nil, err
		}
	}
	return s.respJson, nil
}

// StepModules is the list of functions used to add steps to a godog.ScenarioContext, you can
// add more to this list if you need test TestSuite specific steps.
var StepModules []func(ctx *godog.ScenarioContext, s *TestScenario)

func (suite *TestSuite) InitializeScenario(ctx *godog.ScenarioContext) {
	s := &TestScenario{
		Suite:     suite,
		sessions:  map[string]*TestSession{},
		Variables: map[string]string{},
	}

	for _, module := range StepModules {
		module(ctx, s)
	}
}

var opts = godog.Options{
	Output:      colors.Colored(os.Stdout),
	Format:      "progress", // can define default values
	Paths:       []string{"features"},
	Randomize:   time.Now().UTC().UnixNano(), // randomize TestScenario execution order
	Concurrency: 10,
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

// TestMain runs the scenarios found in the "features" directory.  If m is not nil, it
// also runs it's tests.  Panics if helper is nil.
func TestMain(m *testing.M, helper *test.Helper) {
	helper.NewApiClient()
	s := &TestSuite{
		Helper:    helper,
		ApiURL:    helper.NewApiClient().GetConfig().BasePath,
		users:     map[string]*TestUser{},
		nextOrgId: 20000000,
	}

	// Generate lots of org id's that scenarios can use to avoid
	// conflicting with each other..
	allow := helper.Env().Config.AccessControlList
	if allow != nil {
		for i := 0; i < 1000; i++ {
			allow.AllowList.Organisations = append(allow.AllowList.Organisations, config.Organisation{
				Id:                  fmt.Sprintf("%d", 20000000+i),
				AllowAll:            true,
				MaxAllowedInstances: 1,
			})
		}
	}

	for _, arg := range os.Args[1:] {
		if arg == "-test.v=true" { // go test transforms -v option
			opts.Format = "pretty"
		}
	}

	flag.Parse()
	opts.Paths = flag.Args()

	status := godog.TestSuite{
		Name:                "godogs",
		ScenarioInitializer: s.InitializeScenario,
		Options:             &opts,
	}.Run()

	// Optional: Run `testing` package's logic besides godog.
	if m != nil {
		if st := m.Run(); st > status {
			status = st
		}
	}

	os.Exit(status)
}
