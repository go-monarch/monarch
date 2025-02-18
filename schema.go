package monarch

import (
	"errors"
	"go/ast"
	"reflect"
	"sync"

	"github.com/jinzhu/inflection"
)

type Schema struct {
	Name          string
	SchemaType    reflect.Type
	Collection    string
	Fields        []*Field
	FieldByName   map[string]*Field
	FieldByDBName map[string]*Field
	IndexField    map[string]*Field
	InlineField   map[string]*Field

	cacheStore *sync.Map
	err        error
	loaded     chan struct{}
}

func Parse(obj interface{}, cacheStore *sync.Map) (*Schema, error) {

	if obj == nil {
		return nil, errors.New("err: unexpected type")
	}

	value := reflect.ValueOf(obj)

	if value.Kind() == reflect.Pointer && value.IsNil() {
		value = reflect.New(value.Type().Elem())
	}
	schemaType := reflect.Indirect(value).Type()

	if schemaType.Kind() == reflect.Interface {
		schemaType = reflect.Indirect(reflect.ValueOf(obj)).Elem().Type()
	}

	if schemaType.Kind() != reflect.Struct {
		if schemaType.PkgPath() == "" {
			return nil, errors.New("")
		}
		return nil, errors.New("")
	}

	if v, ok := cacheStore.Load(schemaType); ok {
		s := v.(*Schema)

		<-s.loaded
		return s, s.err
	}

	schema := &Schema{
		Name:          schemaType.Name(),
		SchemaType:    schemaType,
		Collection:    inflection.Plural(toSnakeCase(schemaType.Name())),
		Fields:        make([]*Field, 0),
		FieldByName:   make(map[string]*Field),
		FieldByDBName: make(map[string]*Field),
		IndexField:    make(map[string]*Field),
		cacheStore:    cacheStore,
		loaded:        make(chan struct{}),
	}

	defer close(schema.loaded)

	if v, ok := cacheStore.Load(schemaType); ok {
		s := v.(*Schema)

		<-s.loaded
		return s, s.err
	}

	for i := 0; i < schemaType.NumField(); i++ {
		if fieldStruct := schemaType.Field(i); ast.IsExported(fieldStruct.Name) {
			if field := schema.ParseField(fieldStruct); field != nil {
				schema.Fields = append(schema.Fields, field)
			}
		}
	}
	for _, field := range schema.Fields {
		if field.Index {
			schema.IndexField[field.Name] = field
		}
		if field.DBName != "" {
			schema.FieldByDBName[field.DBName] = field
		}
		schema.FieldByName[field.Name] = field
		field.setupValuerAndSetter()
	}

	if v, ok := cacheStore.LoadOrStore(schemaType, schema); ok {
		s := v.(*Schema)

		<-s.loaded
		return s, s.err
	}

	defer func() {
		if schema.err != nil {
			cacheStore.Delete(schemaType)
		}
	}()

	return schema, schema.err
}
