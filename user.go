package main

import (
	"code.google.com/p/go.crypto/pbkdf2"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/nu7hatch/gouuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"log"
	"net/http"
	"time"
	//"code.google.com/p/go.crypto/bcrypt"
	//"code.google.com/p/gcfg"
)

type User struct {
	Id                 bson.ObjectId `json:"id" bson:"_id"`
	Username           string
	Password           string
	First              string
	Last               string
	Passwords          []Pass `bson:"-"`
	EncryptedPasswords string
	PasswordKey        string `bson:"-"`
}

type Pass struct {
	Name        string
	Password    string
	URL         string
	Description string
}

type AccessToken struct {
	Token        string
	UserId       bson.ObjectId
	LastAccessed time.Time
}

func (u User) registerRoutes(router *mux.Router) {
	router.HandleFunc("/Login", HandleLogin).Methods("POST")
	router.HandleFunc("/UserCreate", UserCreate).Methods("POST")
	router.HandleFunc("/User", Insert).Methods("POST")
	router.HandleFunc("/User", Update).Methods("PUT")
	//router.HandleFunc("/User", Delete).Methods("DELETE")
	//router.HandleFunc("/User/{id}", Find).Methods("GET")
	//router.HandleFunc("/User", Get).Methods("GET")
}

func UserCreate(rw http.ResponseWriter, req *http.Request) {
	//get the posted data into a User struct
	params := json.NewDecoder(req.Body)
	var user User
	err := params.Decode(&user)
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
	err = conn.Find(bson.M{"username": user.Username}).One(existing)
	if err != nil {
		//log.Fatal(err)
	}

	if existing.Username != "" {
		rw.WriteHeader(409)
		rw.Write([]byte("409 Conclict"))
		panic("User exists")
	}

	//save the user
	c := session.DB("wordpass").C("Users")
	user.EncryptPassword()

	user.EncryptRecords()
	user.Id = bson.NewObjectId()
	err = c.Insert(&user)

	if err != nil {
		panic(err)
	}

	rw.WriteHeader(200)
}

func Update(rw http.ResponseWriter, req *http.Request) {
	accessToken := AccessToken{UserId: bson.NewObjectId()}
	tokenText := req.Header.Get("Token")
	accessToken.Token = tokenText

	if !validateToken(&accessToken) {
		panic("Invalid token")
	}

	//get the posted data into a User struct
	params := json.NewDecoder(req.Body)
	var user User
	err := params.Decode(&user)
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
	err = conn.Find(bson.M{"_id": accessToken.UserId}).One(existing)
	if err != nil {
		panic(err)
	}

	if existing.Username == "" {
		rw.WriteHeader(409)
		rw.Write([]byte("409 Conclict"))
		panic("User does not exist")
	}

	var change = mgo.Change{
		ReturnNew: true,
		Update: bson.M{
			"$set": bson.M{
				"username": user.Username,
				"first":    user.First,
				"last":     user.Last,
			},
		},
	}

	_, err = conn.FindId(existing.Id).Apply(change, existing)

	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("500 Internal Server Error"))
		panic("There was an error saving the user")
	}

	return
}

func Insert(rw http.ResponseWriter, req *http.Request) {
	accessToken := AccessToken{UserId: bson.NewObjectId()}
	tokenText := req.Header.Get("Token")
	accessToken.Token = tokenText

	if !validateToken(&accessToken) {
		panic("Invalid token")
	}

	//get the posted data into a User struct
	params := json.NewDecoder(req.Body)
	var user User
	err := params.Decode(&user)
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
	err = conn.Find(bson.M{"username": user.Username}).One(existing)
	if err != nil {
		//log.Fatal(err)
	}

	if existing.Username != "" {
		rw.WriteHeader(409)
		rw.Write([]byte("409 Conclict"))
		panic("User exists")
	}

	//save the user
	c := session.DB("wordpass").C("Users")
	user.EncryptPassword()

	/***
	  this will eventually need to be something passed in or stored in a cookie or something
	  ***/
	user.EncryptRecords()
	user.Id = bson.NewObjectId()
	err = c.Insert(&user)

	if err != nil {
		panic(err)
	}

	rw.WriteHeader(200)
}

