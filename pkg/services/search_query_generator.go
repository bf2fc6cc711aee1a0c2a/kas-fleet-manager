package services

import (
	"fmt"
	"regexp"
	"strings"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	// up to 10 joins (10 * 4 ("<column> <comparator> <value> <join>") + 3 (last query: "<column> <comparator> <value>"))
	maxQueryTokenLength = 43
)

// DbSearchQuery - struct to be used by gorm in search
type DbSearchQuery struct {
	query  string
	values []interface{}
}

var (
	// ValidJoins for chained queries
	ValidJoins = []string{"AND", "OR"}
	// ValidComparators - valid comparators for search queries
	ValidComparators = []string{"=", "<>", "LIKE"}
	// ValidColumnNames - valid column names for search queries
	ValidColumnNames       = []string{"region", "name", "cloud_provider", "status", "owner"}
	validSearchValueRegexp = regexp.MustCompile("^([a-zA-Z0-9-_%]*[a-zA-Z0-9-_%])?$")
)

// GetSearchQuery - parses searchQuery and returns query ready to be passed to
// the database. If the searchQuery isn't valid, error is returned
func GetSearchQuery(searchQuery string) (DbSearchQuery, *errors.ServiceError) {
	noExcessiveWhiteSpaces := removeExcessiveWhiteSpaces(searchQuery) // remove excessive white spaces
	queryTokens := strings.Fields(noExcessiveWhiteSpaces)             // tokenize
	parsedQuery, err := validateAndReturnDbQuery(searchQuery, queryTokens)
	if err != nil {
		return DbSearchQuery{}, err
	}
	return parsedQuery, nil
}

// remove any excessive white spaces inside the query string
func removeExcessiveWhiteSpaces(somestring string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(somestring, " ")
}

// validate search query and return sanitized query ready to be executed by gorm
func validateAndReturnDbQuery(searchQuery string, queryTokens []string) (DbSearchQuery, *errors.ServiceError) {
	// checking for incomplete queries (query tokens length must be either 3 or 3 + n * 4)
	if !(len(queryTokens) == 3 || (len(queryTokens)+1)%4 == 0) {
		return DbSearchQuery{}, errors.FailedToParseSearch("Provided search query seems incomplete: '%s'", searchQuery)
	}
	// limit to 10 joins
	if len(queryTokens) > maxQueryTokenLength {
		return DbSearchQuery{}, errors.FailedToParseSearch("Provided search query has too many joins (max 10 allowed): '%s'", searchQuery)
	}
	var searchValues []string
	var query string
	var comparator string
	index := 0 // used to determine position of the item in order to construct full query or return error if invalid query has been passed
	for _, queryToken := range queryTokens {
		switch index {
		case 0: // column name
			columnName := strings.ToLower(queryToken)
			if !contains(ValidColumnNames, columnName) {
				return DbSearchQuery{}, errors.FailedToParseSearch("Unsupported column name for search: '%s'. Supported column names are: %s. Query invalid: %s", columnName, strings.Join(ValidColumnNames[:], ", "), searchQuery)
			}
			query = fmt.Sprintf("%s%s", query, columnName)
			index++
		case 1: // comparator
			comparator = strings.ToUpper(queryToken)
			if !contains(ValidComparators, comparator) {
				return DbSearchQuery{}, errors.FailedToParseSearch("Unsupported comparator: '%s'. Supported comparators are: %s. Query invalid: %s", queryToken, strings.Join(ValidComparators[:], ", "), searchQuery)
			}
			query = fmt.Sprintf("%s %s ?", query, comparator)
			index++
		case 2: // searched value
			stringToMatchRegexp := queryToken
			stringToMatchRegexp = strings.TrimPrefix(stringToMatchRegexp, "'")
			stringToMatchRegexp = strings.TrimSuffix(stringToMatchRegexp, "'")
			startsWithWildcard := strings.HasPrefix(stringToMatchRegexp, "%")
			endsWithWildcard := strings.HasSuffix(stringToMatchRegexp, "%")

			if (startsWithWildcard || endsWithWildcard) && comparator != "LIKE" {
				return DbSearchQuery{}, errors.FailedToParseSearch("Invalid search value %s in query %s. Wildcards only allowed with `LIKE` comparator", queryToken, searchQuery)
			}
			if startsWithWildcard && endsWithWildcard && len(stringToMatchRegexp) <= 2 {
				return DbSearchQuery{}, errors.FailedToParseSearch("Invalid search value %s in query %s. Search value may start and/ or end with '%%25' and must conform to '%s'", queryToken, searchQuery, validSearchValueRegexp.String())
			}

			if !validSearchValueRegexp.MatchString(stringToMatchRegexp) {
				return DbSearchQuery{}, errors.FailedToParseSearch("Invalid search value %s in query %s. Search value may start and/ or end with '%%25' when using 'LIKE' comparator and must conform to '%s'", queryToken, searchQuery, validSearchValueRegexp.String())
			}
			searchValues = append(searchValues, stringToMatchRegexp)
			index++
		case 3: // only 'and' allowed here
			join := strings.ToUpper(queryToken)
			if !contains(ValidJoins, join) {
				return DbSearchQuery{}, errors.FailedToParseSearch("Unsupported join value: '%s'. Supported joins are: %s. Query invalid: %s", queryToken, strings.Join(ValidJoins[:], ", "), searchQuery)
			}
			query = fmt.Sprintf("%s %s ", query, join)
			index = 0
		}
	}
	return DbSearchQuery{query: strings.TrimSuffix(query, " "), values: stringSliceToInterfaceSlice(searchValues)}, nil
}
