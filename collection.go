package monarch

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-monarch/monarch/query"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Collection[T any] struct {
	coll       *mongo.Collection
	cacheStore *sync.Map
	ctx        context.Context
	limit      int64
	offset     int64
	filter     bson.D
	order      bson.D
	Model[T]
}

type Model[T any] interface {
	Query(ctx context.Context, query ...query.Params) Querier[T]
	ExecRaw(ctx context.Context, f func(coll *Collection[T]) error) (*T, error)
	Saver[T]
}

type Querier[T any] interface {
	FindOne() (*T, error)
	FindMany() ([]*T, error)
	UpdateOne(doc T) error
	UpdateMany(doc T) error
	DeleteOne(doc T) error
	DeleteMany(doc T) error
}

type Saver[T any] interface {
	Save(ctx context.Context, doc T) error
}

func RegisterCollection[T any](m *Monarch, schema T) (*Collection[T], error) {
	s, err := Parse(schema, m.cacheStore)
	if err != nil {
		return nil, err
	}

	coll := m.db.Collection(s.Collection)
	if err := registerIndexes(coll, s.Fields); err != nil {
		return nil, err
	}
	c := &Collection[T]{coll: coll, cacheStore: m.cacheStore}

	return c, nil
}

func (c *Collection[T]) Query(ctx context.Context, queries ...query.Params) Querier[T] {
	c.ctx = ctx
	c.filter = bson.D{}
	c.order = bson.D{}
	var q_params []query.QueryStruct

	for _, param := range queries {
		q_params = append(q_params, param())
	}

	for _, qq := range q_params {
		switch qq.Key() {
		case query.QueryFilter:
			val, ok := qq.Value().(query.FilterStruct)
			if !ok {
				panic(errors.New("unsupported"))
			}
			c.filter = append(c.filter, bson.E{Key: val.Key(), Value: val.Value()})
		case query.QuerySort:
			var val int
			order_val, ok := qq.Value().(query.OrderStruct)
			if !ok {
				panic(errors.New("unsupported"))
			}
			switch order_val.Value() {
			case query.ASC:
				val = -1
			case query.DESC:
				val = 1
			default:
				panic(errors.New("unsupported"))
			}
			c.order = append(c.order, bson.E{Key: order_val.Key(), Value: val})
		case query.QueryLimit:
			val, ok := qq.Value().(int64)
			if !ok {
			}
			c.limit = val
		case query.QueryOffset:
			val, ok := qq.Value().(int64)
			if !ok {
			}
			c.offset = val
		default:
			panic(errors.New("unsupported"))
		}
	}
	return c
}

func (c *Collection[T]) Save(ctx context.Context, doc T) error {
	val, err := c.marshal(doc)
	if err != nil {
		return err
	}
	if _, err := c.coll.InsertOne(ctx, val); err != nil {
		return err
	}
	return nil
}

func (c *Collection[T]) FindOne() (*T, error) {
	defer c.reset()
	var single bson.D
	if err := c.coll.FindOne(c.ctx, c.filter).Decode(&single); err != nil {
		return nil, err
	}
	res, err := c.unMarshal(single)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Collection[T]) FindMany() ([]*T, error) {
	defer c.reset()
	var findResult []*T
	result, err := c.coll.Find(c.ctx, c.filter, options.Find().SetLimit(c.limit).
		SetSkip(c.offset).SetSort(c.order))
	if err != nil {
		return nil, err
	}

	defer result.Close(c.ctx)

	for result.Next(c.ctx) {
		var res bson.D

		if err := result.Decode(&res); err != nil {
			return nil, err
		}

		val, err := c.unMarshal(res)
		if err != nil {
			return nil, err
		}
		findResult = append(findResult, val)
	}
	return findResult, nil
}

func (c *Collection[T]) UpdateOne(doc T) error {
	defer c.reset()
	val, err := c.marshal(doc)
	if err != nil {
		return err
	}

	_, err = c.coll.UpdateOne(c.ctx, c.filter, val)
	return err
}
func (c *Collection[T]) UpdateMany(doc T) error {
	defer c.reset()
	val, err := c.marshal(doc)
	if err != nil {
		return err
	}

	_, err = c.coll.UpdateMany(c.ctx, c.filter, val)

	return err
}
func (c *Collection[T]) DeleteOne(doc T) error {
	defer c.reset()
	_, err := c.coll.DeleteOne(c.ctx, c.filter)
	return err
}
func (c *Collection[T]) DeleteMany(doc T) error {
	defer c.reset()
	_, err := c.coll.DeleteMany(c.ctx, c.filter)
	return err
}

func (c *Collection[T]) Collection() *mongo.Collection {
	return c.coll
}

