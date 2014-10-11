package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
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

	testFindUser()
	result := FindUser(user.Username, user.Password)

	if result == nil {
		rw.WriteHeader(401)
		rw.Write([]byte("401 Unauthorized"))
	} else {
		rw.Header().Set("Token", "****")
		rw.WriteHeader(200)
	}
}

func testFindUser() {
	key := []byte("Batman Punching The Easter Bunny") // 32 bytes
	result := FindUser("test", "testPW")

	if result != nil {
		decrypted, err := decrypt(key, []byte(result.EncryptedPasswords))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Decrypted: %s\n", decrypted)
	}
}

func (u *User) EncryptPasswords() {
	key := []byte("Batman Punching The Easter Bunny") // 32 bytes

	passes, err := json.Marshal(u.Passwords)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Original: %s\n", passes)

	ciphertext, err := encrypt(key, passes)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%0x\n", ciphertext)

	u.EncryptedPasswords = string(ciphertext)

	result, err := decrypt(key, []byte(u.EncryptedPasswords))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Decrypted: %s\n", result)
}

func FindUser(username string, password string) *User {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}

	user := User{}
	c := session.DB("wordpass").C("Users")
	err = c.Find(bson.M{"username": username,
		"password": encryptPassword(password)}).One(&user)
	if err != nil {
		log.Fatal(err)
	}

	return &user
}

func encryptPassword(password string) string {
	return password
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

func encrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func decrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}
