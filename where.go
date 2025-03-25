package datastore

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	customtypes "github.com/mrz1836/go-datastore/custom_types"
	"gorm.io/gorm"
)

// CustomWhereInterface is an interface for the CustomWhere clauses
type CustomWhereInterface interface {
	Where(query interface{}, args ...interface{})
	getGormTx() *gorm.DB
}

// CustomWhere add conditions
func (c *Client) CustomWhere(tx CustomWhereInterface, conditions map[string]interface{}, engine Engine) interface{} {

	// Empty accumulator
	varNum := 0

	// Process the conditions
	processConditions(c, tx, conditions, engine, &varNum, nil)

	// Return the GORM tx
	return tx.getGormTx()
}

// txAccumulator is the accumulator struct
type txAccumulator struct {
	CustomWhereInterface
	WhereClauses []string
	Vars         map[string]interface{}
}

// Where is our custom where method
func (tx *txAccumulator) Where(query interface{}, args ...interface{}) {
	tx.WhereClauses = append(tx.WhereClauses, query.(string))

	if len(args) > 0 {
		for _, variables := range args {
			for key, value := range variables.(map[string]interface{}) {
				tx.Vars[key] = value
			}
		}
	}
}

// getGormTx will get the GORM tx
func (tx *txAccumulator) getGormTx() *gorm.DB {
	return nil
}

// processConditions processes the given conditions and constructs the appropriate SQL WHERE clauses.
// It supports various conditions such as AND, OR, greater than, less than, etc., and formats them
// according to the specified database engine (MySQL, PostgreSQL, SQLite).
//
// Parameters:
// - client: The client interface that provides methods to get array and object fields.
// - tx: The transaction interface that allows adding WHERE clauses.
// - conditions: A map of conditions to be processed.
// - engine: The database engine type (MySQL, PostgreSQL, SQLite).
// - varNum: A pointer to an integer that keeps track of the variable number for parameterized queries.
// - parentKey: An optional parent key used for nested conditions.
//
// Returns:
// - The processed conditions map.
//
// The function iterates over the conditions map and processes each condition based on its key.
// It handles various condition types such as:
// - AND: Combines multiple conditions with AND logic.
// - OR: Combines multiple conditions with OR logic.
// - Greater than, less than, greater than or equal, less than or equal, not equals: Compares field values.
// - EXISTS: Checks if a field exists or not.
// - IN: Checks if a field value is within a specified set of values.
// - Array and object fields: Processes conditions for array and object fields.
//
// The function also formats the conditions based on the database engine and generates the appropriate
// SQL WHERE clauses and variables for parameterized queries.
func processConditions(client ClientInterface, tx CustomWhereInterface, conditions map[string]interface{},
	engine Engine, varNum *int, parentKey *string) map[string]interface{} { //nolint:unparam // this param might be used in the future

	for key, condition := range conditions {
		if key == conditionAnd {
			processWhereAnd(client, tx, condition, engine, varNum)
		} else if key == conditionOr {
			processWhereOr(client, tx, conditions[conditionOr], engine, varNum)
		} else if key == conditionGreaterThan {
			varName := "var" + strconv.Itoa(*varNum)
			tx.Where(*parentKey+" > @"+varName, map[string]interface{}{varName: formatCondition(condition, engine)})
			*varNum++
		} else if key == conditionLessThan {
			varName := "var" + strconv.Itoa(*varNum)
			tx.Where(*parentKey+" < @"+varName, map[string]interface{}{varName: formatCondition(condition, engine)})
			*varNum++
		} else if key == conditionGreaterThanOrEqual {
			varName := "var" + strconv.Itoa(*varNum)
			tx.Where(*parentKey+" >= @"+varName, map[string]interface{}{varName: formatCondition(condition, engine)})
			*varNum++
		} else if key == conditionLessThanOrEqual {
			varName := "var" + strconv.Itoa(*varNum)
			tx.Where(*parentKey+" <= @"+varName, map[string]interface{}{varName: formatCondition(condition, engine)})
			*varNum++
		} else if key == conditionNotEquals {
			varName := "var" + strconv.Itoa(*varNum)
			tx.Where(*parentKey+" != @"+varName, map[string]interface{}{varName: formatCondition(condition, engine)})
			*varNum++
		} else if key == conditionExists {
			if condition.(bool) {
				tx.Where(*parentKey + " IS NOT NULL")
			} else {
				tx.Where(*parentKey + " IS NULL")
			}
		} else if key == conditionIn {
			varNames := make([]string, len(condition.([]interface{})))
			vars := make(map[string]interface{})
			for i, val := range condition.([]interface{}) {
				varName := "var" + strconv.Itoa(*varNum)
				varNames[i] = "@" + varName
				vars[varName] = formatCondition(val, engine)
				*varNum++
			}
			tx.Where(*parentKey+" IN ("+strings.Join(varNames, ",")+")", vars)
		} else if StringInSlice(key, client.GetArrayFields()) {
			tx.Where(whereSlice(engine, key, formatCondition(condition, engine)))
		} else if StringInSlice(key, client.GetObjectFields()) {
			tx.Where(whereObject(engine, key, formatCondition(condition, engine)))
		} else {
			if condition == nil {
				tx.Where(key + " IS NULL")
			} else {
				v := reflect.ValueOf(condition)
				switch v.Kind() { //nolint:exhaustive // we only need to handle these cases
				case reflect.Map:
					if _, ok := condition.(map[string]interface{}); ok {
						processConditions(client, tx, condition.(map[string]interface{}), engine, varNum, &key)
					} else {
						c, _ := json.Marshal(condition) //nolint: errchkjson // this code does not retun an error, we can alternatively log it
						var cc map[string]interface{}
						_ = json.Unmarshal(c, &cc)
						processConditions(client, tx, cc, engine, varNum, &key)
					}
				default:
					varName := "var" + strconv.Itoa(*varNum)
					tx.Where(key+" = @"+varName, map[string]interface{}{varName: formatCondition(condition, engine)})
					*varNum++
				}
			}
		}
	}

	return conditions
}