func HandleLogin(rw http.ResponseWriter, req *http.Request) {
	params := json.NewDecoder(req.Body)
	var user User
	err := params.Decode(&user)
	if err != nil {
		panic(err)
	}

	//testFindUser()
	user.EncryptPassword()
	result := FindUser(user.Username, user.Password)

	if result == nil {
		rw.WriteHeader(401)
		rw.Write([]byte("401 Unauthorized"))
	} else {
		accessToken := getToken(result.Id)
		rw.Header().Set("Token", accessToken.Token)
		rw.WriteHeader(200)
	}
}

func validateToken(accessToken *AccessToken) bool {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	c := session.DB("wordpass").C("Tokens")
	err = c.Find(bson.M{"token": accessToken.Token}).One(accessToken)
	if err != nil {
		panic(err)
		return false
	}

	current := time.Now()
	if accessToken.LastAccessed.Before(current) {
		dif := current.Sub(accessToken.LastAccessed).Minutes()
		if dif > 15 {
			return false
		} else {
			accessToken.LastAccessed = time.Now()
			err = c.Update(bson.M{"token": accessToken.Token}, accessToken)
			if err != nil {
				panic(err)
			}
			return true
		}
	} else {
		return false
	}

}

func getToken(userId bson.ObjectId) AccessToken {
	accessToken := AccessToken{"", userId, time.Now()}
	token, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	accessToken.Token = token.String()

	//connect to mongo
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	//save the token
	c := session.DB("wordpass").C("Tokens")
	err = c.Insert(&accessToken)

	if err != nil {
		panic(err)
	}

	return accessToken
}

func testFindUser() {
	key := []byte(RecordSalt) // 32 bytes
	result := FindUser("test", "testPW")

	if result != nil {
		decrypted, err := decrypt(key, []byte(result.EncryptedPasswords))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Decrypted: %s\n", decrypted)
	}
}

func (u *User) EncryptRecords() {
	userKey := []byte(u.PasswordKey)

	if len(userKey) < 1 {
		panic("userKey cannot be empty")
	}

	key := []byte(RecordSalt) // 32 bytes
	var fullKey []byte

	if len(userKey) < 32 && len(userKey) > 0 {
		//buffer the rest of fullKey with whatever fills it out to
		//32 bytes from the RecordSalt
		fullKey = userKey
		//tempSlice will hold the rest of the RecordSalt after len of userKey
		var tempSlice = key[len(userKey):len(key)]

		for i := 0; i < len(tempSlice); i++ {
			fullKey = append(fullKey, tempSlice[i])
		}
	} else {
		//take the first 32 bytes of the userKey
		fullKey = userKey[:31]
	}

	passes, err := json.Marshal(u.Passwords)
	if err != nil {
		panic(err)
	}

	ciphertext, err := encrypt(fullKey, passes)
	if err != nil {
		log.Fatal(err)
	}

	u.EncryptedPasswords = string(ciphertext)
}

func FindUser(username string, password string) *User {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	user := User{}
	c := session.DB("wordpass").C("Users")
	err = c.Find(bson.M{"username": username,
		"password": password}).One(&user)
	if err != nil {
		log.Fatal(err)
	}

	return &user
}

func (u *User) EncryptPassword() {
	//add encryption routine here
	salt := []byte(PasswordSalt)
	u.Password = string(HashPassword([]byte(u.Password), salt))
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}

func HashPassword(password, salt []byte) []byte {
	defer clear(password)
	return pbkdf2.Key(password, salt, 4096, sha256.Size, sha256.New)
}

/* Just in case we need this for some reason later
func Crypt(password []byte) ([]byte, error) {
	defer clear(password)
	return bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
}
*/

func encryptPassword(password string) string {
	return password
}

func (u User) SaveTest() {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	c := session.DB("wordpass").C("Users")
	err = c.Insert(&u)

	if err != nil {
		panic(err)
	}

	fmt.Println("User saved\r\n")
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
