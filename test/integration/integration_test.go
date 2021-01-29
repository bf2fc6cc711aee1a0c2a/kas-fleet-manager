package integration

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
)

func TestMain(m *testing.M) {
	flag.Parse()
	glog.V(10).Infof("Starting integration test using go version %s", runtime.Version())
	helper := test.NewHelper(&testing.T{}, nil)
	helper.ResetDB()
	exitCode := m.Run()
	helper.Teardown()
	os.Exit(exitCode)
}
