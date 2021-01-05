package observatorium

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/golang/glog"
	pAPI "github.com/prometheus/client_golang/api"
	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"
	"net/http"
	"strings"
	"time"
)

type ClientConfiguration struct {
	BaseURL    string
	Timeout    time.Duration
	AuthToken  string
	Cookie     string
	Debug      bool
	EnableMock bool
	Insecure   bool
}

type Client struct {
	// Configuration
	Config     *ClientConfiguration
	connection pV1.API
	Service    APIObservatoriumService
}

func NewClient(config *Configuration) (*Client, error) {
	client := &Client{
		Config: &ClientConfiguration{
			Timeout:    config.Timeout,
			AuthToken:  config.AuthToken,
			Cookie:     config.Cookie,
			Debug:      config.Debug,
			EnableMock: false,
			Insecure:   config.Insecure,
		},
	}

	// Ensure baseURL has a trailing slash
	baseURL := strings.TrimSuffix(config.BaseURL, "/")
	client.Config.BaseURL = baseURL + "/"
	client.Service = &ServiceObservatorium{client: client}
	if config.Insecure {
		pAPI.DefaultRoundTripper.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	apiClient, err := pAPI.NewClient(
		pAPI.Config{
			Address: client.Config.BaseURL,
			RoundTripper: authRoundTripper{
				config:  *client.Config,
				wrapped: pAPI.DefaultRoundTripper,
			},
		},
	)
	if err != nil {
		return nil, err
	}
	client.connection = pV1.NewAPI(apiClient)
	client.Service = &ServiceObservatorium{client: client}
	return client, nil

}

func NewClientMock(config *Configuration) (*Client, error) {
	client := &Client{
		Config: &ClientConfiguration{
			Timeout:    config.Timeout,
			AuthToken:  config.AuthToken,
			Debug:      false,
			EnableMock: true,
			Insecure:   config.Insecure,
		},
	}

	client.Service = &ServiceObservatorium{client: client}
	client.connection = client.MockAPI()

	return client, nil
}

type authRoundTripper struct {
	config  ClientConfiguration
	wrapped http.RoundTripper
}

func (p authRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if p.config.AuthToken != "" {
		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.AuthToken))
	} else if p.config.Cookie != "" {
		request.Header.Add("Cookie", p.config.Cookie)
	} else {
		return nil, fmt.Errorf("can't request metrics without auth")
	}
	return p.wrapped.RoundTrip(request)
}

// Send a POST request to server.
func (c *Client) send(query string) (pModel.Value, pV1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.Timeout)
	defer cancel()
	return c.connection.Query(ctx, query, time.Now())
}

// Send a POST request to server.
func (c *Client) sendRange(query string, bounds pV1.Range) (pModel.Value, pV1.Warnings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Config.Timeout)
	defer cancel()
	return c.connection.QueryRange(ctx, query, bounds)

}

// Query sends a metrics request to server and returns unmashalled Vector response.
// The VectorResult(s) inside will contain either .Value for queries resulting in instant vector,
// or .Values for queries resulting in a range vector.
//
// queryTemplate must contain one %s for labels, e.g. `some_metric{%s}` or `count(some_metric{state="up",%s})`.
// (Undocumented PromQL: empty labels some_metric{} are OK - https://github.com/prometheus/prometheus/issues/3697)
// labels 0 or more constraints separated by comma e.g. `` or `foo="bar",quux="baz"`.
func (c *Client) Query(queryTemplate string, label string) (*pModel.Vector, error) {

	queryString := fmt.Sprintf(queryTemplate, label)
	glog.Infof("prometheus query = %+v", queryString)
	values, warnings, err := c.send(queryString)

	if len(warnings) > 0 {
		glog.Warningf("Prometheus client got warnings %s", all(warnings, "and"))
	}
	if err != nil {
		return nil, err
	}

	v, ok := values.(pModel.Vector)
	if !ok {
		glog.Errorf("Prometheus client got data of type %T, but expected model.Vector", values)
		return nil, fmt.Errorf("Prometheus client got data of type %T, but expected model.Vector", values)
	}
	return &v, err
}
func (c *Client) QueryRange(queryTemplate string, label string, bounds pV1.Range) Metric {

	queryString := fmt.Sprintf(queryTemplate, label)
	values, warnings, err := c.sendRange(queryString, bounds)
	if len(warnings) > 0 {
		glog.Warningf("Prometheus client got warnings %s", all(warnings, "and"))
	}
	if err != nil {
		return Metric{Err: err}
	}

	m, ok := values.(pModel.Matrix)
	if !ok {
		glog.Errorf("Prometheus client got data of type %T, but expected model.Matrix", values)
		return Metric{Err: fmt.Errorf("Prometheus client got data of type %T, but expected model.Matrix", values)}

	}
	return Metric{Matrix: m}
}

func all(items []string, conjunction string) string {
	count := len(items)
	if count == 0 {
		return ""
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("'%s'", item)
	}
	if count == 1 {
		return quoted[0]
	}
	head := quoted[0 : count-1]
	tail := quoted[count-1]
	return fmt.Sprintf("%s %s %s", strings.Join(head, ", "), conjunction, tail)
}
