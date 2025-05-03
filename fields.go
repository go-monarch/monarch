package monarch

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
)

var embeddedCacheKey = "embedded_cache_store"

type Field struct {
	Name              string
	DBName            string
	FieldType         reflect.Type
	FieldTag          reflect.StructTag
	StructField       reflect.StructField
	IndirectFieldType reflect.Type
	Schema            *Schema
	EmbeddedSchema    *Schema
	Index             bool
	ReflectValueOf    func(ctx context.Context, val reflect.Value) reflect.Value
}

func (schema *Schema) parseField(fieldStruct reflect.StructField) *Field {
	var (
		tags = parseTagSetting(fieldStruct.Tag.Get("monarch"), ",")
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

	fieldValue := reflect.New(field.IndirectFieldType)

	for field.IndirectFieldType.Kind() == reflect.Pointer {
		field.IndirectFieldType = field.IndirectFieldType.Elem()
	}

	if fieldStruct.Anonymous {
		kind := reflect.Indirect(fieldValue).Kind()
		switch kind {
		case reflect.Struct:
			var err error
			cacheStore := &sync.Map{}
			cacheStore.Store(embeddedCacheKey, true)
			if field.EmbeddedSchema, err = getOrParse(fieldValue.Interface(), cacheStore); err != nil {
				schema.err = err
			}

			for _, ef := range field.EmbeddedSchema.Fields {
				ef.Schema = schema

				// index is negative means is pointer
				if field.FieldType.Kind() == reflect.Struct {
					ef.StructField.Index = append([]int{fieldStruct.Index[0]}, ef.StructField.Index...)
				} else {
					ef.StructField.Index = append([]int{-fieldStruct.Index[0] - 1}, ef.StructField.Index...)
				}

			}
		case reflect.Invalid, reflect.Uintptr, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
			reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
			schema.err = fmt.Errorf("invalid embedded struct for %s's field %s, should be struct, but got %v", field.Schema.Name, field.Name, field.FieldType)
		}
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

func parseTagSetting(tag, seperator string) []string {
	return strings.Split(tag, seperator)
}
func CheckIndex(tag []string) bool {
	return slices.Contains(tag, "index")
}
func CheckSkip(tag []string) bool {
	return slices.Contains(tag, "-")
}
