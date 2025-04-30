package query

type QueryKey string

var (
	QueryFilter QueryKey = "filter"
	QuerySort   QueryKey = "sort"
	QueryLimit  QueryKey = "limit"
	QueryOffset QueryKey = "offset"
)

type QueryStruct struct {
	key   QueryKey
	value interface{}
}

func (q QueryStruct) Key() QueryKey {
	return q.key
}

func (q QueryStruct) Value() interface{} {
	return q.value
}

type Params func() QueryStruct

type OrderStruct struct {
	key   string
	value OrderType
}
type OrderType int

func (o OrderType) String() string {
	return orderTypeMap[o]
}

func (o OrderStruct) Key() string {
	return o.key
}

func (o OrderStruct) Value() OrderType {
	return o.value
}

const (
	ASC OrderType = iota
	DESC
)

var orderTypeMap = map[OrderType]string{
	ASC:  "asc",
	DESC: "desc",
}

func WithOrder(key string, value OrderType) Params {
	return func() QueryStruct {
		return QueryStruct{
			key: QuerySort,
			value: OrderStruct{
				key:   key,
				value: value,
			},
		}
	}
}

type FilterStruct struct {
	key   string
	value interface{}
}

func (f FilterStruct) Key() string {
	return f.key
}

func (f FilterStruct) Value() interface{} {
	return f.value
}

func WithFilter(key string, value interface{}) Params {
	return func() QueryStruct {
		return QueryStruct{
			key: QueryFilter,
			value: FilterStruct{
				key:   key,
				value: value,
			},
		}
	}
}

func SetLimit(value int64) Params {
	return func() QueryStruct {
		return QueryStruct{
			key:   QueryLimit,
			value: value,
		}
	}
}

func SetOffset(value int64) Params {
	return func() QueryStruct {
		return QueryStruct{
			key:   QueryOffset,
			value: value,
		}
	}
}
