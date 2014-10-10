package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	_ "gopkg.in/mgo.v2/bson"
	"net/http"
	//"net/url"
	//"strings"
)

func main() {
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
	router.HandleFunc("/Login", HandleLogin).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))

	//tell http to use the mux router
	http.Handle("/", router)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

type User struct {
	Username           string
	Password           string
	First              string
	Last               string
	Passwords          []Pass `bson:"-"`
	EncryptedPasswords string
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

	user.Passwords = append(user.Passwords, Pass{"pName", "pPass", "pUrl", "pDesc"})
	user.EncryptPasswords()
	user.SaveTest()

	rw.Header().Set("Token", "****")
	rw.WriteHeader(401)
	rw.Write([]byte("401 Unauthorized"))
}

func (u *User) EncryptPasswords() {
	passes, err := json.Marshal(u.Passwords)
	if err != nil {
		panic(err)
	}
	u.EncryptedPasswords = string(passes)
}

func (u User) SaveTest() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}

	c := session.DB("wordpass").C("Users")
	err = c.Insert(&u)

	if err != nil {
		panic(err)
	}

	fmt.Println(u.EncryptedPasswords)
	fmt.Println("User saved\r\n")

	defer session.Close()
}

func testMgo() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	fmt.Println("Mongo connection works\r\n")

	defer session.Close()
}
