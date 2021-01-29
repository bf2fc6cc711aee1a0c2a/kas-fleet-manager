package services

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
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
	incompleteQuery := fmt.Sprintf("%s %s", ValidColumnNames[0], ValidComparators[0])
	incompleteQueryWithJoin := fmt.Sprintf("%s %s %s AND %s %s", ValidColumnNames[0], ValidComparators[0], searchValue, ValidColumnNames[0], ValidComparators[0])
	unsupportedColumnName := "unsupported_column_name"
	unsupportedColumnNameQuery := fmt.Sprintf("%s %s %s", unsupportedColumnName, ValidComparators[0], searchValue)
	unsupportedComparator := "similar_to"
	unsupportedComparatorQuery := fmt.Sprintf("%s %s %s", ValidColumnNames[0], unsupportedComparator, searchValue)
	unsupportedJoinValue := "nand"
	unsupportedJoinQuery := fmt.Sprintf("%s %s %s %s %s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue, unsupportedJoinValue, ValidColumnNames[1], ValidComparators[1], searchValue)
	unsupportedSearchString := "%%777"
	unsupportedSearchStringQuery := fmt.Sprintf("%s %s %s", ValidColumnNames[0], ValidComparators[0], unsupportedSearchString)
	wildcardString := "%kafka"
	wildcardStringQuery := fmt.Sprintf("%s LIKE %s", ValidColumnNames[0], wildcardString)
	wildcardStringQueryWithoutLike := fmt.Sprintf("%s = %s", ValidColumnNames[0], wildcardString)
	tooManyJoins := fmt.Sprintf(
		"%s %s %s AND %s %s %s OR %s %s %s AND %s %s %s OR %s %s %s OR %s %s %s OR %s %s %s AND %s %s %s OR %s %s %s AND %s %s %s OR %s %s %s OR %s %s %s",
		ValidColumnNames[0],
		ValidComparators[0],
		searchValue,
		ValidColumnNames[1],
		ValidComparators[1],
		searchValue,
		ValidColumnNames[2],
		ValidComparators[2],
		searchValue,
		ValidColumnNames[0],
		ValidComparators[0],
		searchValue,
		ValidColumnNames[1],
		ValidComparators[1],
		searchValue,
		ValidColumnNames[2],
		ValidComparators[2],
		searchValue,
		ValidColumnNames[0],
		ValidComparators[0],
		searchValue,
		ValidColumnNames[1],
		ValidComparators[1],
		searchValue,
		ValidColumnNames[2],
		ValidComparators[2],
		searchValue,
		ValidColumnNames[0],
		ValidComparators[0],
		searchValue,
		ValidColumnNames[1],
		ValidComparators[1],
		searchValue,
		ValidColumnNames[2],
		ValidComparators[2],
		searchValue,
	)
	type args struct {
		query string
	}
	type want struct {
		parsedQuery DbSearchQuery
		err         *errors.ServiceError
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    want
	}{
		{
			name: "Incomplete query",
			args: args{
				query: incompleteQuery,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Provided search query seems incomplete: '%s'", incompleteQuery),
			},
		},
		{
			name: "Wildcard without using LIKE comparator",
			args: args{
				query: wildcardStringQueryWithoutLike,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Invalid search value %s in query %s. Wildcards only allowed with `LIKE` comparator", wildcardString, wildcardStringQueryWithoutLike),
			},
		},
		{
			name: "Incomplete query with join",
			args: args{
				query: incompleteQuery,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Provided search query seems incomplete: '%s'", incompleteQueryWithJoin),
			},
		},
		{
			name: "Query with not supported column name",
			args: args{
				query: unsupportedColumnNameQuery,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Unsupported column name for search: '%s'. Supported column names are: %s. Query invalid: %s", unsupportedColumnName, strings.Join(ValidColumnNames[:], ", "), unsupportedColumnNameQuery),
			},
		},
		{
			name: "Query with not supported comparator",
			args: args{
				query: unsupportedComparatorQuery,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Unsupported comparator: '%s'. Supported comparators are: %s. Query invalid: %s", unsupportedComparator, strings.Join(ValidComparators[:], ", "), unsupportedComparatorQuery),
			},
		},
		{
			name: "Query with not supported search value",
			args: args{
				query: unsupportedSearchStringQuery,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Invalid search value %s in query %s. Search value may start and end with '%%' and must conform to '%s'", unsupportedSearchString, unsupportedSearchStringQuery, validSearchValueRegexp),
			},
		},
		{
			name: "Query with not supported join value",
			args: args{
				query: unsupportedJoinQuery,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Currently only 'and' is allowed search with chained queries. Got: %s in the query %s", unsupportedJoinValue, unsupportedJoinQuery),
			},
		},
		{
			name: "Too many joins",
			args: args{
				query: tooManyJoins,
			},
			wantErr: true,
			want: want{
				parsedQuery: DbSearchQuery{},
				err:         errors.FailedToParseSearch("Provided search query has too many joins (max 10 allowed): '%s'", tooManyJoins),
			},
		},
		{
			name: "Valid query with excessive white spaces",
			args: args{
				query: fmt.Sprintf("  %s    %s    %s  ", ValidColumnNames[0], ValidComparators[0], searchValue),
			},
			wantErr: false,
			want: want{
				parsedQuery: DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[0], ValidComparators[0]), values: stringSliceToInterfaceSlice([]string{searchValue})},
				err:         nil,
			},
		},
		{
			name: "Valid wildcard query with LIKE",
			args: args{
				query: wildcardStringQuery,
			},
			wantErr: false,
			want: want{
				parsedQuery: DbSearchQuery{query: fmt.Sprintf("%s LIKE ?", ValidColumnNames[0]), values: stringSliceToInterfaceSlice([]string{"%kafka"})},
				err:         nil,
			},
		},
		{
			name: "Valid query without ANDs",
			args: args{
				query: fmt.Sprintf("%s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue),
			},
			wantErr: false,
			want: want{
				parsedQuery: DbSearchQuery{query: fmt.Sprintf("%s %s ?", ValidColumnNames[0], ValidComparators[0]), values: stringSliceToInterfaceSlice([]string{searchValue})},
				err:         nil,
			},
		},
		{
			name: "Valid query with valid join",
			args: args{
				query: fmt.Sprintf("%s %s %s AND %s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue, ValidColumnNames[1], ValidComparators[1], searchValue),
			},
			wantErr: false,
			want: want{
				parsedQuery: DbSearchQuery{query: fmt.Sprintf("%s %s ? AND %s %s ?", ValidColumnNames[0], ValidComparators[0], ValidColumnNames[1], ValidComparators[1]), values: stringSliceToInterfaceSlice([]string{searchValue, searchValue})},
				err:         nil,
			},
		},
		{
			name: "Valid query with two valid joins",
			args: args{
				query: fmt.Sprintf("%s %s %s AND %s %s %s OR %s %s %s", ValidColumnNames[0], ValidComparators[0], searchValue, ValidColumnNames[1], ValidComparators[1], searchValue, ValidColumnNames[2], ValidComparators[2], searchValue),
			},
			wantErr: false,
			want: want{
				parsedQuery: DbSearchQuery{query: fmt.Sprintf("%s %s ? AND %s %s ? OR %s %s ?", ValidColumnNames[0], ValidComparators[0], ValidColumnNames[1], ValidComparators[1], ValidColumnNames[2], ValidComparators[2]), values: stringSliceToInterfaceSlice([]string{searchValue, searchValue, searchValue})},
				err:         nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSearchQuery(tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSearchQuery() error: %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want.parsedQuery) {
				t.Errorf("GetSearchQuery() got: %v, want %v", got, tt.want)
				return
			}
		})
	}
}
