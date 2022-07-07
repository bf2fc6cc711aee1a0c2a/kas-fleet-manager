package workers

import (
	"sync"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/onsi/gomega"
)

func TestBaseWorker_GetID(t *testing.T) {
	type fields struct {
		Id string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "should return the id of the base worker",
			fields: fields{
				Id: "base_worker_id",
			},
			want:    "base_worker_id",
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := &BaseWorker{
				Id: tt.fields.Id,
			}
			g.Expect(b.GetID()).To(gomega.Equal(tt.want))
		})
	}
}

func TestBaseWorker_GetWorkerType(t *testing.T) {
	type fields struct {
		WorkerType string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should return the base workers worker type",
			fields: fields{
				WorkerType: "leader",
			},
			want: "leader",
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := &BaseWorker{
				WorkerType: tt.fields.WorkerType,
			}
			g.Expect(b.GetWorkerType()).To(gomega.Equal(tt.want))
		})
	}
}

func TestBaseWorker_GetStopChan(t *testing.T) {
	type fields struct {
		imStop chan struct{}
	}
	var stopchan chan struct{}
	tests := []struct {
		name   string
		fields fields
		want   *chan struct{}
	}{
		{
			name: "should return the base workers stop chan",
			fields: fields{
				imStop: stopchan,
			},
			want: &stopchan,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := &BaseWorker{
				imStop: tt.fields.imStop,
			}
			g.Expect(b.GetStopChan()).To(gomega.Equal(tt.want))
		})
	}
}

func TestBaseWorker_GetSyncGroup(t *testing.T) {
	type fields struct {
		syncTeardown *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
		want   *sync.WaitGroup
	}{
		{
			name: "should return the base workers sync group",
			fields: fields{
				syncTeardown: &sync.WaitGroup{},
			},
			want: &sync.WaitGroup{},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := BaseWorker{
				syncTeardown: sync.WaitGroup{},
			}
			g.Expect(b.GetSyncGroup()).To(gomega.Equal(tt.want))
		})
	}
}

func TestBaseWorker_IsRunning(t *testing.T) {
	type fields struct {
		isRunning bool
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should return true if the worker is running",
			fields: fields{
				isRunning: true,
			},
			want: true,
		},
		{
			name: "should return false if the worker is not running",
			fields: fields{
				isRunning: false,
			},
			want: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := &BaseWorker{
				isRunning: tt.fields.isRunning,
			}
			g.Expect(b.IsRunning()).To(gomega.Equal(tt.want))
		})
	}
}

func TestBaseWorker_SetIsRunning(t *testing.T) {
	type fields struct {
		isRunning bool
	}
	type args struct {
		val bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "should set the variable isRunning",
			fields: fields{
				isRunning: false,
			},
			args: args{
				val: true,
			},
			want: true,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := &BaseWorker{
				isRunning: tt.fields.isRunning,
			}
			b.SetIsRunning(tt.args.val)
			g.Expect(b.IsRunning()).To(gomega.Equal(tt.want))
		})
	}
}

func TestBaseWorker_StartWorker(t *testing.T) {
	type fields struct {
		Id         string
		WorkerType string
		Reconciler Reconciler
		isRunning  bool
		imStop     chan struct{}
	}
	reconcileChan := make(chan time.Time, 1000)
	var wg sync.WaitGroup
	var stopchan chan struct{}

	type args struct {
		w Worker
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "should start the worker",
			fields: fields{
				Id:         "base_worker_id",
				WorkerType: "cluster",
				isRunning:  false,
				Reconciler: Reconciler{
					SignalBus:        signalbus.NewSignalBus(),
					ReconcilerConfig: NewReconcilerConfig(),
				},
				imStop: stopchan,
			},
			args: args{
				w: &WorkerMock{
					GetStopChanFunc: func() *chan struct{} {
						return &stopchan
					},
					GetSyncGroupFunc: func() *sync.WaitGroup {
						return &wg
					},
					SetIsRunningFunc: func(val bool) {
					},
					GetIDFunc: func() string {
						return "base_worker_id"
					},
					GetWorkerTypeFunc: func() string {
						return "cluster"
					},
					IsRunningFunc: func() bool {
						return false
					},
					ReconcileFunc: func() []error {
						var errors []error
						reconcileChan <- time.Now()
						return errors
					},
				},
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := &BaseWorker{
				Id:         tt.fields.Id,
				WorkerType: tt.fields.WorkerType,
				Reconciler: tt.fields.Reconciler,
				isRunning:  tt.fields.isRunning,
				imStop:     tt.fields.imStop,
			}
			b.StartWorker(tt.args.w)
			workerMock := tt.args.w.(*WorkerMock)
			g.Expect(workerMock.calls.SetIsRunning).To(gomega.HaveLen(1))
			g.Expect(workerMock.calls.GetSyncGroup).To(gomega.HaveLen(1))
		})
	}
}

func TestBaseWorker_StopWorker(t *testing.T) {
	type fields struct {
		Id         string
		WorkerType string
		Reconciler Reconciler
		isRunning  bool
		imStop     chan struct{}
	}
	stopChan := make(chan struct{})
	var wg sync.WaitGroup
	type args struct {
		w Worker
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "should stop the worker",
			fields: fields{
				Id:         "base_worker_id",
				WorkerType: "cluster",
				isRunning:  true,
				Reconciler: Reconciler{
					SignalBus:        signalbus.NewSignalBus(),
					ReconcilerConfig: NewReconcilerConfig(),
				},
			},
			args: args{
				w: &WorkerMock{
					ReconcileFunc: func() []error {
						return nil
					},
					GetIDFunc: func() string {
						return "base_worker_id"
					},
					GetStopChanFunc: func() *chan struct{} {
						return &stopChan
					},
					GetSyncGroupFunc: func() *sync.WaitGroup {
						return &wg
					},
					SetIsRunningFunc: func(val bool) {},
					IsRunningFunc: func() bool {
						return true
					},
					GetWorkerTypeFunc: func() string {
						return "cluster"
					},
				},
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			b := &BaseWorker{
				Id:         tt.fields.Id,
				WorkerType: tt.fields.WorkerType,
				Reconciler: tt.fields.Reconciler,
				isRunning:  tt.fields.isRunning,
				imStop:     tt.fields.imStop,
			}
			b.StopWorker(tt.args.w)
			workerMock := tt.args.w.(*WorkerMock)
			g.Expect(workerMock.calls.SetIsRunning).To(gomega.HaveLen(1))
			g.Expect(workerMock.calls.GetSyncGroup).To(gomega.HaveLen(1))
		})
	}
}
