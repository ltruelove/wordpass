package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"net/http"
)

var PasswordSalt string
var RecordSalt string

type Config struct {
	Dev struct {
		DatabaseName string
		Salt         string
		RecordSalt   string
	}
	Prod struct {
		DatabaseName string
		Salt         string
		RecordSalt   string
	}
}

func main() {
	//get our configs
	var cfg Config
	err := gcfg.ReadFileInto(&cfg, "config.gcfg")
	if err != nil {
		fmt.Printf("%s\r\n", err)
	}

	PasswordSalt = cfg.Dev.Salt
	RecordSalt = cfg.Dev.RecordSalt

	//Lets test our mongo db
	testMgo()

	//on to the rest of it
	port := "8085"
	fmt.Printf("Listening at :%s\n"+
		"Routes:\n"+
		"/\n"+
		"/Login\n", port)

	//use gorilla mux for handling routes
	router := mux.NewRouter()
	user := User{}
	user.registerRoutes(router)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	//tell http to use the mux router
	http.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func testMgo() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	fmt.Println("Mongo connection works\r\n")

	defer session.Close()
}
