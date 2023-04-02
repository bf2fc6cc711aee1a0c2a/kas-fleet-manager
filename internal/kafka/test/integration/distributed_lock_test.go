package integration

import (
	kafkatest "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	sync2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/sync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/onsi/gomega"
	"sync"
	"testing"
	"time"
)

func TestDistributedLock_ConcurrentAccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, _, teardown := kafkatest.NewKafkaHelper(t, ocmServer)
	defer teardown()

	type fields struct {
		lockName1 string
		lockName2 string
	}
	tests := []struct {
		name           string
		fields         fields
		expectedValue1 int
		expectedValue2 int
	}{
		{
			name: "should block concurrent access to the same lock",
			fields: fields{
				lockName1: "test-lock",
				lockName2: "test-lock",
			},
			expectedValue1: 1,
			expectedValue2: 2,
		},
		{
			name: "should not block concurrent access to the different locks",
			fields: fields{
				lockName1: "test-lock",
				lockName2: "test-lock1",
			},
			expectedValue1: 2,
			expectedValue2: 1,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			dbConn := h.DBFactory().New()

			dl1 := sync2.NewDistributedLock(dbConn, tt.fields.lockName1)
			dl2 := sync2.NewDistributedLock(dbConn, tt.fields.lockName2)

			counter := 0
			goRoutineValue := 0
			mainValue := 0

			ch := make(chan bool, 1)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				err := dl1.Lock()
				ch <- true
				g.Expect(err).NotTo(gomega.HaveOccurred())

				time.Sleep(100 * time.Millisecond)

				counter++
				goRoutineValue = counter

				err = dl1.Unlock()
				g.Expect(err).NotTo(gomega.HaveOccurred())
				wg.Done()
			}()

			<-ch // wait for the routine to create the lock

			err := dl2.Lock()
			g.Expect(err).NotTo(gomega.HaveOccurred())

			counter++
			mainValue = counter

			err = dl2.Unlock()
			wg.Wait()
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(mainValue).To(gomega.Equal(tt.expectedValue2))
			g.Expect(goRoutineValue).To(gomega.Equal(tt.expectedValue1))
		})
	}
}
