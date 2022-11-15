package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
)

var (
	mockConn = NewMockConnectionFactory(nil)
	tx       = txFactory{
		resolved:          true,
		rollbackFlag:      true,
		tx:                &sql.Tx{},
		txid:              0,
		postCommitActions: nil,
		db:                nil,
	}
	c  = context.Background()
	c2 = context.WithValue(c, constants.TransactionKey, &tx)
	f  = func() {}
)

func Test_NewContext(t *testing.T) {
	transaction, err := mockConn.newTransaction()
	c3 := context.WithValue(c, constants.TransactionIDkey, transaction.txid)
	c3 = context.WithValue(c3, constants.TransactionKey, transaction)
	type fields struct {
		c *ConnectionFactory
	}
	type args struct {
		ctx context.Context
		err error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    context.Context
		wantErr bool
	}{
		{
			name: "should return an error if original context is returned",
			fields: fields{
				c: mockConn,
			},
			args: args{
				ctx: c,
				err: err,
			},
			want:    c,
			wantErr: true,
		},
		{
			name: "should succeed if returned a new context with transaction stored in it",
			fields: fields{
				c: mockConn,
			},
			args: args{
				ctx: c,
				err: nil,
			},
			want:    c3,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			_, err := tt.fields.c.NewContext(tt.args.ctx)
			if err != nil {
				g.Expect(err != nil).To(gomega.Equal(true))
			} else {
				g.Expect(tt.fields.c.NewContext(tt.args.ctx)).To(gomega.Equal(tt.want))
			}
		})
	}
}

func Test_TXContext(t *testing.T) {
	testWithContext, err := mockConn.NewContext(c)
	type fields struct {
		c *ConnectionFactory
	}
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    context.Context
		wantErr bool
	}{
		{
			name: "should fail if creating a new transaction context fails",
			fields: fields{
				c: mockConn,
			},
			args: args{
				err: err,
			},
			want:    testWithContext,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			_, err := tt.fields.c.TxContext()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_Resolve(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    error
		wantErr bool
	}{
		{
			name: "should fail if unable to retrieve transaction from context",
			args: args{
				ctx: c,
			},
			want:    fmt.Errorf("could not retrieve transaction from context"),
			wantErr: true,
		},
		{
			name: "should succeed if transaction retrieved",
			args: args{
				ctx: c2,
			},
			want:    nil,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			if tt.wantErr {
				g.Expect(Resolve(tt.args.ctx)).To(gomega.Equal(tt.want))
			}
		})
	}
}

func Test_Begin(t *testing.T) {
	type fields struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		want    error
		wantErr bool
	}{
		{
			name: "should fail if unable to retrieve transaction from context",
			fields: fields{
				ctx: c,
			},
			want:    fmt.Errorf("could not retrieve transaction from context"),
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			g.Expect(Begin(tt.fields.ctx)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_AddPostCommitAction(t *testing.T) {
	tx.postCommitActions = append(tx.postCommitActions, f)
	type fields struct {
		ctx context.Context
		f   func()
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr error
	}{
		{
			name: "should succeed",
			fields: fields{
				ctx: c2,
			},
			wantErr: nil,
		},
		{
			name: "should fail if unable to retrieve transaction from context",
			fields: fields{
				ctx: c,
			},
			wantErr: fmt.Errorf("could not retrieve transaction from context"),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			if tt.wantErr != nil {
				g.Expect(AddPostCommitAction(tt.fields.ctx, tt.fields.f)).To(gomega.Equal(tt.wantErr))
			} else {
				g.Expect(AddPostCommitAction(tt.fields.ctx, tt.fields.f)).To(gomega.BeNil())
			}
		})
	}
}

func Test_FromContext(t *testing.T) {
	type fields struct {
		ctx context.Context
	}
	type args struct {
		err error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *sql.Tx
		wantErr bool
	}{
		{
			name: "should fail if could not retrieve transaction from context",
			fields: fields{
				ctx: c,
			},
			args: args{
				err: errors.GeneralError("could not retrieve transaction from context"),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "should succeed and retrieve transaction",
			fields: fields{
				ctx: c2,
			},
			args: args{
				err: nil,
			},
			want:    tx.tx,
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			_, err := FromContext(tt.fields.ctx)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_MarkForRollback(t *testing.T) {
	txF := txFactory{
		resolved:          true,
		rollbackFlag:      false,
		tx:                &sql.Tx{},
		txid:              0,
		postCommitActions: nil,
		db:                nil,
	}
	c4 := context.WithValue(c, constants.TransactionKey, &txF)
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name             string
		args             args
		transactionFound bool
	}{
		{
			name: "should fail is unable to mark transaction for rollback",
			args: args{
				ctx: c,
			},
		},
		{
			name: "should succeed if transaction was marked",
			args: args{
				ctx: c4,
			},
			transactionFound: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			transaction, ok := tt.args.ctx.Value(constants.TransactionKey).(*txFactory)
			if ok {
				g.Expect(transaction.rollbackFlag).To(gomega.Equal(false))
				MarkForRollback(tt.args.ctx, nil)
				transaction = tt.args.ctx.Value(constants.TransactionKey).(*txFactory)
				g.Expect(transaction.rollbackFlag).To(gomega.Equal(true))
			} else {
				MarkForRollback(tt.args.ctx, errors.GeneralError(""))
			}
		})
	}
}
