package workers

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/onsi/gomega"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/gorm"
)

func TestLeaderElectionManager_acquireLeaderLease(t *testing.T) {
	type args struct {
		workerId   string
		workerType string
		dbConn     *gorm.DB
	}
	type errorCheck struct {
		wantErr     bool
		expectedErr string
	}

	tests := []struct {
		name       string
		args       args
		wantFn     func(*leaderLeaseAcquisition) error
		errorCheck errorCheck
		setupFn    func()
	}{
		{
			name: "failure listing any lease record in lease table",
			args: args{
				workerId:   "000-000",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: nil,
			errorCheck: errorCheck{
				wantErr:     true,
				expectedErr: "expected to find a lease entry",
			},
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "failure listing leases table results in error",
			args: args{
				workerId:   "000-001",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: nil,
			errorCheck: errorCheck{
				wantErr: true,
			},
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithQueryException()
			},
		},
		{
			name: "successfully acquired lease and elected  as the leader",
			args: args{
				workerId:   "000-002",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: func(acquisition *leaderLeaseAcquisition) error {
				if !acquisition.acquired {
					return errors.New("expected lease acquisition succeeded.")
				}
				if acquisition.currentLease.Leader != "000-002" {
					return errors.New("unexpected leader acquired.")
				}
				return nil
			},
			errorCheck: errorCheck{
				wantErr: false,
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "000-002",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT * FROM leader_leases").WithReply([]map[string]interface{}{mockEntry})
			},
		},
		{
			name: "valid lease within a minute of expiry extends lease",
			args: args{
				workerId:   "000-003",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: func(acquisition *leaderLeaseAcquisition) error {
				if !acquisition.acquired {
					return errors.New("expected lease acquisition succeeded.")
				}
				if !acquisition.currentLease.Expires.After(time.Now().Add(time.Second * 30)) {
					return errors.New("expected lease to be extended")
				}
				return nil
			},
			errorCheck: errorCheck{
				wantErr: false,
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "000-003",
					"id":      "leader-lease-id",
					"expires": time.Now().Add(time.Second * 30),
				}
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1 FOR UPDATE`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().
					WithQuery(`UPDATE "leader_leases" SET "expires"=$1,"leader"=$2,"updated_at"=$3 WHERE "leader_leases"."deleted_at" IS NULL AND "id" = $4`)
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
		{
			name: "valid lease for another worker not to be acquired",
			args: args{
				workerId:   "000-004",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn: func(acquisition *leaderLeaseAcquisition) error {
				if acquisition.acquired {
					return errors.New("expected lease acquisition to not be acquired")
				}
				if acquisition.currentLease.Leader != "otherLeader" {
					return errors.New("unexpected lease leader: " + acquisition.currentLease.Leader)
				}
				return nil
			},
			errorCheck: errorCheck{
				wantErr: false,
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "otherLeader",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().NewMock().
					WithQuery(`SELECT * FROM leader_leases where deleted_at is null and lease_type = $1`).
					WithArgs("cluster").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithQueryException().WithExecException()
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			s := &LeaderElectionManager{leaderLeaseExpirationTime: 3 * time.Minute}
			got, err := s.acquireLeaderLease(tt.args.workerId, tt.args.workerType, tt.args.dbConn)
			if (err != nil) != tt.errorCheck.wantErr {
				t.Errorf("acquireLeaderLease() error = %v, wantErr %v", err, tt.errorCheck.wantErr)
				return
			}
			if err != nil && !strings.Contains(err.Error(), tt.errorCheck.expectedErr) {
				t.Errorf("Expecting error '%s' but got '%s'", tt.errorCheck.expectedErr, err.Error())
			}
			if tt.wantFn == nil {
				return
			}
			if err := tt.wantFn(got); err != nil {
				t.Errorf("acquireLeaderLease() got = %v, want %v", got, err.Error())
			}
		})
	}
}

func TestNewLeaderElectionManager(t *testing.T) {
	type args struct {
		workers           []Worker
		connectionFactory *db.ConnectionFactory
		reconcilerConfig  *ReconcilerConfig
	}
	tests := []struct {
		name string
		args args
		want *LeaderElectionManager
	}{
		{
			name: "should return a New Leader Election Manager",
			args: args{
				workers:           []Worker{},
				connectionFactory: &db.ConnectionFactory{},
				reconcilerConfig: &ReconcilerConfig{
					ReconcilerRepeatInterval:               30 * time.Second,
					LeaderElectionReconcilerRepeatInterval: 15 * time.Second,
					LeaderLeaseExpirationTime:              1 * time.Minute,
				},
			},
			want: &LeaderElectionManager{
				workers:                                []Worker{},
				connectionFactory:                      &db.ConnectionFactory{},
				tearDown:                               nil,
				leaderElectionReconcilerRepeatInterval: 15 * time.Second,
				leaderLeaseExpirationTime:              1 * time.Minute,
				workerGrp:                              sync.WaitGroup{},
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			g.Expect(NewLeaderElectionManager(tt.args.workers, tt.args.connectionFactory, tt.args.reconcilerConfig)).To(gomega.Equal(tt.want))
		})
	}
}

func TestLeaderElectionManager_start(t *testing.T) {
	type fields struct {
		workers                                []Worker
		connectionFactory                      *db.ConnectionFactory
		tearDown                               chan struct{}
		leaderElectionReconcilerRepeatInterval time.Duration
		workerGrp                              *sync.WaitGroup
	}
	var wg sync.WaitGroup
	tests := []struct {
		name    string
		fields  fields
		setupFn func()
	}{
		{
			name: "should start the worker if the worker is a leader and is not running",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "01"
						},
						GetWorkerTypeFunc: func() string {
							return "cluster"
						},
						IsRunningFunc: func() bool {
							return false
						},
						GetSyncGroupFunc: func() *sync.WaitGroup {
							return &sync.WaitGroup{}
						},
						StartFunc: func() {},
						StopFunc:  nil, // should never be called
						HasTerminatedFunc: func() bool {
							return false
						},
					},
				},
				connectionFactory:                      db.NewMockConnectionFactory(nil),
				leaderElectionReconcilerRepeatInterval: 15 * time.Second,
				workerGrp:                              &sync.WaitGroup{},
				tearDown:                               make(chan struct{}),
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "01",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery("SELECT * FROM leader_leases").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should stop worker if the worker is not leader and it is running",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "01"
						},
						GetWorkerTypeFunc: func() string {
							return "cluster"
						},
						IsRunningFunc: func() bool {
							return true
						},
						GetSyncGroupFunc: func() *sync.WaitGroup {
							return &wg
						},
						StartFunc: nil, // should never be called
						StopFunc:  func() {},
						HasTerminatedFunc: func() bool {
							return false
						},
					},
				},
				connectionFactory:                      db.NewMockConnectionFactory(nil),
				leaderElectionReconcilerRepeatInterval: 15 * time.Second,
				workerGrp:                              &sync.WaitGroup{},
				tearDown:                               make(chan struct{}),
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "02",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery("SELECT * FROM leader_leases").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should neither stop not start worker if the worker is leader and it is running",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "01"
						},
						GetWorkerTypeFunc: func() string {
							return "cluster"
						},
						IsRunningFunc: func() bool {
							return true
						},
						GetSyncGroupFunc: func() *sync.WaitGroup {
							return &sync.WaitGroup{}
						},
						StartFunc: nil, // should never be called
						StopFunc:  nil, // should never be called
						HasTerminatedFunc: func() bool {
							return false
						},
					},
				},
				connectionFactory:                      db.NewMockConnectionFactory(nil),
				leaderElectionReconcilerRepeatInterval: 15 * time.Second,
				workerGrp:                              &sync.WaitGroup{},
				tearDown:                               make(chan struct{}),
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "01",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery("SELECT * FROM leader_leases").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn()

			workerMock := tt.fields.workers[0].(*WorkerMock)
			if workerMock.StopFunc != nil {
				wg.Add(1) // assume that there is one worker group running that we is going to be stopped
			}

			s := &LeaderElectionManager{
				workers:                                tt.fields.workers,
				connectionFactory:                      tt.fields.connectionFactory,
				tearDown:                               tt.fields.tearDown,
				leaderElectionReconcilerRepeatInterval: tt.fields.leaderElectionReconcilerRepeatInterval,
				workerGrp:                              *tt.fields.workers[0].GetSyncGroup(),
			}

			s.Start()

			g.Expect(workerMock.calls.GetWorkerType).To(gomega.HaveLen(1))
			if workerMock.StartFunc != nil {
				g.Expect(workerMock.calls.Start).To(gomega.HaveLen(1))
				g.Expect(workerMock.calls.GetID).To(gomega.HaveLen(2)) // it is called 2 times when starting the worker
			}

			if workerMock.StopFunc != nil {
				g.Expect(workerMock.calls.Stop).To(gomega.HaveLen(1))
				g.Expect(workerMock.calls.GetID).To(gomega.HaveLen(3)) // it is called three times when stopping the worker
			}

			if workerMock.StopFunc == nil && workerMock.StartFunc == nil {
				g.Expect(workerMock.calls.GetID).To(gomega.HaveLen(1)) // it should be called once when the worker is a leader and it is running
				g.Expect(workerMock.calls.Stop).To(gomega.HaveLen(0))  // it should never be called when the worker is a leader and it is running
				g.Expect(workerMock.calls.Start).To(gomega.HaveLen(0)) // it should never be called when the worker is a leader and it is running
			}

		})
	}
}

func TestLeaderElectionManager_Stop(t *testing.T) {
	type fields struct {
		workers                                []Worker
		connectionFactory                      *db.ConnectionFactory
		tearDown                               chan struct{}
		leaderElectionReconcilerRepeatInterval time.Duration
		workerGrp                              *sync.WaitGroup
	}
	var wg sync.WaitGroup
	wg.Add(1)
	tests := []struct {
		name    string
		fields  fields
		setupFn func()
	}{
		{
			name: "should return if teardown is nil",
			fields: fields{
				tearDown: nil,
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "02"
						},
						GetWorkerTypeFunc: func() string {
							return "leader"
						},
						IsRunningFunc: func() bool {
							return true
						},
						GetSyncGroupFunc: func() *sync.WaitGroup {
							return &sync.WaitGroup{}
						},
						StopFunc: func() {
						},
						HasTerminatedFunc: func() bool {
							return false
						},
					},
				},
				connectionFactory:                      nil,
				leaderElectionReconcilerRepeatInterval: 15 * time.Second,
				workerGrp:                              &sync.WaitGroup{},
			},
			setupFn: nil,
		},
		{
			name: "should stop the worker",
			fields: fields{
				tearDown:  make(chan struct{}),
				workerGrp: &wg,
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "02"
						},
						GetWorkerTypeFunc: func() string {
							return "leader"
						},
						IsRunningFunc: func() bool {
							return true
						},
						GetSyncGroupFunc: func() *sync.WaitGroup {
							return &wg
						},
						StopFunc: func() {
							wg.Done()
						},
						HasTerminatedFunc: func() bool {
							return false
						},
					},
				},
				leaderElectionReconcilerRepeatInterval: 15 * time.Second,
				connectionFactory:                      db.NewMockConnectionFactory(nil),
			},
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "02",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery("SELECT * FROM leader_leases").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			if tt.setupFn != nil {
				tt.setupFn()
			}
			s := &LeaderElectionManager{
				workers:                                tt.fields.workers,
				connectionFactory:                      tt.fields.connectionFactory,
				tearDown:                               tt.fields.tearDown,
				leaderElectionReconcilerRepeatInterval: tt.fields.leaderElectionReconcilerRepeatInterval,
				workerGrp:                              *tt.fields.workers[0].GetSyncGroup(),
			}
			if s.tearDown != nil {
				s.Start()
			}
			s.Stop()
			workerMock := tt.fields.workers[0].(*WorkerMock)

			if s.tearDown != nil {
				g.Expect(workerMock.calls.Stop).To(gomega.HaveLen(1))
			} else {
				g.Expect(workerMock.calls.Stop).To(gomega.HaveLen(0))
			}

		})
	}
}

func TestLeaderElectionManager_isWorkerLeader(t *testing.T) {
	type fields struct {
		workers           []Worker
		connectionFactory *db.ConnectionFactory
	}
	type args struct {
		worker Worker
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		setupFn func()
	}{
		{
			name: "should return true if the worker is a leader",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "01"
						},
						GetWorkerTypeFunc: func() string {
							return "cluster"
						},
						HasTerminatedFunc: func() bool {
							return false
						},
					},
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				worker: &WorkerMock{
					GetIDFunc: func() string {
						return "01"
					},
					GetWorkerTypeFunc: func() string {
						return "leader"
					},
					HasTerminatedFunc: func() bool {
						return false
					},
				},
			},
			want: true,
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "01",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery("SELECT * FROM leader_leases").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should return false if the worker is not leader",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "01"
						},
						GetWorkerTypeFunc: func() string {
							return "leader"
						},
					},
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				worker: &WorkerMock{
					GetIDFunc: func() string {
						return "02"
					},
					GetWorkerTypeFunc: func() string {
						return "cluster"
					},
				},
			},
			want: false,
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "01",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().
					NewMock().
					WithQuery("SELECT * FROM leader_leases").
					WithReply([]map[string]interface{}{mockEntry})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
		{
			name: "should return false if an error is returned when trying to check if it is a leader",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "01"
						},
						GetWorkerTypeFunc: func() string {
							return "leader"
						},
					},
				},
				connectionFactory: db.NewMockConnectionFactory(nil),
			},
			args: args{
				worker: &WorkerMock{
					GetIDFunc: func() string {
						return "02"
					},
					GetWorkerTypeFunc: func() string {
						return "cluster"
					},
				},
			},
			want: false,
			setupFn: func() {
				mocket.Catcher.Reset().
					NewMock().
					WithQuery("SELECT * FROM leader_leases").
					WithReply([]map[string]interface{}{})
				mocket.Catcher.NewMock().WithExecException().WithQueryException()
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			tt.setupFn()
			s := &LeaderElectionManager{
				workers:           tt.fields.workers,
				connectionFactory: tt.fields.connectionFactory,
			}
			g.Expect(s.isWorkerLeader(tt.args.worker)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_HasTerminated(t *testing.T) {
	type fields struct {
		workers     []Worker
		workerCount int
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "should not stop normal worker",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "02"
						},
						GetWorkerTypeFunc: func() string {
							return "leader"
						},
						IsRunningFunc: func() bool {
							return true
						},
						GetSyncGroupFunc: func() *sync.WaitGroup {
							return &sync.WaitGroup{}
						},
						StopFunc: func() {
						},
						HasTerminatedFunc: func() bool {
							return false
						},
					},
				},
				workerCount: 1,
			},
		},
		{
			name: "should stop terminated worker",
			fields: fields{
				workers: []Worker{
					&WorkerMock{
						GetIDFunc: func() string {
							return "02"
						},
						GetWorkerTypeFunc: func() string {
							return "leader"
						},
						IsRunningFunc: func() bool {
							return true
						},
						GetSyncGroupFunc: func() *sync.WaitGroup {
							return &sync.WaitGroup{}
						},
						StopFunc: func() {
						},
						HasTerminatedFunc: func() bool {
							return true
						},
					},
				},
				workerCount: 0,
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()

			mockEntry := map[string]interface{}{
				"leader":  "02",
				"expires": time.Now().Add(time.Hour),
			}
			mocket.Catcher.Reset().
				NewMock().
				WithQuery("SELECT * FROM leader_leases").
				WithReply([]map[string]interface{}{mockEntry})
			mocket.Catcher.NewMock().WithExecException().WithQueryException()

			s := &LeaderElectionManager{
				workers:                                tt.fields.workers,
				connectionFactory:                      db.NewMockConnectionFactory(nil),
				tearDown:                               make(chan struct{}),
				leaderElectionReconcilerRepeatInterval: 15 * time.Second,
				workerGrp:                              sync.WaitGroup{},
			}
			s.workerGrp.Add(1)

			// start mock worker
			s.Start()

			// has worker been stopped or not as expected
			workerMock := tt.fields.workers[0].(*WorkerMock)
			g.Expect(workerMock.calls.Stop).To(gomega.HaveLen(1 - tt.fields.workerCount))

			// has worker been removed or not as expected
			g.Expect(len(s.workers)).To(gomega.Equal(tt.fields.workerCount))
		})
	}
}
