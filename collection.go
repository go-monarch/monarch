package monarch

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Collection[T any] struct {
	coll       *mongo.Collection
	cacheStore *sync.Map
	Model[T]
}

type Model[T any] interface {
	ExecRaw(ctx context.Context, f func(coll *Collection[T]) error) (*T, error)
	CreateIndex(ctx context.Context) error
	FindOne(ctx context.Context, query ...QueryOptions) (*T, error)
	FindMany(ctx context.Context, query ...QueryOptions) ([]*T, error)
	UpdateOne(ctx context.Context, query ...QueryOptions) error
	UpdateMany(ctx context.Context, query ...QueryOptions) error
	DeleteOne(ctx context.Context, query ...QueryOptions) error
	DeleteMany(ctx context.Context, query ...QueryOptions) error
	Save(ctx context.Context, query ...QueryOptions) error
}

func RegisterCollection[T any](m *Monarch, schema T) (*Collection[T], error) {
	s, err := parse(schema, m.cacheStore)
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

func (c *Collection[T]) Save(ctx context.Context, query ...QueryOptions) error {
	cfg := &querier{}
	for _, q := range query {
		if err := q(cfg); err != nil {
			return err
		}
	}
	if _, ok := cfg.data.(T); !ok {
		return errors.New("invalid type provided")
	}
	val, err := c.marshal(ctx, cfg.data)
	if err != nil {
		return err
	}
	if _, err := c.coll.InsertOne(ctx, val); err != nil {
		return err
	}
	return nil
}

func (c *Collection[T]) FindOne(ctx context.Context, query ...QueryOptions) (*T, error) {
	cfg := &querier{
		filter: make(bson.D, 0),
	}
	for _, q := range query {
		if err := q(cfg); err != nil {
			return nil, err
		}
	}

	var single bson.D
	if err := c.coll.FindOne(ctx, cfg.filter).Decode(&single); err != nil {
		return nil, err
	}
	res, err := c.unMarshal(single)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *Collection[T]) FindMany(ctx context.Context, query ...QueryOptions) ([]*T, error) {
	cfg := &querier{
		filter: make(bson.D, 0),
		order:  make(bson.D, 0),
		limit:  0,
		offset: 0,
	}
	for _, q := range query {
		if err := q(cfg); err != nil {
			return nil, err
		}
	}

	var findResult []*T
	result, err := c.coll.Find(ctx, cfg.filter, options.Find().SetLimit(cfg.limit).
		SetSkip(cfg.offset).SetSort(cfg.order))
	if err != nil {
		return nil, err
	}

	defer result.Close(ctx)

	for result.Next(ctx) {
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

func (c *Collection[T]) UpdateOne(ctx context.Context, query ...QueryOptions) error {
	cfg := &querier{
		filter: make(bson.D, 0),
		order:  make(bson.D, 0),
		limit:  0,
		offset: 0,
	}
	for _, q := range query {
		if err := q(cfg); err != nil {
			return err
		}
	}
	val, err := c.marshal(ctx, cfg.data)
	if err != nil {
		return err
	}

	_, err = c.coll.UpdateOne(ctx, cfg.filter, bson.D{{Key: "$set", Value: val}})
	return err
}
func (c *Collection[T]) UpdateMany(ctx context.Context, query ...QueryOptions) error {
	cfg := &querier{
		filter: make(bson.D, 0),
		order:  make(bson.D, 0),
		limit:  0,
		offset: 0,
	}
	for _, q := range query {
		if err := q(cfg); err != nil {
			return err
		}
	}
	if cfg.data == nil {
		return errors.New("no file provided")
	}
	val, err := c.marshal(ctx, cfg.data)
	if err != nil {
		return err
	}

	_, err = c.coll.UpdateMany(ctx, cfg.filter, bson.D{{Key: "$set", Value: val}})

	return err
}
func (c *Collection[T]) DeleteOne(ctx context.Context, query ...QueryOptions) error {
	cfg := &querier{
		filter: make(bson.D, 0),
		order:  make(bson.D, 0),
		limit:  0,
		offset: 0,
	}
	for _, q := range query {
		if err := q(cfg); err != nil {
			return err
		}
	}
	_, err := c.coll.DeleteOne(ctx, cfg.filter)
	return err
}
func (c *Collection[T]) DeleteMany(ctx context.Context, query ...QueryOptions) error {
	cfg := &querier{
		filter: make(bson.D, 0),
		order:  make(bson.D, 0),
		limit:  0,
		offset: 0,
	}
	for _, q := range query {
		if err := q(cfg); err != nil {
			return err
		}
	}
	_, err := c.coll.DeleteMany(ctx, cfg.filter)
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

func (c *Collection[T]) marshal(ctx context.Context, data any) (bson.D, error) {
	schema, err := parse(data, c.cacheStore)
	if err != nil {
		return nil, err
	}
	var doc = bson.D{}
	value := reflect.ValueOf(data)
	for _, f := range schema.Fields {
		v := f.ReflectValueOf(ctx, value)

		switch f.FieldType.Kind() {
		case reflect.Struct:
			switch v.Interface().(type) {
			case time.Time, *time.Time:
				fdata, err := c.encodeValue(v)
				if err != nil {
					return nil, err
				}
				doc = append(doc, bson.E{Key: f.DBName, Value: fdata})
			default:
				fdata, err := c.marshal(ctx, v.Interface())
				if err != nil {
					return nil, err
				}
				doc = append(doc, bson.E{Key: f.DBName, Value: fdata})
			}
		case reflect.Slice, reflect.Array:
			switch v.Type().Elem().Kind() {
			case reflect.Struct:
				var arr bson.A
				for i := range v.Len() {
					elem := v.Index(i)
					var fdata interface{}
					switch elem.Interface().(type) {
					case time.Time, *time.Time:
						fdata, err = c.encodeValue(elem)
						if err != nil {
							return nil, err
						}
					default:
						fdata, err = c.marshal(ctx, elem.Interface())
						if err != nil {
							return nil, err
						}
					}
					arr = append(arr, fdata)
				}
				doc = append(doc, bson.E{Key: f.DBName, Value: arr})
			default:
				fdata, err := c.encodeValue(v)
				if err != nil {
					return nil, err
				}
				doc = append(doc, bson.E{Key: f.DBName, Value: fdata})
			}
		case reflect.Map:
			fdata, err := c.encodeValue(v)
			if err != nil {
				return nil, err
			}
			doc = append(doc, bson.E{Key: f.DBName, Value: fdata})
		default:
			fdata, err := c.encodeValue(v)
			if err != nil {
				return nil, err
			}
			doc = append(doc, bson.E{Key: f.DBName, Value: fdata})
		}

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
	s, err := parse(result, c.cacheStore)
	if err != nil {
		return nil, err
	}
	r := reflect.ValueOf(&result)
	for _, e := range doc {
		if err := decodeValue(ctx, r, e, s.Fields, c.cacheStore); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

func decodeValue(ctx context.Context, r reflect.Value, e bson.E, fields []*Field, c *sync.Map) error {
	for _, f := range fields {
		if e.Key == f.DBName {
			switch e.Value.(type) {
			case bson.A:
				val := e.Value.(bson.A)
				newVal := reflect.MakeSlice(f.FieldType, 0, 0)
				switch f.FieldType.Elem().Kind() {
				case reflect.Struct:
					for _, v := range val {
						t := reflect.New(f.FieldType.Elem())
						h, err := parse(t.Interface(), c)
						if err != nil {
							return err
						}
						switch v := v.(type) {
						case bson.D:
							for _, e := range v {
								if err := decodeValue(ctx, t.Elem(), e, h.Fields, c); err != nil {
									return err
								}
							}
						default:
							if err := decodeValue(ctx, t.Elem(), e, h.Fields, c); err != nil {
								return err
							}
						}
						newVal = reflect.Append(newVal, t.Elem())
					}
				case reflect.Map:
					fmt.Println("why")
				default:
					for _, v := range val {
						newVal = reflect.Append(newVal, reflect.ValueOf(v))
					}
				}

				f.ReflectValueOf(ctx, r).Set(newVal)
			case bson.Binary:
				if val, ok := e.Value.(bson.Binary); ok {
					if val.Subtype == 4 {
						id, err := uuid.FromBytes(val.Data)
						if err != nil {
							return err
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
					if f.FieldType.Elem().Kind() == reflect.Struct {
						for _, v := range val {
							t := reflect.New(f.FieldType.Elem())
							h, err := parse(t.Interface(), c)
							if err != nil {
								return err
							}
							if err := decodeValue(ctx, t.Elem(), v, h.Fields, c); err != nil {
								return err
							}
							newMap.SetMapIndex(reflect.ValueOf(v.Key), t.Elem())
						}

					} else {
						for _, v := range val {
							newMap.SetMapIndex(reflect.ValueOf(v.Key), reflect.ValueOf(v.Value))
						}
					}

					f.ReflectValueOf(ctx, r).Set(newMap)
				case reflect.Struct:
					t := reflect.New(f.FieldType)
					h, err := parse(t.Interface(), c)
					if err != nil {
						return err
					}
					val, ok := e.Value.(bson.D)
					if !ok {
						return errors.New("not bson type")
					}
					for _, v := range val {
						if err := decodeValue(ctx, t.Elem(), v, h.Fields, c); err != nil {
							return err
						}
					}

					f.ReflectValueOf(ctx, r).Set(t.Elem())
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
	return nil
}
