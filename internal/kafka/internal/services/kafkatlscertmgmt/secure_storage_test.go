package kafkatlscertmgmt

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/caddyserver/certmagic"
	"github.com/onsi/gomega"
)

func Test_secureStorage_Delete(t *testing.T) {
	type fields struct {
		secretPrefix string
		secretClient SecretManagerClient
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "return an error when aws secret manager client returns an error",
			fields: fields{
				secretPrefix: "some-prefix",
				secretClient: &SecretManagerClientMock{
					DeleteSecretFunc: func(deleteSecretInput *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
						return nil, fmt.Errorf("some error")
					},
				},
			},
			args: args{
				key: "some-key",
			},
			wantErr: true,
		},
		{
			name: "successfully deletes the secret",
			fields: fields{
				secretPrefix: "some-other-prefix",
				secretClient: &SecretManagerClientMock{
					DeleteSecretFunc: func(deleteSecretInput *secretsmanager.DeleteSecretInput) (*secretsmanager.DeleteSecretOutput, error) {
						return &secretsmanager.DeleteSecretOutput{}, nil
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			storage := &secureStorage{
				secretPrefix: testcase.fields.secretPrefix,
				secretClient: testcase.fields.secretClient,
			}
			err := storage.Delete(context.Background(), testcase.args.key)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))

			// assert that aws secret manager call was done appropriately

			mock, ok := testcase.fields.secretClient.(*SecretManagerClientMock)
			g.Expect(ok).To(gomega.BeTrue())
			deleteCalls := mock.DeleteSecretCalls()
			g.Expect(deleteCalls).To(gomega.HaveLen(1))

			name := fmt.Sprintf("%s/%s", testcase.fields.secretPrefix, testcase.args.key)
			force := true
			g.Expect(deleteCalls[0].DeleteSecretInput).To(gomega.Equal(&secretsmanager.DeleteSecretInput{
				SecretId:                   &name,
				ForceDeleteWithoutRecovery: &force,
			}))
		})
	}
}

func Test_secureStorage_Store(t *testing.T) {
	type fields struct {
		secretPrefix string
		secretClient SecretManagerClient
	}
	type args struct {
		key   string
		value []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successfully stores the secret the secret",
			fields: fields{
				secretPrefix: "some-prefix",
				secretClient: &SecretManagerClientMock{
					CreateSecretFunc: func(createSecretInput *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
						return &secretsmanager.CreateSecretOutput{}, nil
					},
				},
			},
			args: args{
				key:   "some-key",
				value: []byte("some byte"),
			},
			wantErr: false,
		},
		{
			name: "returns an error when the error is different than ResourceExistsException",
			fields: fields{
				secretPrefix: "some-other-prefix",
				secretClient: &SecretManagerClientMock{
					CreateSecretFunc: func(createSecretInput *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
						return nil, fmt.Errorf("some error")
					},
				},
			},
			args: args{
				key:   "some-key-to-store",
				value: []byte("some byte to store"),
			},
			wantErr: true,
		},
		{
			name: "successfully updates the secret when it exists rather than raising ResourceExistsException",
			fields: fields{
				secretPrefix: "some-other-prefix",
				secretClient: &SecretManagerClientMock{
					CreateSecretFunc: func(createSecretInput *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
						return nil, &secretsmanager.ResourceExistsException{}
					},
					UpdateSecretFunc: func(updateSecretInput *secretsmanager.UpdateSecretInput) (*secretsmanager.UpdateSecretOutput, error) {
						return &secretsmanager.UpdateSecretOutput{}, nil
					},
				},
			},
			args: args{
				key:   "key-to-update",
				value: []byte("some up to date value"),
			},
			wantErr: false,
		},
		{
			name: "returns an error when fails to update the secret",
			fields: fields{
				secretPrefix: "some-other-prefix",
				secretClient: &SecretManagerClientMock{
					CreateSecretFunc: func(createSecretInput *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
						return nil, &secretsmanager.ResourceExistsException{}
					},
					UpdateSecretFunc: func(updateSecretInput *secretsmanager.UpdateSecretInput) (*secretsmanager.UpdateSecretOutput, error) {
						return nil, fmt.Errorf("some error")
					},
				},
			},
			args: args{
				key:   "key",
				value: []byte("value"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			storage := &secureStorage{
				secretPrefix: testcase.fields.secretPrefix,
				secretClient: testcase.fields.secretClient,
			}
			err := storage.Store(context.Background(), testcase.args.key, testcase.args.value)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))

			mock, ok := testcase.fields.secretClient.(*SecretManagerClientMock)
			g.Expect(ok).To(gomega.BeTrue())
			createSecretCalls := mock.CreateSecretCalls()
			g.Expect(createSecretCalls).To(gomega.HaveLen(1))
			name := fmt.Sprintf("%s/%s", testcase.fields.secretPrefix, testcase.args.key)
			g.Expect(createSecretCalls[0].CreateSecretInput).To(gomega.Equal(&secretsmanager.CreateSecretInput{
				Name:         &name,
				SecretBinary: testcase.args.value,
			}))

			if mock.UpdateSecretFunc != nil {
				updateSecretCalls := mock.UpdateSecretCalls()
				g.Expect(updateSecretCalls).To(gomega.HaveLen(1))
				g.Expect(updateSecretCalls[0].UpdateSecretInput).To(gomega.Equal(&secretsmanager.UpdateSecretInput{
					SecretId:     &name,
					SecretBinary: testcase.args.value,
				}))
			}
		})
	}
}

