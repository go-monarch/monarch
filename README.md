# Monarch - ORM for mongodb

## Introduction

Monarch is an ORM written for mongodb queries in go. While the current mongo driver works great, writing repository logic over and over again can get very tedious and repetitive. Monarch exists as a middle ground for developers, giving the ease of an ORM while not as bloated as a traditional one.

## Usage

### Download

``` bash
go mod init my-app

go get -u github.com/go-monarch/monarch
```

### Set up

``` go
package main

type User struct {
    ID string `monarch:"id,index"`
    Email string `monarch:"email"`
    Password string `monarch:"password"`
    Auto bool
}

func main(){
    c, err := monarch.Connect("mongodb://localhost")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	m := monarch.New(c)

	u, err := monarch.RegisterCollection(m, User{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
```
Monarch has a beatiful collection api that allows for seamless data queries.

#### Queries

``` go
//Find Many
d, err := u.Query(context.Background(), query.SetLimit(3)).FindMany()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

//Find One
d, err := u.Query(context.Background(), query.WithFilter("id", "user_id")).FindOne()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

//Update One
err := u.Query(context.Background(), query.WithFilter("id", "user_id")).UpdateOne(User{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

//Update Many
err := u.Query(context.Background(), query.WithFilter("id", "user_id")).UpdateMany(User{})
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

//Delete One
err := u.Query(context.Background(), query.WithFilter("id", "user_id")).DeleteOne()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

//Delete Many
err := u.Query(context.Background(), query.WithFilter("id", "user_id")).DeleteMany()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
```

#### Save
```go
if err := u.Save(context.Background(), User{
		ID:        uuid.New(),
		Email:     "jon@doe.com",
	}); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
```
