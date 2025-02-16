package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-monarch/monarch"
)

func main() {
	type User struct {
		ID        string `monarch:"id,index"`
		Name      string `monarch:"name"`
		Skip      string
		Group     []string  `monarch:"group"`
		CreatedAt time.Time `monarch:"created_at"`
	}

	type UserProfile struct {
		ID    string `monarch:"id,index"`
		Email int    `monarch:"email"`
	}

	type Address struct {
		Street string `monarch:"street"`
	}

	m, err := monarch.Connect("mongodb://localhost:27017")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}

	client := monarch.New(m)
	client.UseDB("monarch-test")

	monarch.RegisterCollection(client, User{})
	u, err := monarch.RegisterCollection(client, UserProfile{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(u.Query().FindMany())
	monarch.RegisterCollection(client, Address{})
}
