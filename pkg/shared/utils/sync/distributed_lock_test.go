package sync

import (
	"database/sql/driver"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/onsi/gomega"
	mocket "github.com/selvatico/go-mocket"
	"sync"
	"testing"
	"time"
)

func TestDistributedLock_LockUnlock(t *testing.T) {
	g := gomega.NewWithT(t)

	dbConn := db.NewMockConnectionFactory(nil).New()
	dl := NewDistributedLock(dbConn, "test-lock")

	g.Expect(dl.Lock()).To(gomega.Succeed())

	g.Expect(dl.Unlock()).To(gomega.Succeed())
}

func TestDistributedLock_UnlockWithoutLock(t *testing.T) {
	g := gomega.NewWithT(t)

	dbConn := db.NewMockConnectionFactory(nil).New()
	dl := NewDistributedLock(dbConn, "test-lock")

	g.Expect(dl.Unlock()).ToNot(gomega.Succeed())
}

func TestDistributedLock_ConcurrentAccess(t *testing.T) {
	type fields struct {
		lockName1 string
		lockName2 string
	}
	tests := []struct {
		name           string
		fields         fields
		setupFn        func()
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
			setupFn: func() {
				locks := map[string]*sync.Mutex{}
				mocket.Catcher.Reset().NewMock().WithCallback(func(s string, values []driver.NamedValue) {
					mutex, ok := locks[values[0].Value.(string)]
					if !ok {
						var mu sync.Mutex
						locks[values[0].Value.(string)] = &mu
						mutex = &mu
					}
					mutex.Lock()
				}).WithQuery("INSERT INTO distributed_locks (ID) VALUES ($1) ON CONFLICT DO NOTHING")

				mocket.Catcher.NewMock().
					WithCallback(func(s string, values []driver.NamedValue) {
						mutex, ok := locks[values[0].Value.(string)]
						if !ok {
							t.Fail()
						}
						mutex.Unlock()
					}).
					WithQuery("DELETE FROM distributed_locks where ID = $1")

				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should not block concurrent access to the different locks",
			fields: fields{
				lockName1: "test-lock",
				lockName2: "test-lock1",
			},
			expectedValue1: 2,
			expectedValue2: 1,
			setupFn: func() {
				locks := map[string]*sync.Mutex{}
				mocket.Catcher.Reset().NewMock().WithCallback(func(s string, values []driver.NamedValue) {
					mutex, ok := locks[values[0].Value.(string)]
					if !ok {
						var mu sync.Mutex
						locks[values[0].Value.(string)] = &mu
						mutex = &mu
					}
					mutex.Lock()
				}).WithQuery("INSERT INTO distributed_locks (ID) VALUES ($1) ON CONFLICT DO NOTHING")

				mocket.Catcher.NewMock().
					WithCallback(func(s string, values []driver.NamedValue) {
						mutex, ok := locks[values[0].Value.(string)]
						if !ok {
							t.Fail()
						}
						mutex.Unlock()
					}).
					WithQuery("DELETE FROM distributed_locks where ID = $1")

				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			if tt.setupFn != nil {
				tt.setupFn()
			}

			dbConn := db.NewMockConnectionFactory(nil).New()

			dl1 := newDistributedLockMock(dbConn, tt.fields.lockName1)
			dl2 := newDistributedLockMock(dbConn, tt.fields.lockName2)

			counter := 0
			goRoutineValue := 0
			mainValue := 0

			var wg sync.WaitGroup
			wg.Add(1)

			ch := make(chan bool, 1)

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