// formatCondition formats the given condition based on the specified database engine.
// It handles custom types and ensures the condition is in the correct format for the database.
//
// Parameters:
// - condition: The condition to be formatted. It can be of various types, including custom types.
// - engine: The database engine type (MySQL, PostgreSQL, SQLite).
//
// Returns:
// - The formatted condition, ready to be used in a SQL query.
//
// The function checks the type of the condition and formats it accordingly:
// - For customtypes.NullTime, it formats the time based on the database engine:
//   - MySQL: "2006-01-02 15:04:05"
//   - PostgreSQL: "2006-01-02T15:04:05Z07:00"
//   - SQLite (default): "2006-01-02T15:04:05.000Z"
//
// - For other types, it returns the condition as-is.
func formatCondition(condition interface{}, engine Engine) interface{} {
	switch v := condition.(type) {
	case customtypes.NullTime:
		if v.Valid {
			if engine == MySQL {
				return v.Time.Format("2006-01-02 15:04:05")
			} else if engine == PostgreSQL {
				return v.Time.Format("2006-01-02T15:04:05Z07:00")
			}
			// default & SQLite
			return v.Time.Format("2006-01-02T15:04:05.000Z")
		}
		return nil
	}

	return condition
}

// processWhereAnd processes the AND conditions and constructs the appropriate SQL WHERE clauses.
// It accumulates the conditions and combines them with AND logic.
//
// Parameters:
// - client: The client interface that provides methods to get array and object fields.
// - tx: The transaction interface that allows adding WHERE clauses.
// - condition: The AND condition to be processed. It is expected to be a slice of maps containing conditions.
// - engine: The database engine type (MySQL, PostgreSQL, SQLite).
// - varNum: A pointer to an integer that keeps track of the variable number for parameterized queries.
//
// The function iterates over the slice of conditions and processes each one using the processConditions function.
// It accumulates the WHERE clauses and variables, and combines them with AND logic.
// Finally, it adds the combined WHERE clause to the transaction.
func processWhereAnd(client ClientInterface, tx CustomWhereInterface, condition interface{}, engine Engine, varNum *int) {
	accumulator := &txAccumulator{
		WhereClauses: make([]string, 0),
		Vars:         make(map[string]interface{}),
	}
	for _, c := range condition.([]map[string]interface{}) {
		processConditions(client, accumulator, c, engine, varNum, nil)
	}

	if len(accumulator.Vars) > 0 {
		tx.Where(" ( "+strings.Join(accumulator.WhereClauses, " AND ")+" ) ", accumulator.Vars)
	} else {
		tx.Where(" ( " + strings.Join(accumulator.WhereClauses, " AND ") + " ) ")
	}
}

// processWhereOr processes the OR conditions and constructs the appropriate SQL WHERE clauses.
// It accumulates the conditions and combines them with OR logic.
//
// Parameters:
// - client: The client interface that provides methods to get array and object fields.
// - tx: The transaction interface that allows adding WHERE clauses.
// - condition: The OR condition to be processed. It is expected to be a slice of maps containing conditions.
// - engine: The database engine type (MySQL, PostgreSQL, SQLite).
// - varNum: A pointer to an integer that keeps track of the variable number for parameterized queries.
//
// The function iterates over the slice of conditions and processes each one using the processConditions function.
// It accumulates the WHERE clauses and variables, and combines them with OR logic.
// Finally, it adds the combined WHERE clause to the transaction.
func processWhereOr(client ClientInterface, tx CustomWhereInterface, condition interface{}, engine Engine, varNum *int) {
	or := make([]string, 0)
	orVars := make(map[string]interface{})
	for _, cond := range condition.([]map[string]interface{}) {
		statement := make([]string, 0)
		accumulator := &txAccumulator{
			WhereClauses: make([]string, 0),
			Vars:         make(map[string]interface{}),
		}
		processConditions(client, accumulator, cond, engine, varNum, nil)
		statement = append(statement, accumulator.WhereClauses...)
		for varName, varValue := range accumulator.Vars {
			orVars[varName] = varValue
		}
		or = append(or, strings.Join(statement[:], " AND "))
	}

	if len(orVars) > 0 {
		tx.Where(" ( ("+strings.Join(or, ") OR (")+") ) ", orVars)
	} else {
		tx.Where(" ( (" + strings.Join(or, ") OR (") + ") ) ")
	}
}

