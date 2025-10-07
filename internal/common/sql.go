package common

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// get value each fields from entities
func GetFieldValues(i interface{}) ([]interface{}, error) {
	entities := reflect.ValueOf(i)
	if entities.Kind() != reflect.Struct {
		return nil, errors.New("invalid entity for get field values")
	}

	values := make([]interface{}, entities.NumField())
	for i := 0; i < entities.NumField(); i++ {
		v := entities.Field(i).Interface()
		values[i] = v
	}
	return values, nil
}

// Function for replacing ? with $n for postgres
func ReplaceSQL(old, searchPattern string) string {
	tmpCount := strings.Count(old, searchPattern)
	for m := 1; m <= tmpCount; m++ {
		old = strings.Replace(old, searchPattern, "$"+strconv.Itoa(m), 1)
	}
	return old
}