func Test_secureStorage_List(t *testing.T) {
	type fields struct {
		secretPrefix string
		secretClient SecretManagerClient
	}
	type args struct {
		prefix    string
		recursive bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "return an error when list returns an error",
			wantErr: true,
			want:    []string{},
			fields: fields{
				secretPrefix: "some-prefix",
				secretClient: &SecretManagerClientMock{
					ListSecretsPagesFunc: func(listSecretsInput *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
						return fmt.Errorf("some error")
					},
				},
			},
			args: args{
				prefix:    "key prefix",
				recursive: true,
			},
		},
		{
			name:    "returns the secret keys with the global prefix removed",
			wantErr: false,
			want:    []string{"example/key1", "example2/keys/key2"},
			fields: fields{
				secretPrefix: "some-prefix",
				secretClient: &SecretManagerClientMock{
					ListSecretsPagesFunc: func(listSecretsInput *secretsmanager.ListSecretsInput, fn func(*secretsmanager.ListSecretsOutput, bool) bool) error {
						keyname1 := "some-prefix/example/key1"
						keyname2 := "some-prefix/example2/keys/key2"
						fn(&secretsmanager.ListSecretsOutput{
							SecretList: []*secretsmanager.SecretListEntry{
								{
									Name: &keyname1,
								},
								{
									Name: &keyname2,
								},
							},
						}, true)
						return nil
					},
				},
			},
			args: args{
				prefix:    "example",
				recursive: true,
			},
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			storage := &secureStorage{
				secretPrefix: testcase.fields.secretPrefix,
				secretClient: testcase.fields.secretClient,
			}
			keys, err := storage.List(context.Background(), testcase.args.prefix, testcase.args.recursive)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
			g.Expect(keys).To(gomega.Equal(testcase.want))

			mock, ok := testcase.fields.secretClient.(*SecretManagerClientMock)
			g.Expect(ok).To(gomega.BeTrue())
			listPagesCalls := mock.ListSecretsPagesCalls()
			g.Expect(listPagesCalls).To(gomega.HaveLen(1))
			filterKey := "name"
			filter := fmt.Sprintf("%s/%s", testcase.fields.secretPrefix, testcase.args.prefix)
			g.Expect(listPagesCalls[0].ListSecretsInput).To(gomega.Equal(&secretsmanager.ListSecretsInput{
				Filters: []*secretsmanager.Filter{
					{Key: &filterKey, Values: []*string{&filter}}},
			}))

		})
	}
}

func Test_secureStorage_String(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)
	storage := &secureStorage{}
	g.Expect(storage.String()).To(gomega.Equal("SecureStorage"))
}

