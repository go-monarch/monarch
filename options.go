package monarch

import (
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type OrderType int

const (
	ASC OrderType = iota
	DESC
)

type querier struct {
	limit  int64
	offset int64
	filter bson.D
	order  bson.D
	data   any
}

type QueryOptions func(q *querier) error

func Equals(key string, value any) QueryOptions {
	return func(q *querier) error {
		q.filter = append(q.filter, bson.E{Key: key, Value: value})
		return nil
	}
}

func Size() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func In() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func LessThanEqual() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func GreaterThanEqual() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func LessThan() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func GreaterThan() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func Regex() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func Or() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func Not() QueryOptions {
	return func(q *querier) error {
		panic("unimplemented")
	}
}

func OrderBy(key string, val OrderType) QueryOptions {
	return func(q *querier) error {
		var order int64
		switch val {
		case ASC:
			order = -1
		case DESC:
			order = 1
		default:
			return errors.New("error, unrecognized order")
		}
		q.order = append(q.order, bson.E{Key: key, Value: order})
		return nil
	}
}

func Limit(limit int64) QueryOptions {
	return func(q *querier) error {
		q.limit = limit
		return nil
	}
}

func Skip(skip int64) QueryOptions {
	return func(q *querier) error {
		q.offset = skip
		return nil
	}
}

func Data(data any) QueryOptions {
	return func(q *querier) error {
		q.data = data
		return nil
	}
}
