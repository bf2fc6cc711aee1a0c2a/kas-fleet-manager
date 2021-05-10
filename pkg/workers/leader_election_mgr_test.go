package workers

import (
	"errors"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	mocket "github.com/selvatico/go-mocket"
	"gorm.io/gorm"
)

func TestLeaderElectionManager_acquireLeaderLease(t *testing.T) {
	type args struct {
		workerId   string
		workerType string
		dbConn     *gorm.DB
	}
	tests := []struct {
		name    string
		args    args
		wantFn  func(*leaderLeaseAcquisition) error
		wantErr bool
		setupFn func()
	}{
		{
			name: "failure listing any lease record in lease table",
			args: args{
				workerId:   "000-000",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn:  nil,
			wantErr: true,
			setupFn: func() {
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply([]map[string]interface{}{})
			},
		},
		{
			name: "failure listing leases table results in error",
			args: args{
				workerId:   "000-001",
				workerType: "cluster",
				dbConn:     db.NewMockConnectionFactory(nil).DB,
			},
			wantFn:  nil,
			wantErr: true,
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
			wantErr: false,
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "000-002",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply([]map[string]interface{}{mockEntry})
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
			wantErr: false,
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "000-003",
					"id":      "leader-lease-id",
					"expires": time.Now().Add(time.Second * 30),
				}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply([]map[string]interface{}{mockEntry})
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
			wantErr: false,
			setupFn: func() {
				mockEntry := map[string]interface{}{
					"leader":  "otherLeader",
					"expires": time.Now().Add(time.Hour),
				}
				mocket.Catcher.Reset().NewMock().WithQuery("SELECT").WithReply([]map[string]interface{}{mockEntry})
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFn()
			s := &LeaderElectionManager{leaseRenewTime: 3 * time.Minute}
			got, err := s.acquireLeaderLease(tt.args.workerId, tt.args.workerType, tt.args.dbConn)
			if (err != nil) != tt.wantErr {
				t.Errorf("acquireLeaderLease() error = %v, wantErr %v", err, tt.wantErr)
				return
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
