package utils

import (
	"go.mongodb.org/mongo-driver/bson"
)

type FlexibleStrings []string

// UnmarshalBSON permite que el campo acepte distintos formatos en MongoDB
func (fs *FlexibleStrings) UnmarshalBSON(data []byte) error {
	var str string
	if err := bson.Unmarshal(data, &str); err == nil {
		*fs = []string{str}
		return nil
	}

	var arr []string
	if err := bson.Unmarshal(data, &arr); err == nil {
		*fs = arr
		return nil
	}

	var objArr []map[string]interface{}
	if err := bson.Unmarshal(data, &objArr); err == nil {
		for _, obj := range objArr {
			if url, ok := obj["url"].(string); ok {
				*fs = append(*fs, url)
			}
		}
		return nil
	}

	*fs = []string{}
	return nil
}
