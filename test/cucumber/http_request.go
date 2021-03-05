// Setting a path prefixed to subsequent http requests:
//    Given the path prefix is "/api/managed-services-api"
// Send an http request. Supports (GET|POST|PUT|DELETE|PATCH):
//    When I GET path "/v1/some/${kid}
// Send an http request with a body. Supports (GET|POST|PUT|DELETE|PATCH):
//    When I POST path "/v1/some/${kid}" with json body:
//      """
//      {"some":"${kid}"}
//      """
// Wait until an http get responds with an expected result or a timeout occurs:
//    Given I wait up to "35.5" seconds for a GET on path "/v1/some/path" response ".total" selection to match "1"
package cucumber

import (
	"bytes"
	"context"
	fmt "fmt"
	"github.com/cucumber/godog"
	"io/ioutil"
	"net/http"
	"time"
)

func init() {
	StepModules = append(StepModules, func(ctx *godog.ScenarioContext, s *TestScenario) {
		ctx.Step(`^the path prefix is "([^"]*)"$`, s.theApiPrefixIs)
		ctx.Step(`^I (GET|POST|PUT|DELETE|PATCH) path "([^"]*)"$`, s.sendHttpRequest)
		ctx.Step(`^I (GET|POST|PUT|DELETE|PATCH) path "([^"]*)" with json body:$`, s.sendHttpRequestWithJsonBody)
		ctx.Step(`^I wait up to "([^"]*)" seconds for a GET on path "([^"]*)" response "([^"]*)" selection to match "([^"]*)"$`, s.iWaitUpToSecondsForAGETOnPathResponseSelectionToMatch)
	})
}

func (s *TestScenario) theApiPrefixIs(prefix string) error {
	s.PathPrefix = prefix
	return nil
}

func (s *TestScenario) sendHttpRequest(method, path string) error {
	return s.sendHttpRequestWithJsonBody(method, path, nil)
}

func (s *TestScenario) sendHttpRequestWithJsonBody(method, path string, jsonTxt *godog.DocString) (err error) {
	// handle panic
	defer func() {
		switch t := recover().(type) {
		case string:
			err = fmt.Errorf(t)
		case error:
			err = t
		}
	}()

	session := s.Session()

	body := &bytes.Buffer{}
	if jsonTxt != nil {
		body.WriteString(s.Expand(jsonTxt.Content))
	}
	fullUrl := s.Suite.ApiURL + s.PathPrefix + s.Expand(path)

	session.Resp = nil
	session.RespBytes = nil
	session.respJson = nil

	ctx := session.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullUrl, body)
	if err != nil {
		return err
	}

	if session.AuthorizationHeader != "" {
		req.Header.Set("Authorization", session.AuthorizationHeader)
	} else if session.TestUser.Token != "" {
		req.Header.Set("Authorization", "Bearer "+session.TestUser.Token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := session.Client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	session.Resp = resp
	session.RespBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (s *TestScenario) iWaitUpToSecondsForAGETOnPathResponseSelectionToMatch(timeout float64, path string, selection, expected string) error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout*float64(time.Second)))
	defer cancel()
	session := s.Session()
	session.Ctx = ctx
	defer func() {
		session.Ctx = nil
	}()

	for {
		err := s.sendHttpRequest("GET", path)
		if err == nil {
			err = s.theSelectionFromTheResponseShouldMatch(selection, expected)
			if err == nil {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(time.Duration(timeout * float64(time.Second) / 10.0))
		}
	}
}
