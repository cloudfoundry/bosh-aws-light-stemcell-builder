package table

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func Marshall(rawInput string, target interface{}) error {
	targetPtr := reflect.ValueOf(target)
	if targetPtr.Kind() != reflect.Ptr {
		return fmt.Errorf("Expected marshall target to be pointer but was %s", targetPtr.Kind())
	}

	targetValue := targetPtr.Elem()
	targetType := targetValue.Type()

	input, err := parseInput(rawInput)
	if err != nil {
		return err
	}

	for i := 0; i < targetValue.NumField(); i++ {
		structField := targetType.Field(i)

		targetField := targetValue.Field(i)
		key := structField.Tag.Get("key")
		if len(key) == 0 {
			return errors.New("Expected to find `key` in struct tag")
		}

		rawPosition := structField.Tag.Get("position")
		if len(rawPosition) == 0 {
			return errors.New("Expected to find `position` in struct tag")
		}
		position, err := strconv.Atoi(rawPosition)
		if err != nil {
			return fmt.Errorf("Failed to parse `position` value: %s", err.Error())
		}

		fieldsForKey, foundKey := input[key]
		if foundKey == false {
			continue
		}

		if position >= len(fieldsForKey) {
			return fmt.Errorf("Position `%d` is out of range for fields", position)
		}

		newValue := fieldsForKey[position]
		targetField.Set(reflect.ValueOf(newValue))
	}
	return nil
}

func parseInput(rawInputString string) (map[string][]string, error) {
	result := map[string][]string{}

	lines := strings.Split(rawInputString, "\n")
	for _, line := range lines {
		if strings.ContainsAny(line, "\t") == false {
			return nil, fmt.Errorf("Expected fields to be tab separated in input: %s", line)
		}

		fields := strings.Split(line, "\t")
		key := fields[0]
		result[key] = fields[1:]
	}

	return result, nil
}
