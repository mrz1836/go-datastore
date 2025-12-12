package datastore

import (
	"encoding/json"

	"github.com/99designs/gqlgen/graphql"
)

// QueryParams object to use when limiting and sorting database query results
type QueryParams struct {
	Page          int    `json:"page,omitempty"`
	PageSize      int    `json:"page_size,omitempty"`
	OrderByField  string `json:"order_by_field,omitempty"`
	SortDirection string `json:"sort_direction,omitempty"`
}

// MarshalQueryParams will marshal the QueryParams struct into a GraphQL marshaler.
// If all fields of the QueryParams struct are empty or zero, it returns a GraphQL null value.
//
// Parameters:
// - m: The QueryParams struct to be marshaled.
//
// Returns:
// - A graphql.Marshaler that represents the marshaled QueryParams struct.
//
// The function performs the following steps:
// 1. Checks if all fields of the QueryParams struct are empty or zero.
// 2. If all fields are empty or zero, returns graphql.Null.
// 3. Otherwise, marshals the QueryParams struct into a generic GraphQL marshaler using graphql.MarshalAny.
func MarshalQueryParams(m QueryParams) graphql.Marshaler {
	if m.Page == 0 && m.PageSize == 0 && m.OrderByField == "" && m.SortDirection == "" {
		return graphql.Null
	}
	return graphql.MarshalAny(m)
}

// UnmarshalQueryParams will unmarshal the provided interface into a QueryParams struct.
// It handles the conversion from a generic interface to the specific QueryParams type,
// ensuring that the data is correctly parsed and assigned to the struct fields.
//
// Parameters:
// - v: The any to be unmarshalled. It is expected to be a map or a JSON-like structure.
//
// Returns:
// - QueryParams: The unmarshalled QueryParams struct with the parsed data.
// - error: An error if the unmarshalling process fails.
//
// The function performs the following steps:
// 1. Checks if the provided interface is nil, returning an empty QueryParams struct if true.
// 2. Marshals the interface into a JSON byte slice.
// 3. Unmarshal the JSON byte slice into a QueryParams struct.
// 4. Returns the populated QueryParams struct and any error encountered during the process.
func UnmarshalQueryParams(v any) (QueryParams, error) {
	if v == nil {
		return QueryParams{}, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return QueryParams{}, err
	}

	var q QueryParams
	if err = json.Unmarshal(data, &q); err != nil {
		return QueryParams{}, err
	}

	return q, nil
}
