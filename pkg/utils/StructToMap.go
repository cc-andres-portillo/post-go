package utils

import "encoding/json"

// Convierte un struct a map[string]interface{}
func StructToMap(v interface{}) map[string]interface{} {
	var m map[string]interface{}
	data, _ := json.Marshal(v)
	json.Unmarshal(data, &m)
	return m
}
