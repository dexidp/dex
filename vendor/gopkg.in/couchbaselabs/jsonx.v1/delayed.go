package jsonx

import (
	"encoding/json"
	"errors"
)

var ErrKeyNotFound = errors.New("The key specified does not exist.")

type DelayedObject struct {
	data map[string]*json.RawMessage
}

func (o *DelayedObject) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &o.data)
}

func (o DelayedObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.data)
}

func (o *DelayedObject) Has(key string) bool {
	if o.data == nil {
		return false
	}
	_, ok := o.data[key]
	return ok
}

func (o *DelayedObject) Get(key string, val interface{}) error {
	if o.data == nil {
		return ErrKeyNotFound
	}
	bytes, ok := o.data[key]
	if !ok {
		return ErrKeyNotFound
	}
	return json.Unmarshal(*bytes, val)
}

func (o *DelayedObject) Set(key string, val interface{}) error {
	if o.data == nil {
		o.data = make(map[string]*json.RawMessage)
	}

	switch valTyped := val.(type) {
	case *json.RawMessage:
		o.data[key] = valTyped
		return nil
	case json.RawMessage:
		o.data[key] = &valTyped
		return nil
	}

	valBytes, err := json.Marshal(val)
	if err != nil {
		return err
	}
	valRaw := json.RawMessage(valBytes)
	o.data[key] = &valRaw
	return nil
}
