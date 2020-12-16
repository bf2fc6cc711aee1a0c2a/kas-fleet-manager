package services

import (
	"fmt"
	"regexp"
	"strings"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

// DbSearchQuery - struct to be used by gorm in search
type DbSearchQuery struct {
	query string
	value string
}

var (
	// ValidComparators - valid comparators for search queries
	ValidComparators = []string{"=", "<>"}
	// ValidColumnNames - valid column names for search queries
	ValidColumnNames       = []string{"region", "name", "cloud_provider", "status", "owner"}
	validSearchValueRegexp = regexp.MustCompile("^([a-zA-Z0-9-_]*[a-zA-Z0-9-_])?$")
)

// GetSearchQuery - parses searchQuery and returns query ready to be passed to
// the database. If the searchQuery isn't valid, error is returned
func GetSearchQuery(searchQuery string) ([]DbSearchQuery, *errors.ServiceError) {
	noExcessiveWhiteSpaces := removeExcessiveWhiteSpaces(searchQuery) // remove excessive white spaces
	queryTokens := strings.Fields(noExcessiveWhiteSpaces)             // tokenize
	parsedQueries, err := validateAndReturnDbQuery(searchQuery, queryTokens)
	if err != nil {
		return nil, err
	}
	return parsedQueries, nil
}

// remove any excessive white spaces inside the query string
func removeExcessiveWhiteSpaces(somestring string) string {
	space := regexp.MustCompile(`\s+`)
	return space.ReplaceAllString(somestring, " ")
}

// validate search query and return sanitized query ready to be executed by gorm
func validateAndReturnDbQuery(searchQuery string, queryTokens []string) ([]DbSearchQuery, *errors.ServiceError) {
	var dbQueries []DbSearchQuery
	var query string
	index := 0 // used to determine position of the item in order to construct full query or return error if invalid query has been passed
	for _, queryToken := range queryTokens {
		switch index {
		case 0: // column name
			columnName := strings.ToLower(queryToken)
			if !contains(ValidColumnNames, columnName) {
				return nil, errors.FailedToParseSearch("Unsupported column name for search: '%s'. Supported column names are: %s. Query invalid: %s", columnName, strings.Join(ValidColumnNames[:], ", "), searchQuery)
			}
			query = columnName
			index++
		case 1: // comparator
			comparator := strings.ToUpper(queryToken)
			if !contains(ValidComparators, comparator) {
				return nil, errors.FailedToParseSearch("Unsupported comparator: '%s'. Supported comparators are: %s. Query invalid: %s", queryToken, strings.Join(ValidComparators[:], ", "), searchQuery)
			}
			query = fmt.Sprintf("%s %s ?", query, comparator)
			index++
		case 2: // searched value
			stringToMatchRegexp := queryToken
			stringToMatchRegexp = strings.TrimPrefix(stringToMatchRegexp, "'")
			stringToMatchRegexp = strings.TrimSuffix(stringToMatchRegexp, "'")
			// startsWithWildcard := strings.HasPrefix(stringToMatchRegexp, "%")
			// endsWithWildcard := strings.HasSuffix(stringToMatchRegexp, "%")

			// if startsWithWildcard && endsWithWildcard && len(stringToMatchRegexp) <= 2 {
			// 	return nil, errors.FailedToParseSearch("Invalid search value %s in query %s. Search value may start and end with '%%' and must conform to '%s'", queryToken, searchQuery, searchValueFormat)
			// }
			// // if startsWithWildcard {
			// // 	stringToMatchRegexp = strings.TrimPrefix(stringToMatchRegexp, "%")
			// // }
			// // if endsWithWildcard {
			// // 	stringToMatchRegexp = strings.TrimSuffix(stringToMatchRegexp, "%")
			// // }
			if !validSearchValueRegexp.MatchString(stringToMatchRegexp) {
				return nil, errors.FailedToParseSearch("Invalid search value %s in query %s. Search value must conform to '%s'", queryToken, searchQuery, validSearchValueRegexp)
			}
			// if startsWithWildcard {
			// 	stringToMatchRegexp = fmt.Sprintf("%%%s", stringToMatchRegexp)
			// }
			// if endsWithWildcard {
			// 	stringToMatchRegexp = fmt.Sprintf("%s%%", stringToMatchRegexp)
			// }
			dbQueries = append(dbQueries, DbSearchQuery{query: query, value: stringToMatchRegexp})
			index++
		case 3: // only 'and' allowed here
			if strings.ToLower(queryToken) != "and" {
				return nil, errors.FailedToParseSearch("Currently only 'and' is allowed search with chained queries. Got: %s in the query %s", queryToken, searchQuery)
			}
			index = 0
		}
	}
	return dbQueries, nil
}