func Test_secureStorage_Load(t *testing.T) {
	type fields struct {
		secretPrefix string
		secretClient SecretManagerClient
	}
	type args struct {
		key string
	}
	version := "AWSCURRENT"

	tests := []struct {
		name                           string
		fields                         fields
		args                           args
		want                           []byte
		wantErr                        bool
		shouldReturnFileNotExistsError bool
	}{
		{
			name: "return an error when loading the secret fails with a different error than ResourceNotFoundException during secret describing from secret manager",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return nil, fmt.Errorf("some error")
					},
				},
			},
			args: args{
				key: "some-key",
			},
			wantErr: true,
		},
		{
			name: "return an error when loading the secret fails with a different error than ResourceNotFoundException during getting secret from secret manager",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return &secretsmanager.DescribeSecretOutput{
							VersionIdsToStages: map[string][]*string{
								"AWSCURRENT": {&version},
							},
						}, nil
					},
					GetSecretValueWithContextFunc: func(contextMoqParam context.Context, getSecretValueInput *secretsmanager.GetSecretValueInput, options ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
						return nil, fmt.Errorf("some-error")
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			wantErr: true,
		},
		{
			name: "return the file not found error when the secret manager returns ResourceNotFoundException",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return nil, &secretsmanager.ResourceNotFoundException{}
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			wantErr:                        true,
			shouldReturnFileNotExistsError: true,
		},
		{
			name: "return the secret",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return &secretsmanager.DescribeSecretOutput{
							VersionIdsToStages: map[string][]*string{
								"AWSCURRENT": {&version},
							},
						}, nil
					},
					GetSecretValueWithContextFunc: func(contextMoqParam context.Context, getSecretValueInput *secretsmanager.GetSecretValueInput, options ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
						return &secretsmanager.GetSecretValueOutput{
							SecretBinary: []byte("some-value"),
						}, nil
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			wantErr: false,
			want:    []byte("some-value"),
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			secretCache, err := secretcache.New(func(cache *secretcache.Cache) {
				cache.Client = testcase.fields.secretClient
			})
			g.Expect(err).ToNot(gomega.HaveOccurred())
			storage := &secureStorage{
				secretCache:  secretCache,
				secretPrefix: testcase.fields.secretPrefix,
				secretClient: testcase.fields.secretClient,
			}
			value, err := storage.Load(context.Background(), testcase.args.key)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
			g.Expect(value).To(gomega.Equal(testcase.want))
			if testcase.wantErr && testcase.shouldReturnFileNotExistsError {
				g.Expect(errors.Is(err, fs.ErrNotExist))
			}
			mock, ok := testcase.fields.secretClient.(*SecretManagerClientMock)
			g.Expect(ok).To(gomega.BeTrue())

			name := fmt.Sprintf("%s/%s", testcase.fields.secretPrefix, testcase.args.key)
			describeWithContextCalls := mock.DescribeSecretWithContextCalls()
			g.Expect(describeWithContextCalls).To(gomega.HaveLen(1))
			g.Expect(describeWithContextCalls[0].DescribeSecretInput.SecretId).To(gomega.Equal(&name))
			if mock.GetSecretValueWithContextFunc != nil {
				getWithContextCalls := mock.GetSecretValueWithContextCalls()
				g.Expect(getWithContextCalls).To(gomega.HaveLen(1))
				g.Expect(getWithContextCalls[0].GetSecretValueInput.SecretId).To(gomega.Equal(&name))
				g.Expect(getWithContextCalls[0].GetSecretValueInput.VersionId).To(gomega.Equal(&version))
			}
		})
	}
}

func Test_secureStorage_Exists(t *testing.T) {
	type fields struct {
		secretPrefix string
		secretClient SecretManagerClient
	}
	type args struct {
		key string
	}
	version := "AWSCURRENT"

	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "return false when loading secret fails with a different error than ResourceNotFoundException",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return &secretsmanager.DescribeSecretOutput{
							VersionIdsToStages: map[string][]*string{
								"AWSCURRENT": {&version},
							},
						}, nil
					},
					GetSecretValueWithContextFunc: func(contextMoqParam context.Context, getSecretValueInput *secretsmanager.GetSecretValueInput, options ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
						return nil, fmt.Errorf("some-error")
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			want: false,
		},
		{
			name: "return false when the secret manager returns ResourceNotFoundException",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return nil, &secretsmanager.ResourceNotFoundException{}
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			want: false,
		},
		{
			name: "return the true when secret exists",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return &secretsmanager.DescribeSecretOutput{
							VersionIdsToStages: map[string][]*string{
								"AWSCURRENT": {&version},
							},
						}, nil
					},
					GetSecretValueWithContextFunc: func(contextMoqParam context.Context, getSecretValueInput *secretsmanager.GetSecretValueInput, options ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
						return &secretsmanager.GetSecretValueOutput{
							SecretBinary: []byte("some-value"),
						}, nil
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			secretCache, err := secretcache.New(func(cache *secretcache.Cache) {
				cache.Client = testcase.fields.secretClient
			})
			g.Expect(err).ToNot(gomega.HaveOccurred())
			storage := &secureStorage{
				secretCache:  secretCache,
				secretPrefix: testcase.fields.secretPrefix,
				secretClient: testcase.fields.secretClient,
			}
			value := storage.Exists(context.Background(), testcase.args.key)
			g.Expect(value).To(gomega.Equal(testcase.want))
			mock, ok := testcase.fields.secretClient.(*SecretManagerClientMock)
			g.Expect(ok).To(gomega.BeTrue())

			name := fmt.Sprintf("%s/%s", testcase.fields.secretPrefix, testcase.args.key)
			describeWithContextCalls := mock.DescribeSecretWithContextCalls()
			g.Expect(describeWithContextCalls).To(gomega.HaveLen(1))
			g.Expect(describeWithContextCalls[0].DescribeSecretInput.SecretId).To(gomega.Equal(&name))
			if mock.GetSecretValueWithContextFunc != nil {
				getWithContextCalls := mock.GetSecretValueWithContextCalls()
				g.Expect(getWithContextCalls).To(gomega.HaveLen(1))
				g.Expect(getWithContextCalls[0].GetSecretValueInput.SecretId).To(gomega.Equal(&name))
				g.Expect(getWithContextCalls[0].GetSecretValueInput.VersionId).To(gomega.Equal(&version))
			}
		})
	}
}

