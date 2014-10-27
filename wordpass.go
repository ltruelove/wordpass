package main

import (
	"code.google.com/p/gcfg"
	"encoding/json"
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

type ApiRequest struct {
	PasswordKey string
}

type SetupCheckResponse struct {
	IsNew bool
	User  *User
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
	fmt.Printf("Listening at :%s\n", port)

	//use gorilla mux for handling routes
	router := mux.NewRouter()
	user := User{}
	router.HandleFunc("/setupCheck", SetupCheck).Methods("GET")
	user.registerRoutes(router)
	user.registerRecordRoutes(router)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	//tell http to use the mux router
	http.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func SetupCheck(rw http.ResponseWriter, req *http.Request) {
	//connect to mongo
	session, err := mgo.Dial("localhost")
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("500 Server Error"))
	}

	defer session.Close()

	conn := session.DB("wordpass").C("Users")
	records, err := conn.Count()
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("500 Server Error"))
	}

	response := SetupCheckResponse{IsNew: false, User: nil}

	if records < 1 {
		response.IsNew = true
	}

	result, err := json.Marshal(response)

	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("Server Error"))
		return
	}
	/*
	   We'll check for a user token in the session here later
	*/

	rw.WriteHeader(200)
	rw.Write(result)
}

func testMgo() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	fmt.Println("Mongo connection works\r\n")

	defer session.Close()
}
