package signalbus

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
)

func notifyAfter(bus *signalBus, name string, d time.Duration) {
	go func() {
		time.Sleep(d)
		bus.Notify(name)
	}()
}

func TestNewSignalBus(t *testing.T) {
	g := gomega.NewWithT(t)

	bus := NewSignalBus().(*signalBus)

	// it's ok to send notifications before subscriptions...
	bus.Notify("unknown")

	aSub1 := bus.Subscribe("a")

	// in about a second the subscription should get signaled..
	notifyAfter(bus, "a", 1*time.Second)
	g.Expect(aSub1.IsSignaled()).Should(gomega.Equal(false))
	g.Eventually(aSub1.IsSignaled, 2*time.Second).Should(gomega.Equal(true))

	// Verify that the same subs share memory structs...
	g.Expect(len(bus.signals)).Should(gomega.Equal(1))
	aSub2 := bus.Subscribe("a")
	g.Expect(len(bus.signals)).Should(gomega.Equal(1))

	// Verify that notifications work on both subs...
	notifyAfter(bus, "a", 1*time.Second)
	g.Expect(aSub1.IsSignaled()).Should(gomega.Equal(false))
	g.Expect(aSub2.IsSignaled()).Should(gomega.Equal(false))
	g.Eventually(aSub1.IsSignaled, 2*time.Second).Should(gomega.Equal(true))
	g.Eventually(aSub2.IsSignaled, 2*time.Second).Should(gomega.Equal(true))

	// Closing all the subs to the same named signal will release memory..
	aSub1.Close()
	g.Expect(len(bus.signals)).Should(gomega.Equal(1))
	aSub2.Close()
	g.Expect(len(bus.signals)).Should(gomega.Equal(0))

}