func Test_secureStorage_Stat(t *testing.T) {
	type fields struct {
		secretPrefix string
		secretClient SecretManagerClient
	}
	type args struct {
		key string
	}
	version := "AWSCURRENT"
	changedTime := time.Now()
	value := []byte("some-value")
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    certmagic.KeyInfo
		wantErr bool
	}{
		{
			name: "return an error when loading the secret fails",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return nil, fmt.Errorf("some error")
					},
				},
			},
			args: args{
				key: "some-key",
			},
			wantErr: true,
		},
		{
			name: "return the key info",
			fields: fields{
				secretPrefix: "seret prefix",
				secretClient: &SecretManagerClientMock{
					DescribeSecretWithContextFunc: func(contextMoqParam context.Context, describeSecretInput *secretsmanager.DescribeSecretInput, options ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
						return &secretsmanager.DescribeSecretOutput{
							VersionIdsToStages: map[string][]*string{
								"AWSCURRENT": {&version},
							},
							LastChangedDate: &changedTime,
						}, nil
					},
					GetSecretValueWithContextFunc: func(contextMoqParam context.Context, getSecretValueInput *secretsmanager.GetSecretValueInput, options ...request.Option) (*secretsmanager.GetSecretValueOutput, error) {
						return &secretsmanager.GetSecretValueOutput{
							SecretBinary: value,
						}, nil
					},
				},
			},
			args: args{
				key: "some-other-key",
			},
			wantErr: false,
			want: certmagic.KeyInfo{
				Modified:   changedTime,
				Key:        "some-other-key",
				IsTerminal: false,
				Size:       int64(len(value)),
			},
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			secretCache, err := secretcache.New(func(cache *secretcache.Cache) {
				cache.Client = testcase.fields.secretClient
			})
			g.Expect(err).ToNot(gomega.HaveOccurred())
			storage := &secureStorage{
				secretCache:  secretCache,
				secretPrefix: testcase.fields.secretPrefix,
				secretClient: testcase.fields.secretClient,
			}
			value, err := storage.Stat(context.Background(), testcase.args.key)
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
			g.Expect(value).To(gomega.Equal(testcase.want))
			mock, ok := testcase.fields.secretClient.(*SecretManagerClientMock)
			g.Expect(ok).To(gomega.BeTrue())

			name := fmt.Sprintf("%s/%s", testcase.fields.secretPrefix, testcase.args.key)
			describeWithContextCalls := mock.DescribeSecretWithContextCalls()
			g.Expect(len(describeWithContextCalls) > 0).To(gomega.BeTrue()) // could be called more than once
			g.Expect(describeWithContextCalls[0].DescribeSecretInput.SecretId).To(gomega.Equal(&name))
			if mock.GetSecretValueWithContextFunc != nil {
				getWithContextCalls := mock.GetSecretValueWithContextCalls()
				g.Expect(getWithContextCalls).To(gomega.HaveLen(1))
				g.Expect(getWithContextCalls[0].GetSecretValueInput.SecretId).To(gomega.Equal(&name))
				g.Expect(getWithContextCalls[0].GetSecretValueInput.VersionId).To(gomega.Equal(&version))
			}
		})
	}
}