// escapeDBString will escape the database string
func escapeDBString(s string) string {
	rs := strings.ReplaceAll(s, "'", "\\'")
	return strings.ReplaceAll(rs, "\"", "\\\"")
}

// whereObject generates the SQL WHERE clause for JSON object fields based on the specified database engine.
// It constructs the appropriate query parts to handle JSON extraction and comparison.
//
// Parameters:
// - engine: The database engine type (MySQL, PostgreSQL, SQLite).
// - k: The key or field name in the database.
// - v: The value to be compared. It is expected to be a map[string]interface{} representing the JSON object.
//
// Returns:
// - The generated SQL WHERE clause as a string.
//
// The function iterates over the map of values and constructs the query parts based on the database engine:
// - For MySQL and SQLite, it uses JSON_EXTRACT to extract and compare JSON values.
// - For PostgreSQL, it uses the jsonb @> operator to check if the JSON object contains the specified key-value pair.
//
// The function handles nested JSON objects by recursively constructing the query parts for each nested key-value pair.
// It also escapes string values to prevent SQL injection.
func whereObject(engine Engine, k string, v interface{}) string {
	queryParts := make([]string, 0)

	// we don't know the type, we handle the rangeValue as a map[string]interface{}
	vJSON, _ := json.Marshal(v) //nolint:errchkjson // this check might break the current code

	var rangeV map[string]interface{}
	_ = json.Unmarshal(vJSON, &rangeV)

	for rangeKey, rangeValue := range rangeV {
		if engine == MySQL || engine == SQLite {
			switch vv := rangeValue.(type) {
			case string:
				rangeValue = "\"" + escapeDBString(rangeValue.(string)) + "\""
				queryParts = append(queryParts, "JSON_EXTRACT("+k+", '$."+rangeKey+"') = "+rangeValue.(string))
			default:
				metadataJSON, _ := json.Marshal(vv) //nolint:errchkjson // this check might break the current code
				var metadata map[string]interface{}
				_ = json.Unmarshal(metadataJSON, &metadata)
				for kk, vvv := range metadata {
					mJSON, _ := json.Marshal(vvv) //nolint:errchkjson // this check might break the current code
					vvv = string(mJSON)
					queryParts = append(queryParts, "JSON_EXTRACT("+k+", '$."+rangeKey+"."+kk+"') = "+vvv.(string))
				}
			}
		} else if engine == PostgreSQL {
			switch vv := rangeValue.(type) {
			case string:
				rangeValue = "\"" + escapeDBString(rangeValue.(string)) + "\""
			default:
				metadataJSON, _ := json.Marshal(vv) //nolint:errchkjson // this check might break the current code
				rangeValue = string(metadataJSON)
			}
			queryParts = append(queryParts, k+"::jsonb @> '{\""+rangeKey+"\":"+rangeValue.(string)+"}'::jsonb")
		} else {
			queryParts = append(queryParts, "JSON_EXTRACT("+k+", '$."+rangeKey+"') = '"+escapeDBString(rangeValue.(string))+"'")
		}
	}

	if len(queryParts) == 0 {
		return ""
	}
	query := queryParts[0]
	if len(queryParts) > 1 {
		query = "(" + strings.Join(queryParts, " AND ") + ")"
	}

	return query
}

// whereSlice generates the SQL WHERE clause for JSON array fields based on the specified database engine.
// It constructs the appropriate query parts to handle JSON array extraction and comparison.
//
// Parameters:
// - engine: The database engine type (MySQL, PostgreSQL, SQLite).
// - k: The key or field name in the database.
// - v: The value to be compared. It is expected to be a string representing the JSON array element.
//
// Returns:
// - The generated SQL WHERE clause as a string.
//
// The function constructs the query parts based on the database engine:
// - For MySQL, it uses JSON_CONTAINS to check if the JSON array contains the specified value.
// - For PostgreSQL, it uses the jsonb @> operator to check if the JSON array contains the specified value.
// - For SQLite, it uses EXISTS with json_each to check if the JSON array contains the specified value.
func whereSlice(engine Engine, k string, v interface{}) string {
	if engine == MySQL {
		return "JSON_CONTAINS(" + k + ", CAST('[\"" + v.(string) + "\"]' AS JSON))"
	} else if engine == PostgreSQL {
		return k + "::jsonb @> '[\"" + v.(string) + "\"]'"
	}
	return "EXISTS (SELECT 1 FROM json_each(" + k + ") WHERE value = \"" + v.(string) + "\")"
}