func registerIndexes(coll *mongo.Collection, fields []*Field) error {
	var idx []mongo.IndexModel
	for _, field := range fields {
		if field.Index {
			idx = append(idx, mongo.IndexModel{
				Keys:    bson.D{{Key: field.DBName, Value: 1}},
				Options: options.Index().SetUnique(true),
			})
		}
	}

	if len(idx) < 1 {
		return nil
	}
	_, err := coll.Indexes().CreateMany(context.Background(), idx)
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection[T]) reset() {
	c.ctx = context.Background()
	c.filter = bson.D{}
	c.order = bson.D{}
	c.limit = 0
	c.offset = 0
}

func (c *Collection[T]) marshal(data interface{}) (bson.D, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	schema, err := Parse(data, c.cacheStore)
	if err != nil {
		return nil, err
	}
	var doc = bson.D{}
	value := reflect.ValueOf(data)
	for _, f := range schema.Fields {
		v := f.ReflectValueOf(ctx, value)
		fdata, err := c.encodeValue(v)
		if err != nil {
			return nil, err
		}
		if f.DBName == "" {
			f.DBName = strings.ToLower(f.Name)
		}
		doc = append(doc, bson.E{Key: f.DBName, Value: fdata})
	}

	return doc, nil
}

func (c *Collection[T]) encodeValue(v reflect.Value) (interface{}, error) {
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return int32(v.Int()), nil
	case reflect.Int64:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return uint32(v.Uint()), nil
	case reflect.Uint64:
		return v.Uint(), nil
	case reflect.Float32, reflect.Float64:
		return v.Float(), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.Pointer:
		return c.encodeValue(v.Elem())
	default:
		return v.Interface(), nil
	}
}

func (c *Collection[T]) unMarshal(doc bson.D) (*T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var result T
	s, err := Parse(result, c.cacheStore)
	if err != nil {
		return nil, err
	}
	r := reflect.ValueOf(&result)
	for _, e := range doc {
		if e.Key != "_id" {
			f := s.FieldByDBName[e.Key]
			switch e.Value.(type) {
			case bson.A:
				val := e.Value.(bson.A)
				newVal := reflect.MakeSlice(f.FieldType, 0, 0)
				for _, v := range val {
					newVal = reflect.Append(newVal, reflect.ValueOf(v))
				}
				f.ReflectValueOf(ctx, r).Set(newVal)
			case bson.Binary:
				if val, ok := e.Value.(bson.Binary); ok {
					if val.Subtype == 4 {
						id, err := uuid.FromBytes(val.Data)
						if err != nil {
							return nil, err
						}
						f.ReflectValueOf(ctx, r).Set(reflect.ValueOf(id))
					}
				}
			case bson.ObjectID:
				if val, ok := e.Value.(bson.ObjectID); ok {
					f.ReflectValueOf(ctx, r).Set(reflect.ValueOf(val.Hex()))
				}
			case bson.DateTime:
				if val, ok := e.Value.(bson.DateTime); ok {
					f.ReflectValueOf(ctx, r).Set(reflect.ValueOf(val.Time()))
				}
			case bson.D:
				switch f.FieldType.Kind() {
				case reflect.Map:
					val := e.Value.(bson.D)
					newMap := reflect.MakeMap(f.FieldType)
					for _, v := range val {
						newMap.SetMapIndex(reflect.ValueOf(v.Key), reflect.ValueOf(v.Value))
					}
					f.ReflectValueOf(ctx, r).Set(newMap)
				default:
					continue
				}
			case int, int8, int16, int32, int64:
				var val int64
				switch e.Value.(type) {
				case int:
					val = int64(e.Value.(int))
				case int8:
					val = int64(e.Value.(int8))
				case int16:
					val = int64(e.Value.(int16))
				case int32:
					val = int64(e.Value.(int32))
				default:
					val = int64(e.Value.(int64))
				}
				f.ReflectValueOf(ctx, r).SetInt(val)
			case uint, uint8, uint16, uint32, uint64:
				var val uint64
				switch e.Value.(type) {
				case uint:
					val = uint64(e.Value.(uint))
				case uint8:
					val = uint64(e.Value.(uint8))
				case uint16:
					val = uint64(e.Value.(uint16))
				case uint32:
					val = uint64(e.Value.(uint32))
				default:
					val = uint64(e.Value.(uint64))
				}
				f.ReflectValueOf(ctx, r).SetUint(val)
			case float32, float64:
				var val float64
				switch e.Value.(type) {
				case uint:
					val = float64(e.Value.(float32))
				default:
					val = float64(e.Value.(float64))
				}
				f.ReflectValueOf(ctx, r).SetFloat(val)
			case bool:
				val := e.Value.(bool)
				f.ReflectValueOf(ctx, r).SetBool(val)
			case string:
				val := e.Value.(string)
				f.ReflectValueOf(ctx, r).SetString(val)
			default:
				f.ReflectValueOf(ctx, r).Set(reflect.ValueOf(e.Value))
			}

		}
	}
	return &result, nil
}
