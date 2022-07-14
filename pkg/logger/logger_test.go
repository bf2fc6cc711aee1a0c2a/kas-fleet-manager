package logger

import (
	"context"
	"fmt"
	"testing"

	"github.com/onsi/gomega"
)

var (
	et           = "test-type"
	ed           = "test-description"
	testSession  = "test-session"
	testUsername = "admin"
	testAccId    = "test-acc"
	testAction   = "test-action"
	testResult   = "test-result"
	testAddr     = "test-addr"
	testId       = int64(9999)
	testOpId     = "1111"
)

func Test_NewLogEventFromString(t *testing.T) {
	type args struct {
		eventTypeAndDescription string
	}
	tests := []struct {
		name string
		args args
		want LogEvent
	}{
		{
			name: "should create new empty LogEvent",
			args: args{
				eventTypeAndDescription: "",
			},
			want: LogEvent{},
		},
		{
			name: "should create a non-empty LogEvent",
			args: args{
				eventTypeAndDescription: fmt.Sprintf("%s$$%s", et, ed),
			},
			want: LogEvent{
				Type:        et,
				Description: ed,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewLogEventFromString(tt.args.eventTypeAndDescription)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_NewLogEvent(t *testing.T) {
	type args struct {
		eventType        string
		eventDescription string
	}
	tests := []struct {
		name string
		args args
		want LogEvent
	}{
		{
			name: "should create new empty LogEvent",
			args: args{
				eventType: "",
			},
			want: LogEvent{},
		},
		{
			name: "should create a non-empty LogEvent",
			args: args{
				eventType:        et,
				eventDescription: ed,
			},
			want: LogEvent{
				Type:        et,
				Description: ed,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewLogEvent(tt.args.eventType, tt.args.eventDescription)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ToString(t *testing.T) {
	type fields struct {
		l LogEvent
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "should convert LogEvent to its string representation",
			fields: fields{
				l: LogEvent{
					Type:        et,
					Description: ed,
				},
			},
			want: fmt.Sprintf("%s%s%s", et, logEventSeparator, ed),
		},
		{
			name: "should convert LogEvent to its string representation when description is empty",
			fields: fields{
				l: LogEvent{
					Type: et,
				},
			},
			want: et,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.l.ToString()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_prepareLogPrefix(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ActionKey, testAction)
	ctx = context.WithValue(ctx, RemoteAddrKey, testAddr)
	ctx = context.WithValue(ctx, ActionResultKey, testResult)
	ctx = context.WithValue(ctx, TxIdKey, testId)
	ctx = context.WithValue(ctx, OpIDKey, testOpId)
	type args struct {
		format    string
		arguments interface{}
	}
	type fields struct {
		l *logger
	}
	tests := []struct {
		name   string
		args   args
		fields fields
		want   string
	}{
		{
			name: "should prepare Log Prefix for specific logger and arguments",
			fields: fields{
				l: &logger{
					level:     1,
					accountID: testAccId,
					username:  testUsername,
					session:   testSession,
					context:   ctx,
				},
			},
			args: args{
				format:    "%s",
				arguments: "",
			},
			want: fmt.Sprintf("user='%s' action='%s' result='%s' src_ip='%s' session='%s' tx_id='%d' accountID='%s' opid='%s'",
				testUsername, testAction, testResult, testAddr, testSession, testId, testAccId, testOpId),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.l.prepareLogPrefix(tt.args.format, tt.args.arguments)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Verbosity(t *testing.T) {
	level := int32(1)
	type args struct {
		level int32
	}
	type fields struct {
		l *logger
	}
	tests := []struct {
		name   string
		args   args
		fields fields
		want   *logger
	}{
		{
			name: "should return a logger object with verbosity determined by the function argument",
			fields: fields{
				l: &logger{
					level:     level,
					accountID: testAccId,
					username:  testUsername,
					session:   testSession,
					context:   context.Background(),
				},
			},
			args: args{
				level: 1,
			},
			want: &logger{
				context:   context.Background(),
				level:     level,
				accountID: testAccId,
				username:  testUsername,
				session:   testSession,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(tt.fields.l.V(tt.args.level)).To(gomega.Equal(tt.want))
		})
	}
}
