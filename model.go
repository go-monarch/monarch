package monarch

import "time"

type TimeStamp struct {
	CreatedAt time.Time `monarch:"created_at"`
	UpdatedAt time.Time `monarch:"updated_at"`
}
