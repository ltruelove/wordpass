package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	//"net/url"
	//"strings"
)

func main() {
	port := "8085"
	fmt.Printf("Listening at :%s\n"+
		"Routes:\n"+
		"/\n"+
		"/Login\n", port)

	//use gorilla mux for handling routes
	router := mux.NewRouter()
	router.HandleFunc("/Login", HandleLogin).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	//tell http to use the mux router
	http.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

type User struct {
	Username  string
	Password  string
	First     string
	Last      string
	Passwords []Pass
}

type Pass struct {
	Name        string
	Password    string
	URL         string
	Description string
}

func HandleLogin(rw http.ResponseWriter, req *http.Request) {
	params := json.NewDecoder(req.Body)
	var user User
	err := params.Decode(&user)
	if err != nil {
		panic(err)
	}

	rw.Header().Set("Token", "****")
	rw.WriteHeader(401)
	rw.Write([]byte("401 Unauthorized"))
	userJson, _ := json.Marshal(user)
	fmt.Println(string(userJson))
}
