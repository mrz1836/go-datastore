package datastore

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModel struct {
	ID string `json:"id"`
}

type testStruct struct {
	ID           string `json:"id" toml:"id" yaml:"hash" bson:"_id"`
	CurrentValue uint64 `json:"current_value" toml:"current_value" yaml:"current_value" bson:"current_value"`
	InternalNum  uint32 `json:"internal_num" toml:"internal_num" yaml:"internal_num" bson:"internal_num"`
	ExternalNum  uint32 `json:"external_num" toml:"external_num" yaml:"external_num" bson:"external_num"`
}

const (
	objectMetadataField = "object_metadata"
	fieldInIDs          = "field_in_ids"
	fieldOutIDs         = "field_out_ids"
)

// TestClient_getFieldNames will test the method getFieldNames()
func TestClient_getFieldNames(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		fields := getFieldNames(nil)
		assert.Empty(t, fields)
		assert.Equal(t, []string{}, fields)
	})

	t.Run("normal struct", func(t *testing.T) {
		s := testStruct{}
		fields := getFieldNames(s)
		assert.Len(t, fields, 4)
		assert.Equal(t, []string{mongoIDField, "current_value", "internal_num", "external_num"}, fields)
	})

	t.Run("pointer struct", func(t *testing.T) {
		s := &testStruct{}
		fields := getFieldNames(s)
		assert.Len(t, fields, 4)
		assert.Equal(t, []string{mongoIDField, "current_value", "internal_num", "external_num"}, fields)
	})

	t.Run("slice of structs", func(t *testing.T) {
		s := []testStruct{}
		fields := getFieldNames(s)
		assert.Len(t, fields, 4)
		assert.Equal(t, []string{mongoIDField, "current_value", "internal_num", "external_num"}, fields)
	})

	t.Run("pointer of slice of structs", func(t *testing.T) {
		s := &[]testStruct{}
		fields := getFieldNames(s)
		assert.Len(t, fields, 4)
		assert.Equal(t, []string{mongoIDField, "current_value", "internal_num", "external_num"}, fields)
	})

	t.Run("pointer of slice of pointers of structs", func(t *testing.T) {
		s := &[]*testStruct{}
		fields := getFieldNames(s)
		assert.Len(t, fields, 4)
		assert.Equal(t, []string{mongoIDField, "current_value", "internal_num", "external_num"}, fields)
	})
}

