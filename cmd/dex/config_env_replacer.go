package main

import (
	"reflect"
)

// Replace keys of the form $FOO with
func replaceEnvKeys(data interface{}, getenv func(string) string) error {
	val := reflect.ValueOf(data)

	// Elem() only works on interfaces and pointers. We probably only want Ptr's, tho...
	if val.Kind() != reflect.Interface && val.Kind() != reflect.Ptr {
		return nil
	}

	s := val.Elem()

	// Skip things we cannot modify
	if !s.CanSet() {
		return nil
	}

	// Convert strings if they start with '$'
	if s.Kind() == reflect.String {
		value := s.Interface().(string)
		if len(value) > 2 && string(value[0]) == "$" {
			s.SetString(getenv(value[1:]))
		}
		return nil
	}

	// Structs
	if s.Kind() == reflect.Struct {
		for i := 0; i < s.NumField(); i += 1 {
			f := s.Field(i)

			// Recurse through fields
			err := replaceEnvKeys(f.Addr().Interface(), getenv)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// TODO: Treat slices of chars as a string
	if s.Kind() == reflect.Slice {
		for i := 0; i < s.Len(); i += 1 {
			err := replaceEnvKeys(s.Index(i).Addr().Interface(), getenv)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// TODO: Maps, Arrays

	return nil
}
