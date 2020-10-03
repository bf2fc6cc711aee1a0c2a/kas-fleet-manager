package integration

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"

	"gitlab.cee.redhat.com/service/managed-services-api/test"
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
