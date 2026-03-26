package model

import (
	"encoding/json"
	"os"
)

// Schema represents an Avro record schema.
type Schema struct {
	Type      string  `json:"type"`
	Name      string  `json:"name"`
	Namespace string  `json:"namespace,omitempty"`
	Doc       string  `json:"doc,omitempty"`
	Fields    []Field `json:"fields,omitempty"`
}

// Field represents a field in an Avro record.
type Field struct {
	Name       string
	Type       interface{}
	Default    interface{}
	HasDefault bool
	Doc        string
	Aliases    []string
}

// UnmarshalJSON distinguishes between absent default and explicit "default": null.
func (f *Field) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["name"]; ok {
		if err := json.Unmarshal(v, &f.Name); err != nil {
			return err
		}
	}
	if v, ok := raw["type"]; ok {
		if err := json.Unmarshal(v, &f.Type); err != nil {
			return err
		}
	}
	if v, ok := raw["doc"]; ok {
		if err := json.Unmarshal(v, &f.Doc); err != nil {
			return err
		}
	}
	if v, ok := raw["aliases"]; ok {
		if err := json.Unmarshal(v, &f.Aliases); err != nil {
			return err
		}
	}
	if v, ok := raw["default"]; ok {
		f.HasDefault = true
		if err := json.Unmarshal(v, &f.Default); err != nil {
			return err
		}
	}
	return nil
}

// EnumSchema represents an Avro enum type.
type EnumSchema struct {
	Type      string   `json:"type"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace,omitempty"`
	Symbols   []string `json:"symbols"`
	Default   string   `json:"default,omitempty"`
}

func ReadSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSchema(data)
}

func ParseSchema(data []byte) (*Schema, error) {
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
