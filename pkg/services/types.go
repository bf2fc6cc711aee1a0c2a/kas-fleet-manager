package services

import (
	"net/url"
	"strconv"
	"strings"
)

// ListArguments are arguments relevant for listing objects.
// This struct is common to all service List funcs in this package
type ListArguments struct {
	Page     int
	Size     int
	Preloads []string
	Search   string
	OrderBy  []string
}

// Create ListArguments from url query parameters with sane defaults
func NewListArguments(params url.Values) *ListArguments {
	listArgs := &ListArguments{
		Page:   1,
		Size:   100,
		Search: "",
	}
	if v := params.Get("page"); v != "" {
		listArgs.Page, _ = strconv.Atoi(v)
	}
	if v := params.Get("size"); v != "" {
		listArgs.Size, _ = strconv.Atoi(v)
	}
	if listArgs.Size > 65500 || listArgs.Size < 0 {
		// 65500 is the maximum number of parameters that can be provided to a postgres WHERE IN clause
		// Use it as a sane max
		listArgs.Size = 65500
	}
	if v := params.Get("search"); v != "" {
		listArgs.Search = v
	}
	if v := params.Get("orderBy"); v != "" {
		listArgs.OrderBy = strings.Split(v, ",")
	}
	return listArgs
}
