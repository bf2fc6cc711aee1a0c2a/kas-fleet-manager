package db

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

const (
	host            = "http://test.com"
	port            = 8765
	user            = "admin"
	password        = "password"
	dbName          = "test_db"
	sslModeDisabled = "disable"
	sslModeEnabled  = "enable"
	dbCert          = "cert"
	redactedPwd     = "<REDACTED>"
	invalidPath     = "invalid"
)

func Test_AddFlags(t *testing.T) {
	emptyFlagSet := &pflag.FlagSet{}
	config := NewDatabaseConfig()
	config.AddFlags(emptyFlagSet)

	g := gomega.NewWithT(t)

	// confirming that adding flags worked
	flag, err := emptyFlagSet.GetString("db-ssl-certificate-file")

	g.Expect(err).ToNot(gomega.HaveOccurred())

	g.Expect(flag).To(gomega.Equal("secrets/db.ca_cert"))
}

func Test_ReadFiles(t *testing.T) {
	type fields struct {
		config *DatabaseConfig
	}

	tests := []struct {
		name     string
		fields   fields
		modifyFn func(config *DatabaseConfig)
		wantErr  bool
	}{
		{
			name: "should return no error when running ReadFiles with default DatabaseConfig",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			wantErr: false,
		},
		{
			name: "should return an error with misconfigured HostFile",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.HostFile = invalidPath
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured PortFile",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.PortFile = invalidPath
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured NameFile",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.NameFile = invalidPath
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured UsernameFile",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.UsernameFile = invalidPath
			},
			wantErr: true,
		},
		{
			name: "should return an error with misconfigured PasswordFile",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.PasswordFile = invalidPath
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			config := tt.fields.config
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			g.Expect(config.ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_ConnectionString(t *testing.T) {
	type fields struct {
		config *DatabaseConfig
	}

	tests := []struct {
		name     string
		fields   fields
		modifyFn func(config *DatabaseConfig)
		want     string
	}{
		{
			name: "should return a connection string for ssl disabled (by default) db connection",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.Host = host
				config.Port = port
				config.Username = user
				config.Password = password
				config.Name = dbName
			},
			want: fmt.Sprintf("host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s", host, port, user, password, dbName, sslModeDisabled),
		},
		{
			name: "should return a connection string for ssl enabled db connection",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.SSLMode = sslModeEnabled
				config.Host = host
				config.Port = port
				config.Username = user
				config.Password = password
				config.Name = dbName
				config.DatabaseCaCertFile = dbCert
			},
			want: fmt.Sprintf("host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s sslrootcert=%s", host, port, user, password, dbName, sslModeEnabled, dbCert),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			config := tt.fields.config
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			g.Expect(config.ConnectionString()).To(gomega.Equal(tt.want))
		})
	}
}

func Test_LogSafeConnectionString(t *testing.T) {
	type fields struct {
		config *DatabaseConfig
	}

	tests := []struct {
		name     string
		fields   fields
		modifyFn func(config *DatabaseConfig)
		want     string
	}{
		{
			name: "should return a redacted connection string for ssl disabled (by default) db connection",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.Host = host
				config.Port = port
				config.Username = user
				config.Password = password
				config.Name = dbName
			},
			want: fmt.Sprintf("host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s", host, port, user, redactedPwd, dbName, sslModeDisabled),
		},
		{
			name: "should return a redacted connection string for ssl enabled db connection",
			fields: fields{
				config: NewDatabaseConfig(),
			},
			modifyFn: func(config *DatabaseConfig) {
				config.SSLMode = sslModeEnabled
				config.Host = host
				config.Port = port
				config.Username = user
				config.Password = password
				config.Name = dbName
				config.DatabaseCaCertFile = dbCert
			},
			want: fmt.Sprintf("host=%s port=%d user=%s password='%s' dbname=%s sslmode=%s sslrootcert=%s", host, port, user, redactedPwd, dbName, sslModeEnabled, redactedPwd),
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			config := tt.fields.config
			if tt.modifyFn != nil {
				tt.modifyFn(config)
			}
			g.Expect(config.LogSafeConnectionString()).To(gomega.Equal(tt.want))
		})
	}
}
