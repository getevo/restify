package restify

import (
	"errors"
	"reflect"
)

// KeyValue represents a key-value pair with a generic value type.
type KeyValue[T any] struct {
	Key   string `json:"key,omitempty"`
	Value T      `json:"value,omitempty"`
}

// Dictionary represents a collection of key-value pairs with a generic value type.
type Dictionary[T any] []KeyValue[T]

// Has checks if a key exists in the dictionary.
func (d *Dictionary[T]) Has(key string) bool {
	for _, kv := range *d {
		if kv.Key == key {
			return true
		}
	}
	return false
}

// ContainsValue checks if a value exists in the dictionary and returns the KeyValue if found.
func (d *Dictionary[T]) ContainsValue(v T) (bool, KeyValue[T]) {
	for _, kv := range *d {
		if reflect.DeepEqual(kv.Value, v) {
			return true, kv
		}
	}
	return false, KeyValue[T]{}
}

// Set sets a key to a value in the dictionary, updating if it already exists.
func (d *Dictionary[T]) Set(key string, v T) {
	for i, kv := range *d {
		if kv.Key == key {
			(*d)[i].Value = v
			return
		}
	}
	*d = append(*d, KeyValue[T]{Key: key, Value: v})
}

// Delete removes a key (and its associated value) from the dictionary.
func (d *Dictionary[T]) Delete(key string) error {
	for i, kv := range *d {
		if kv.Key == key {
			*d = append((*d)[:i], (*d)[i+1:]...)
			return nil
		}
	}
	return errors.New("key not found")
}
