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
//	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
//	defer ocmServer.Close()
//
//	h, _, teardown := test.RegisterIntegration(&testing.T{}, ocmServer)
//	defer teardown()
//
//	cucumber.TestMain(h)
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
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/quota_management"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/itchyny/gojq"
	"github.com/pmezard/go-difflib/difflib"
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
	Name     string
	Token    string
	UserName string
	Ctx      context.Context
	Mu       sync.Mutex
}

// TestScenario holds that state of single scenario.  It is not accessed
// concurrently.
type TestScenario struct {
	Suite           *TestSuite
	CurrentUser     string
	PathPrefix      string
	sessions        map[string]*TestSession
	Variables       map[string]interface{}
	hasTestCaseLock bool
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
			Header:   http.Header{},
		}
		s.sessions[s.CurrentUser] = result
	}
	return result
}

func (s *TestScenario) JsonMustMatch(actual, expected string, expand bool) error {

	var actualParsed interface{}
	err := json.Unmarshal([]byte(actual), &actualParsed)
	if err != nil {
		return fmt.Errorf("error parsing actual json: %v\njson was:\n%s\n", err, actual)
	}

	var expectedParsed interface{}
	expanded := expected
	if expand {
		expanded, err = s.Expand(expected, []string{"defs", "ref"})
		if err != nil {
			return err
		}
	}

	if err := json.Unmarshal([]byte(expanded), &expectedParsed); err != nil {
		return fmt.Errorf("error parsing expected json: %v\njson was:\n%s\n", err, expanded)
	}

	if !reflect.DeepEqual(expectedParsed, actualParsed) {
		expected, _ := json.MarshalIndent(expectedParsed, "", "  ")
		actual, _ := json.MarshalIndent(actualParsed, "", "  ")

		diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(expected)),
			B:        difflib.SplitLines(string(actual)),
			FromFile: "Expected",
			FromDate: "",
			ToFile:   "Actual",
			ToDate:   "",
			Context:  1,
		})
		return fmt.Errorf("actual does not match expected, diff:\n%s\n", diff)
	}

	return nil
}

// Expand replaces ${var} or $var in the string based on saved Variables in the session/test scenario.
func (s *TestScenario) Expand(value string, skippedVars []string) (result string, rerr error) {
	session := s.Session()
	return os.Expand(value, func(name string) string {
		if contains(skippedVars, name) {
			return "$" + name
		}

		arrayResponse := strings.HasPrefix(name, "response[")
		if strings.HasPrefix(name, "response.") || arrayResponse {

			selector := "." + name
			query, err := gojq.Parse(selector)
			if err != nil {
				rerr = err
				return ""
			}

			j, err := session.RespJson()
			if err != nil {
				rerr = err
				return ""
			}

			j = map[string]interface{}{
				"response": j,
			}

			iter := query.Run(j)
			if next, found := iter.Next(); found {
				switch next := next.(type) {
				case string:
					return next
				case float32:
				case float64:
					// handle int64 returned as float in json
					return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%f", next), "0"), ".")
				case nil:
					rerr = fmt.Errorf("field ${%s} not found in json response:\n%s\n", name, string(session.RespBytes))
					return ""
				case error:
					rerr = fmt.Errorf("failed to evaluate selection: %s: %v", name, next)
					return ""
				default:
					return fmt.Sprintf("%s", next)
				}
			} else {
				rerr = fmt.Errorf("field ${%s} not found in json response:\n%s\n", name, string(session.RespBytes))
				return ""
			}
		}
		value, found := s.Variables[name]
		if !found {
			return ""
		}
		return fmt.Sprint(value)
	}), rerr
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// TestSession holds the http context for a user kinda like a browser.  Each scenario
// had a different session even if using the same user.
type TestSession struct {
	TestUser          *TestUser
	Client            *http.Client
	Resp              *http.Response
	Ctx               context.Context
	RespBytes         []byte
	respJson          interface{}
	Header            http.Header
	EventStream       bool
	EventStreamEvents chan interface{}
	Debug             bool
}

// RespJson returns the last http response body as json
func (s *TestSession) RespJson() (interface{}, error) {
	if s.respJson == nil {
		if err := json.Unmarshal(s.RespBytes, &s.respJson); err != nil {
			return nil, fmt.Errorf("error parsing json response: %v\nbody: %s\n", err, string(s.RespBytes))
		}

		if s.Debug {
			fmt.Println("response json:")
			e := json.NewEncoder(os.Stdout)
			e.SetIndent("", "  ")
			_ = e.Encode(s.respJson)
			fmt.Println("")
		}
	}
	return s.respJson, nil
}

func (s *TestSession) SetRespBytes(bytes []byte) {
	s.RespBytes = bytes
	s.respJson = nil
}

// StepModules is the list of functions used to add steps to a godog.ScenarioContext, you can
// add more to this list if you need test TestSuite specific steps.
var StepModules []func(ctx *godog.ScenarioContext, s *TestScenario)

func (suite *TestSuite) InitializeScenario(ctx *godog.ScenarioContext) {
	s := &TestScenario{
		Suite:     suite,
		sessions:  map[string]*TestSession{},
		Variables: map[string]interface{}{},
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
func TestMain(helper *test.Helper) int {
	s := &TestSuite{
		Helper:    helper,
		ApiURL:    "http://localhost:8000",
		users:     map[string]*TestUser{},
		nextOrgId: 20000000,
	}

	var allow *quota_management.QuotaManagementListConfig
	helper.Env.MustResolveAll(&allow)

	// Generate lots of org id's that scenarios can use to avoid
	// conflicting with each other..
	if allow != nil {
		for i := 0; i < 1000; i++ {
			allow.QuotaList.Organisations = append(allow.QuotaList.Organisations, quota_management.Organisation{
				Id:                  fmt.Sprintf("%d", 20000000+i),
				AnyUser:             true,
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

	return godog.TestSuite{
		Name:                "godogs",
		ScenarioInitializer: s.InitializeScenario,
		Options:             &opts,
	}.Run()
}
