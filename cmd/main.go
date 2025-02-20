package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-monarch/monarch"
	"github.com/go-monarch/monarch/query"
	"github.com/google/uuid"
)

func main() {
	type Status string
	type LastLogin int

	type UserProfile struct {
		ID        uuid.UUID         `monarch:"id,index"`
		Email     string            `monarch:"email"`
		Age       int               `monarch:"age"`
		LastLogin LastLogin         `monarch:"last_login"`
		Brands    []string          `monarch:"brands"`
		Status    Status            `monarch:"status"`
		Session   map[string]string `monarch:"session"`
		Skip      string            `monarch:"-"`
		Details   struct {
			ID string `monarch:"id"`
		} `monarch:"details"`
		CreatedAt time.Time `monarch:"created_at"`
	}

	c, err := monarch.Connect("mongodb://localhost")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	m := monarch.New(c)

	u, err := monarch.RegisterCollection(m, UserProfile{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := u.Save(context.Background(), UserProfile{
		ID:        uuid.New(),
		Email:     "jon@doe.com",
		Age:       20,
		LastLogin: LastLogin(300),
		Brands:    []string{"lorem", "ipsum"},
		Session: map[string]string{
			"id": "slsdhvsdjlkdssdlj",
		},
		Skip: "hello",
		Details: struct {
			ID string "monarch:\"id\""
		}{
			ID: "hello",
		},
		Status:    Status("pending"),
		CreatedAt: time.Now(),
	}); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	d, err := u.Query(context.Background(), query.SetLimit(3)).FindMany()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	for _, a := range d {
		fmt.Println(a)
	}

	fmt.Println(u.Query(context.Background()).FindOne())
}
