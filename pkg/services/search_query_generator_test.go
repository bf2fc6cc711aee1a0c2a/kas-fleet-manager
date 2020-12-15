package services

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

func Test_removeExcessiveWhiteSpaces(t *testing.T) {
	whiteSpaced := "some   string     with     spaces"
	nonWhiteSpaced := "some string with spaces"
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Remove excessive white spaces",
			args: args{
				s: whiteSpaced,
			},
			want: nonWhiteSpaced,
		},
		{
			name: "No extra spaces - nothing to remove",
			args: args{
				s: nonWhiteSpaced,
			},
			want: nonWhiteSpaced,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeExcessiveWhiteSpaces(tt.args.s)
			if got != tt.want {
				t.Errorf("removeExcessiveWhiteSpaces() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetSearchQuery(t *testing.T) {
	searchValue := "searchValue"
	unsupportedColumnName := "unsupported_column_name"
	unsupportedColumnNameQuery := fmt.Sprintf("%s %s %s", unsupportedColumnName, ValidComparators[0], searchValue)
	unsupportedComparator := "similar_to"
	unsupportedComparatorQuery := fmt.Sprintf("%s %s %s", ValidColumnNames[0], unsupportedComparator, searchValue)
	unsupportedJoinValue := "nand"
	unsupportedJoinQuery := fmt.Sprintf("%s %s %s %s %s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue, unsupportedJoinValue, ValidColumnNames[1], ValidComparators[1], searchValue)
	unsupportedSearchString := "777"
	unsupportedSearchStringQuery := fmt.Sprintf("%s %s %s", ValidColumnNames[0], ValidComparators[0], unsupportedSearchString)
	type args struct {
		query string
	}
	type want struct {
		queries []DbSearchQuery
		err     *errors.ServiceError
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    want
	}{
		{
			name: "Query with not supported column name",
			args: args{
				query: unsupportedColumnNameQuery,
			},
			wantErr: true,
			want: want{
				queries: nil,
				err:     errors.FailedToParseSearch("Unsupported column name for search: '%s'. Supported column names are: %s. Query invalid: %s", unsupportedColumnName, strings.Join(ValidColumnNames[:], ", "), unsupportedColumnNameQuery),
			},
		},
		{
			name: "Query with not supported comparator",
			args: args{
				query: unsupportedComparatorQuery,
			},
			wantErr: true,
			want: want{
				queries: nil,
				err:     errors.FailedToParseSearch("Unsupported comparator: '%s'. Supported comparators are: %s. Query invalid: %s", unsupportedComparator, strings.Join(ValidComparators[:], ", "), unsupportedComparatorQuery),
			},
		},
		{
			name: "Query with not supported search value",
			args: args{
				query: unsupportedComparatorQuery,
			},
			wantErr: true,
			want: want{
				queries: nil,
				err:     errors.FailedToParseSearch("Invalid search value %s in query %s. Search value may start and end with '%%' and must conform to '%s'", unsupportedSearchString, unsupportedSearchStringQuery, validSearchValueRegexp),
			},
		},
		{
			name: "Query with not supported join value",
			args: args{
				query: unsupportedJoinQuery,
			},
			wantErr: true,
			want: want{
				queries: nil,
				err:     errors.FailedToParseSearch("Currently only 'and' is allowed search with chained queries. Got: %s in the query %s", unsupportedJoinValue, unsupportedJoinQuery),
			},
		},
		{
			name: "Valid query with excessive white spaces",
			args: args{
				query: fmt.Sprintf("  %s    %s    %s  ", ValidColumnNames[0], ValidComparators[0], searchValue),
			},
			wantErr: false,
			want: want{
				queries: []DbSearchQuery{DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[0], ValidComparators[0]), value: searchValue}},
				err:     nil,
			},
		},
		{
			name: "Valid query without ANDs",
			args: args{
				query: fmt.Sprintf("%s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue),
			},
			wantErr: false,
			want: want{
				queries: []DbSearchQuery{DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[0], ValidComparators[0]), value: searchValue}},
				err:     nil,
			},
		},
		{
			name: "Valid query with one AND",
			args: args{
				query: fmt.Sprintf("%s %s %s and %s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue, ValidColumnNames[1], ValidComparators[1], searchValue),
			},
			wantErr: false,
			want: want{
				queries: []DbSearchQuery{
					DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[0], ValidComparators[0]), value: searchValue},
					DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[1], ValidComparators[1]), value: searchValue},
				},
				err: nil,
			},
		},
		{
			name: "Valid query with two ANDs",
			args: args{
				query: fmt.Sprintf("%s %s %s and %s %s %s and %s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue, ValidColumnNames[1], ValidComparators[1], searchValue, ValidColumnNames[2], ValidComparators[0], searchValue),
			},
			wantErr: false,
			want: want{
				queries: []DbSearchQuery{
					DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[0], ValidComparators[0]), value: searchValue},
					DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[1], ValidComparators[1]), value: searchValue},
					DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[2], ValidComparators[0]), value: searchValue},
				},
				err: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSearchQuery(tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSearchQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want.queries) {
				t.Errorf("GetSearchQuery() got = %v, want %v", got, tt.want)
				return
			}
		})
	}
}
