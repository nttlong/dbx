package dbx

import (
	"fmt"
	"regexp"
	"strings"
)

type SqlCmd[T any] struct {
	string
	args map[string]interface{}
}

func SQL[T any](sql string) SqlCmd[T] {
	return SqlCmd[T]{
		string: sql,
		args:   map[string]interface{}{},
	}
}
func (s SqlCmd[T]) Params(args ...interface{}) SqlCmd[T] {
	if len(args)%2 != 0 {
		panic("Params must be string key and value pairs")
	}
	for i := 0; i < len(args); i += 2 {
		strParam, ok := args[i].(string)
		if !ok {
			panic("Params must be string key and value pairs")
		}
		s.args[strParam] = args[i+1]
	}
	return s
}
func (s SqlCmd[T]) Items(db *DBXTenant) ([]T, error) {
	execSQL, execArg, err := convertNamedParamsToPositional(s.string, s.args)
	if err != nil {
		return nil, err
	}

	return Select[T](db, execSQL, execArg...)

}
func (s SqlCmd[T]) Item(db *DBXTenant) (*T, error) {
	execSQL, execArg, err := convertNamedParamsToPositional(s.string, s.args)
	if err != nil {
		return nil, err
	}
	sqlOne := "SELECT * FROM (" + execSQL + ") AS t LIMIT 1"
	ret, err := Select[T](db, sqlOne, execArg...)
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}
	retOne := ret[0]
	return &retOne, nil
}

// ConvertNamedParamsToPositional converts a SQL query with named parameters (@key)
// to a query with positional parameters (?) and returns the ordered slice of values.
//
// sqlQuery: The input SQL query string with named parameters like @paramName.
// paramsMap: A map where keys are parameter names (without @) and values are their corresponding values.
//
// Returns the modified SQL query string, an ordered slice of parameter values, and an error if any parameter is missing.
func convertNamedParamsToPositional(sqlQuery string, paramsMap map[string]interface{}) (string, []interface{}, error) {
	var positionalParams []interface{} // Slice to store parameter values in order
	var builder strings.Builder        // Used to build the new SQL query string efficiently

	// Regular expression to find @ followed by one or more word characters (letters, numbers, underscore).
	// The parentheses around `[a-zA-Z0-9_]+` create a capturing group for the parameter name itself.
	re := regexp.MustCompile(`@([a-zA-Z0-9_]+)`)

	lastIndex := 0 // Keeps track of the last processed index in the original SQL query

	// Find all matches of the regex in the SQL query.
	// FindAllStringSubmatchIndex returns a slice of matches, where each match is a slice of indices:
	// [start_of_full_match, end_of_full_match, start_of_group1, end_of_group1, ...]
	matches := re.FindAllStringSubmatchIndex(sqlQuery, -1)

	// Iterate through each found named parameter
	for _, match := range matches {
		// Append the part of the original SQL query before the current named parameter
		// This is the text from the last processed index up to the start of the current match.
		builder.WriteString(sqlQuery[lastIndex:match[0]])

		// Extract the parameter name from the captured group (excluding the '@' symbol).
		// match[2] is the start index of the first capturing group.
		// match[3] is the end index of the first capturing group.
		paramName := sqlQuery[match[2]:match[3]]

		// Look up the parameter's value in the provided map.
		value, ok := paramsMap[paramName]
		if !ok {
			// If a parameter found in the query is not present in the map, return an error.
			return "", nil, fmt.Errorf("missing value for parameter: @%s", paramName)
		}

		// Append the positional placeholder '?' to the new SQL query.
		builder.WriteString("?")
		// Add the retrieved parameter value to our ordered slice.
		positionalParams = append(positionalParams, value)

		// Update lastIndex to the end of the current full match (@paramName)
		lastIndex = match[1]
	}

	// Append any remaining part of the original SQL query after the last named parameter.
	builder.WriteString(sqlQuery[lastIndex:])

	// Return the new SQL query with positional parameters and the ordered slice of values.
	return builder.String(), positionalParams, nil
}