// TestClient_getMongoQueryConditions will test the method getMongoQueryConditions()
func TestClient_getMongoQueryConditions(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		condition := map[string]any{}
		queryConditions := getMongoQueryConditions(Transaction{}, condition, nil)
		assert.Equal(t, map[string]any{}, queryConditions)
	})

	t.Run("simple", func(t *testing.T) {
		condition := map[string]any{
			"test-key": "test-value",
		}
		queryConditions := getMongoQueryConditions(Transaction{}, condition, nil)
		assert.Equal(t, map[string]any{"test-key": "test-value"}, queryConditions)
	})

	t.Run("test "+sqlIDFieldProper, func(t *testing.T) {
		condition := map[string]any{}
		queryConditions := getMongoQueryConditions(mockModel{
			ID: "identifier",
		}, condition, nil)
		assert.Equal(t, map[string]any{mongoIDField: "identifier"}, queryConditions)
	})

	t.Run(conditionOr+" "+sqlIDFieldProper, func(t *testing.T) {
		condition := map[string]any{
			conditionOr: []map[string]any{{
				sqlIDField: "test-key",
			}},
		}
		queryConditions := getMongoQueryConditions(nil, condition, nil)
		expected := map[string]any{
			conditionOr: []map[string]any{{
				mongoIDField: "test-key",
			}},
		}
		assert.Equal(t, expected, queryConditions)
	})

	t.Run(conditionAnd+" "+conditionOr+" "+sqlIDFieldProper, func(t *testing.T) {
		condition := map[string]any{
			metadataField: map[string]any{},
			conditionAnd: []map[string]any{{
				conditionOr: []map[string]any{{
					sqlIDField: "test-key",
				}},
			}},
		}
		queryConditions := getMongoQueryConditions(nil, condition, nil)
		expected := map[string]any{
			conditionAnd: []map[string]any{{
				conditionOr: []map[string]any{{
					mongoIDField: "test-key",
				}},
			}},
		}
		assert.Equal(t, expected, queryConditions)
	})

	t.Run("embedded "+sqlIDFieldProper, func(t *testing.T) {
		condition := map[string]any{
			conditionAnd: []map[string]any{{
				metadataField: map[string]any{
					"test-key": "test-value",
				},
			}, {
				sqlIDField: "identifier",
			}},
		}
		expected := map[string]any{
			conditionAnd: []map[string]any{{
				conditionAnd: []map[string]any{{
					metadataField + ".k": "test-key", metadataField + ".v": "test-value",
				}},
			}, {
				mongoIDField: "identifier",
			}},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, nil)
		assert.Equal(t, expected, queryConditions)
	})

	t.Run(metadataField, func(t *testing.T) {
		condition := map[string]any{
			metadataField: map[string]any{
				"test-key": "test-value",
			},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, nil)
		expected := map[string]any{
			conditionAnd: []map[string]any{{
				metadataField + ".k": "test-key",
				metadataField + ".v": "test-value",
			}},
		}
		assert.Equal(t, expected, queryConditions)
	})

	t.Run(metadataField+" test 2", func(t *testing.T) {
		condition := map[string]any{
			metadataField: map[string]any{
				"test-key":  "test-value",
				"test-key2": "test-value2",
			},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, nil)
		expected := []map[string]any{{
			metadataField + ".k": "test-key",
			metadataField + ".v": "test-value",
		}, {
			metadataField + ".k": "test-key2",
			metadataField + ".v": "test-value2",
		}}
		assert.Len(t, queryConditions[conditionAnd], 2)
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[0])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[1])
	})

	t.Run(metadataField+" test 3", func(t *testing.T) {
		condition := map[string]any{
			metadataField: map[string]any{
				"test-key":  "test-value",
				"test-key2": "test-value2",
			},
			conditionAnd: []map[string]any{{
				"amount": map[string]any{
					conditionLessThan: 98,
				},
			}},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, nil)
		expected := []map[string]any{{
			metadataField + ".k": "test-key",
			metadataField + ".v": "test-value",
		}, {
			metadataField + ".k": "test-key2",
			metadataField + ".v": "test-value2",
		}, {
			"amount": map[string]any{
				conditionLessThan: float64(98),
			},
		}}
		assert.Len(t, queryConditions[conditionAnd], 3)
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[0])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[1])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[2])
	})

	t.Run(metadataField+" "+conditionOr, func(t *testing.T) {
		condition := map[string]any{
			metadataField: map[string]any{
				"test-key":  "test-value",
				"test-key2": "test-value2",
			},
			conditionOr: []map[string]any{{
				"amount": map[string]any{
					conditionLessThan: 98,
				},
			}},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, nil)
		expected := []map[string]any{{
			metadataField + ".k": "test-key",
			metadataField + ".v": "test-value",
		}, {
			metadataField + ".k": "test-key2",
			metadataField + ".v": "test-value2",
		}}
		expectedOr := []map[string]any{{
			"amount": map[string]any{
				conditionLessThan: float64(98),
			},
		}}
		assert.Len(t, queryConditions[conditionAnd], 2)
		assert.Len(t, queryConditions[conditionOr], 1)
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[0])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[1])
		assert.Contains(t, expectedOr, queryConditions[conditionOr].([]map[string]any)[0])
	})

	t.Run("testing "+objectMetadataField, func(t *testing.T) {
		condition := map[string]any{
			objectMetadataField: map[string]any{
				"testID": map[string]any{
					"test-key": "test-value",
				},
			},
			conditionAnd: []map[string]any{{
				"amount": map[string]any{
					conditionLessThan: 98,
				},
			}},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, processObjectMetadataConditions)
		expected := []map[string]any{{
			objectMetadataField + ".x": "testID",
			objectMetadataField + ".k": "test-key",
			objectMetadataField + ".v": "test-value",
		}, {
			"amount": map[string]any{
				conditionLessThan: float64(98),
			},
		}}
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[0])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[1])
	})

	t.Run("testing "+objectMetadataField+" x2", func(t *testing.T) {
		condition := map[string]any{
			objectMetadataField: map[string]any{
				"testID": map[string]any{
					"test-key":  "test-value",
					"test-key2": "test-value2",
				},
			},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, processObjectMetadataConditions)
		expected := []map[string]any{{
			objectMetadataField + ".x": "testID",
			objectMetadataField + ".k": "test-key",
			objectMetadataField + ".v": "test-value",
		}, {
			objectMetadataField + ".x": "testID",
			objectMetadataField + ".k": "test-key2",
			objectMetadataField + ".v": "test-value2",
		}}
		assert.Len(t, queryConditions[conditionAnd], 2)
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[0])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[1])
	})

	t.Run("testing json interface", func(t *testing.T) {
		condition := map[string]any{
			objectMetadataField: map[string]any{
				"testID": map[string]any{
					"test-key":  "test-value",
					"test-key2": "test-value2",
				},
			},
		}
		c, err := json.Marshal(condition)
		require.NoError(t, err)

		var cc any
		err = json.Unmarshal(c, &cc)
		require.NoError(t, err)
		queryConditions := getMongoQueryConditions(mockModel{}, cc.(map[string]any), processObjectMetadataConditions)
		expected := []map[string]any{{
			objectMetadataField + ".x": "testID",
			objectMetadataField + ".k": "test-key",
			objectMetadataField + ".v": "test-value",
		}, {
			objectMetadataField + ".x": "testID",
			objectMetadataField + ".k": "test-key2",
			objectMetadataField + ".v": "test-value2",
		}}
		assert.Len(t, queryConditions[conditionAnd], 2)
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[0])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[1])
	})

	t.Run("testing "+objectMetadataField+" x3", func(t *testing.T) {
		arrayName1 := fieldInIDs
		arrayName2 := fieldOutIDs
		condition := map[string]any{
			conditionOr: []map[string]any{{
				arrayName1: "test_id",
			}, {
				arrayName2: "test_id",
			}},
			conditionAnd: []map[string]any{{
				conditionOr: []map[string]any{{
					metadataField: map[string]any{"test-key": "test-value"},
				}, {
					objectMetadataField: map[string]any{
						"test_id": map[string]any{"test-key": "test-value"},
					},
				}},
			}},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, processObjectMetadataConditions)
		// {"$and":[{"$or":[{"$and":[{"metadata.k":"test-key","metadata.v":"test-value"}]},{"$and":[{"object_metadata.k":"test-key","object_metadata.v":"test-value"}],"object_metadata.x":"test_id"}]}],"$or":[{"field_in_ids":"test_id"},{"field_out_ids":"test_id"}]}
		assert.Len(t, queryConditions[conditionAnd], 1)
		assert.Len(t, queryConditions[conditionOr], 2)

		expectedXpubID := []map[string]any{{
			arrayName1: "test_id",
		}, {
			arrayName2: "test_id",
		}}
		assert.Contains(t, expectedXpubID, queryConditions[conditionOr].([]map[string]any)[0])
		assert.Contains(t, expectedXpubID, queryConditions[conditionOr].([]map[string]any)[1])

		expected0 := map[string]any{
			metadataField + ".k": "test-key",
			metadataField + ".v": "test-value",
		}
		expected1 := map[string]any{
			objectMetadataField + ".x": "test_id",
			objectMetadataField + ".k": "test-key",
			objectMetadataField + ".v": "test-value",
		}
		or := (queryConditions[conditionAnd].([]map[string]any)[0])[conditionOr]
		or0 := or.([]map[string]any)[0]
		or1 := or.([]map[string]any)[1]
		assert.Equal(t, expected0, or0[conditionAnd].([]map[string]any)[0])
		assert.Equal(t, expected1, or1[conditionAnd].([]map[string]any)[0])
	})

	t.Run("object_output_value", func(t *testing.T) {
		fieldName := "object_output_value"
		condition := map[string]any{
			fieldName: map[string]any{
				"testID": map[string]any{
					conditionGreaterThan: 0,
				},
			},
			conditionAnd: []map[string]any{{
				"amount": map[string]any{
					conditionLessThan: 98,
				},
			}},
		}
		queryConditions := getMongoQueryConditions(mockModel{}, condition, processObjectOutputValueConditions)
		expected := []map[string]any{{
			fieldName + ".testID": map[string]any{
				conditionGreaterThan: float64(0),
			},
		}, {
			"amount": map[string]any{
				conditionLessThan: float64(98),
			},
		}}
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[0])
		assert.Contains(t, expected, queryConditions[conditionAnd].([]map[string]any)[1])
	})
}

