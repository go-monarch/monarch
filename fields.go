package monarch

import (
	"context"
	"reflect"
	"slices"
	"strings"
)

type Field struct {
	Name              string
	DBName            string
	FieldType         reflect.Type
	FieldTag          reflect.StructTag
	StructField       reflect.StructField
	IndirectFieldType reflect.Type
	Schema            *Schema
	Index             bool
	ReflectValueOf    func(ctx context.Context, val reflect.Value) reflect.Value
}

func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {
	var (
		tags = ParseTagSetting(fieldStruct.Tag.Get("monarch"), ",")
	)
	if CheckSkip(tags) {
		return nil
	}

	field := &Field{
		Name:              fieldStruct.Name,
		DBName:            tags[0],
		FieldType:         fieldStruct.Type,
		FieldTag:          fieldStruct.Tag,
		IndirectFieldType: fieldStruct.Type,
		Schema:            schema,
		StructField:       fieldStruct,
		Index:             CheckIndex(tags),
	}

	for field.IndirectFieldType.Kind() == reflect.Pointer {
		field.IndirectFieldType = field.IndirectFieldType.Elem()
	}

	return field

}

func (field *Field) setupValuerAndSetter() {

	// ValueOf returns field's value and if it is zero
	fieldIndex := field.StructField.Index[0]

	// ReflectValueOf returns field's reflect value
	switch {
	case len(field.StructField.Index) == 1 && fieldIndex > 0:
		field.ReflectValueOf = func(ctx context.Context, value reflect.Value) reflect.Value {
			return reflect.Indirect(value).Field(fieldIndex)
		}
	default:
		field.ReflectValueOf = func(ctx context.Context, v reflect.Value) reflect.Value {
			v = reflect.Indirect(v)
			for idx, fieldIdx := range field.StructField.Index {
				if fieldIdx >= 0 {
					v = v.Field(fieldIdx)
				} else {
					v = v.Field(-fieldIdx - 1)

					if v.IsNil() {
						v.Set(reflect.New(v.Type().Elem()))
					}

					if idx < len(field.StructField.Index)-1 {
						v = v.Elem()
					}
				}
			}
			return v
		}
	}
}

func ParseTagSetting(tag, seperator string) []string {
	return strings.Split(tag, seperator)
}
func CheckIndex(tag []string) bool {
	return slices.Contains(tag, "index")
}
func CheckSkip(tag []string) bool {
	return slices.Contains(tag, "-")
}
