package monarch

import (
	"reflect"
	"slices"
	"strings"
)

type Field struct {
	Name        string
	DBName      string
	FieldType   reflect.Type
	FieldValue  reflect.Value
	FieldTag    reflect.StructTag
	StructField reflect.StructField
	Schema      *Schema
	Index       bool
}

func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {
	var (
		tags = ParseTagSetting(fieldStruct.Tag.Get("monarch"), ",")
	)

	field := &Field{
		Name:        fieldStruct.Name,
		DBName:      tags[0],
		FieldType:   fieldStruct.Type,
		FieldValue:  reflect.ValueOf(fieldStruct),
		FieldTag:    fieldStruct.Tag,
		Schema:      schema,
		StructField: fieldStruct,
		Index:       CheckIndex(tags),
	}

	return field

}

func ParseTagSetting(tag, seperator string) []string {
	return strings.Split(tag, seperator)
}
func CheckIndex(tag []string) bool {
	return slices.Contains(tag, "index")
}
