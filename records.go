package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	//"fmt"
	//"code.google.com/p/go.crypto/bcrypt"
	//"code.google.com/p/gcfg"
)

func (u User) registerRecordRoutes(router *mux.Router) {
	router.HandleFunc("/Records", SaveRecords).Methods("POST")
	router.HandleFunc("/RecordList", GetRecords).Methods("POST")
	router.HandleFunc("/Record/Blank", GetBlank).Methods("GET")
}

func GetBlank(rw http.ResponseWriter, req *http.Request) {
	pass := Pass{}
	result, err := json.Marshal(pass)

	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("There was an error retrieving the blank record"))
		return
	}

	rw.WriteHeader(200)
	rw.Write(result)
}

func GetRecords(rw http.ResponseWriter, req *http.Request) {
	//validte the API token
	tokenUser, err := validateToken(req)
	if err != nil {
		panic(err)
	}

	//get the posted data into a slice of Passes
	params := json.NewDecoder(req.Body)
	var apiRequest ApiRequest

	// The thing that's REALLY needed here is the the password key
	err = params.Decode(&apiRequest)
	if err != nil {
		panic(err)
	}

	//connect to mongo
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	//check for an existing user
	existing := &User{}
	conn := session.DB("wordpass").C("Users")
	err = conn.Find(bson.M{"_id": tokenUser.Id}).One(existing)
	if err != nil {
		//log.Fatal(err)
	}

	if len(apiRequest.PasswordKey) < 1 {
		panic("userKey cannot be empty")
	}

	fullKey := GetFullKey(apiRequest.PasswordKey)

	decrypted, decryptErr := decrypt(fullKey, []byte(existing.EncryptedPasswords))
	if decryptErr != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(decryptErr.Error()))
		return
	}

	if decrypted == nil {
		decrypted = []byte("[]")
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)
	rw.Write(decrypted)
}

func SaveRecords(rw http.ResponseWriter, req *http.Request) {
	//validte the API token
	tokenUser, err := validateToken(req)
	if err != nil {
		rw.WriteHeader(401)
		rw.Write([]byte("Token is invalid"))
		return
	}

	//get the posted data into a slice of Passes
	params := json.NewDecoder(req.Body)
	var user User

	// The things that are REALLY needed here are the array of Pass records and
	// the password key
	err = params.Decode(&user)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("Could not decode the data"))
		return
	}

	//connect to mongo
	session, err := mgo.Dial("localhost")
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("Connection error"))
		return
	}
	defer session.Close()
	conn := session.DB("wordpass").C("Users")

	existing := &User{}
	err = conn.Find(bson.M{"_id": tokenUser.Id}).One(existing)
	if err != nil {
		rw.WriteHeader(404)
		rw.Write([]byte("User not found"))
		return
	}

	if len(user.PasswordKey) < 1 {
		panic("userKey cannot be empty")
	}

	fullKey := GetFullKey(user.PasswordKey)

	_, decryptErr := decrypt(fullKey, []byte(existing.EncryptedPasswords))
	if decryptErr != nil {
		rw.WriteHeader(401)
		rw.Write([]byte("Password key is incorrect"))
		return
	}

	//if our user key is correct then we can continue
	user.EncryptRecords()

	//set up the change in the database
	var change = mgo.Change{
		ReturnNew: true,
		Update: bson.M{
			"$set": bson.M{
				"encryptedpasswords": user.EncryptedPasswords,
			},
		},
	}

	//this saves the changed encrypted passwords string
	_, err = conn.FindId(tokenUser.Id).Apply(change, &user)

	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("500 Internal Server Error"))
		return
	}

	rw.WriteHeader(200)
	rw.Write([]byte("success"))
}
