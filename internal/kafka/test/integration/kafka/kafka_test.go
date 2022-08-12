package kafka

import (
	"flag"
	ktest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/cucumber"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/cucumber/godog"
	"github.com/onsi/gomega"
	"github.com/spf13/pflag"
	"os"
	"testing"
)

var helper *test.Helper
var teardown func()
var opts = cucumber.DefaultOptions()

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

func TestMain(m *testing.M) {
	for _, arg := range os.Args[1:] {
		if arg == "-test.v=true" || arg == "-test.v" || arg == "-v" { // go test transforms -v option
			opts.Format = "pretty"
		}
	}

	flag.Parse()

	pflag.Parse()
	opts.Paths = pflag.Args()

	// Startup all the services and mocks that are needed to test the
	// kafka features.
	t := &testing.T{}

	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	helper, _, teardown = ktest.NewKafkaHelper(t, ocmServer)
	defer teardown()

	os.Exit(m.Run())
}

func TestFeatures(t *testing.T) {
	g := gomega.NewWithT(t)

	o := opts
	o.TestingT = t

	s := cucumber.NewTestSuite(helper)

	status := godog.TestSuite{
		Name:                "kafka",
		Options:             &o,
		ScenarioInitializer: s.InitializeScenario,
	}.Run()

	g.Expect(status).To(gomega.Equal(0))
}