// processObjectMetadataConditions is an example of processing custom object metadata
// ObjectID -> Key/Value
func processObjectMetadataConditions(conditions *map[string]any) {
	// marshal / unmarshal into standard map[string]any
	m, _ := json.Marshal((*conditions)[objectMetadataField]) //nolint:errchkjson // this check might break the current code
	var r map[string]any
	_ = json.Unmarshal(m, &r)

	for object, xr := range r {
		objectMetadata := make([]map[string]any, 0)
		for key, value := range xr.(map[string]any) {
			objectMetadata = append(objectMetadata, map[string]any{
				objectMetadataField + ".x": object,
				objectMetadataField + ".k": key,
				objectMetadataField + ".v": value,
			})
		}
		if len(objectMetadata) > 0 {
			_, ok := (*conditions)[conditionAnd]
			if ok {
				and := (*conditions)[conditionAnd].([]map[string]any)
				and = append(and, objectMetadata...)
				(*conditions)[conditionAnd] = and
			} else {
				(*conditions)[conditionAnd] = objectMetadata
			}
		}
	}
	delete(*conditions, objectMetadataField)
}

// processObjectOutputValueConditions is an example of processing custom object value
// ObjectID -> Value
func processObjectOutputValueConditions(conditions *map[string]any) {
	fieldName := "object_output_value"

	m, _ := json.Marshal((*conditions)[fieldName]) //nolint:errchkjson // this check might break the current code
	var r map[string]any
	_ = json.Unmarshal(m, &r)

	objectOutputValue := make([]map[string]any, 0)
	for object, value := range r {
		outputKey := fieldName + "." + object
		objectOutputValue = append(objectOutputValue, map[string]any{
			outputKey: value,
		})
	}
	if len(objectOutputValue) > 0 {
		_, ok := (*conditions)[conditionAnd]
		if ok {
			and := (*conditions)[conditionAnd].([]map[string]any)
			and = append(and, objectOutputValue...)
			(*conditions)[conditionAnd] = and
		} else {
			(*conditions)[conditionAnd] = objectOutputValue
		}
	}

	delete(*conditions, fieldName)
}
