package migrations

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

const mockDinosaurRequestID = "1ilzo99dVkVAoQNJeovhP8pFIzS"
const mockIDWithInvalidChars = "vp&xG^nl9MStC@SI*#c$6V^TKq0"

func Test_buildOldDinosaurNamespace(t *testing.T) {
	mockDinosaurName := "example-dinosaur"
	mockDinosaurNamespace := "a1ilzo99dvkvaoqnjeovhp8pfizs"

	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Build namespace name successfully",
			args: args{
				dinosaurRequest: &dbapi.DinosaurRequest{
					Name: mockDinosaurName,
					Meta: api.Meta{
						ID: mockDinosaurRequestID,
					},
				},
			},
			wantErr: false,
			want:    mockDinosaurNamespace,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildOldDinosaurNamespace(tt.args.dinosaurRequest)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildOldDinosaurNamespace() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildOldDinosaurNamespace() got = %+v, want %+v", got, tt.want)
			}

		})
	}
}

func Test_buildDinosaurNamespaceIdentifier(t *testing.T) {
	mockShortOwnerUsername := "sample_owner_username_short"
	mockLongOwnerUsername := fmt.Sprintf("sample_owner_username_long_%s", mockDinosaurRequestID)
	namespaceLimit := 63 // Maximum namespace name length as validated by OpenShift

	type args struct {
		dinosaurRequest *dbapi.DinosaurRequest
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "build dinosaur namespace id successfully with a short owner username",
			args: args{
				dinosaurRequest: &dbapi.DinosaurRequest{
					Owner: mockShortOwnerUsername,
				},
			},
			want: fmt.Sprintf("%s-%s", mockShortOwnerUsername, strings.ToLower(mockDinosaurRequestID)),
		},
		{
			name: "build dinosaur namespace id successfully with a long owner username",
			args: args{
				dinosaurRequest: &dbapi.DinosaurRequest{
					Owner: mockLongOwnerUsername,
				},
			},
			want: fmt.Sprintf("%s-%s", mockLongOwnerUsername[0:truncatedNamespaceLen], strings.ToLower(mockDinosaurRequestID)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.dinosaurRequest.ID = mockDinosaurRequestID
			got := buildDinosaurNamespaceIdentifier(tt.args.dinosaurRequest)
			if got != tt.want {
				t.Errorf("BuildDinosaurNamespaceIdentifier() = %v, want %v", got, tt.want)
			}
			if len(got) > namespaceLimit {
				t.Errorf("BuildDinosaurNamespaceIdentifier() namespace identifier length is %v, this is over the %v maximum namespace name limit", len(got), namespaceLimit)
			}
		})
	}
}

func Test_replaceNamespaceSpecialChar(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "replace all invalid characters in an invalid namespace name",
			args: args{
				name: fmt.Sprintf("project-%s-", mockIDWithInvalidChars),
			},
			want: "project-vp-xg-nl9mstc-si-c-6v-tkq0a",
		},
		{
			name: "valid namespace should not be modified",
			args: args{
				name: fmt.Sprintf("project-%s", mockDinosaurRequestID),
			},
			want: fmt.Sprintf("project-%s", strings.ToLower(mockDinosaurRequestID)),
		},
		{
			name: "should return an error if given namespace name is an empty string",
			args: args{
				name: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceNamespaceSpecialChar(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceNamespaceSpecialChar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReplaceNamespaceSpecialChar() = %v, want %v", got, tt.want)
			}
		})
	}
}
