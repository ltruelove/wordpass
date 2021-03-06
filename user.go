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
	//"log"
	"net/http"
	"time"
	//"code.google.com/p/go.crypto/bcrypt"
	//"code.google.com/p/gcfg"
)

type HomeUser struct {
	Username string
	First    string
	Last     string
}

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

type UserPermission struct {
	Id     bson.ObjectId `json:"id" bson:"_id"`
	UserId bson.ObjectId
	Type   string //create users, IsAdmin, etc
	Level  int    // 0:none, 1:read, 2:write, 3:delete
}

type Pass struct {
	Name        string
	Username    string
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
	router.HandleFunc("/UserCreateAdmin", CreateAdmin).Methods("POST")
	router.HandleFunc("/User", Insert).Methods("POST")
	router.HandleFunc("/User", Update).Methods("PUT")
	//router.HandleFunc("/User", Delete).Methods("DELETE")
	//router.HandleFunc("/User/{id}", Find).Methods("GET")
	//router.HandleFunc("/User", Get).Methods("GET")
}

func CreateAdmin(rw http.ResponseWriter, req *http.Request) {
	//get the posted data into a User struct
	params := json.NewDecoder(req.Body)
	var user User
	err := params.Decode(&user)
	if err != nil {
		panic(err)
	}

	if user.Username == "" {
		panic("Username is required")
	}

	if user.Password == "" {
		panic("Password is required")
	}

	if user.PasswordKey == "" {
		panic("Password Key is required")
	}

	//connect to mongo
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	//check for existing users
	conn := session.DB("wordpass").C("Users")
	records, err := conn.Count()
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("Error getting user count"))
		return
	}

	//cannot create the admin user if there are existing users
	if records > 0 {
		rw.WriteHeader(401)
		rw.Write([]byte("401 Unauthorized. Users exist."))
		return
	}

	//save the user
	user.EncryptPassword()

	user.EncryptRecords()
	user.Id = bson.NewObjectId()
	err = conn.Insert(&user)

	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("Error saving user."))
		return
	}

	//add the user to the admin list
	userP := UserPermission{Id: bson.NewObjectId(), UserId: user.Id, Type: "Admin", Level: 3}
	permCollection := session.DB("wordpass").C("Permissions")
	err = permCollection.Insert(&userP)

	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte(err.Error()))
		return
	}

	marshaledUser, err := json.Marshal(user)
	if err != nil {
		rw.WriteHeader(500)
		rw.Write([]byte("Error marshaling user"))
		return
	}

	rw.WriteHeader(200)
	rw.Write(marshaledUser)
}

/***
This will create a new user record. Make sure that Password and PasswordKey are
set so that the encryption works on their records.
***/
func UserCreate(rw http.ResponseWriter, req *http.Request) {
	/*
	 * TODO we want to check for the user's permissions before this now that we have a
	 * way to set up an admin
	 */

	//get the posted data into a User struct
	params := json.NewDecoder(req.Body)
	var user User
	err := params.Decode(&user)
	if err != nil {
		panic(err)
	}

	if user.Username == "" {
		panic("Username is required")
	}

	if user.Password == "" {
		panic("Password is required")
	}

	if user.PasswordKey == "" {
		panic("Password Key is required")
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
	tokenUser, err := validateToken(req)
	if err != nil {
		panic(err)
	}

	//get the posted data into a User struct
	params := json.NewDecoder(req.Body)
	var user User
	err = params.Decode(&user)
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
	_, err := validateToken(req)
	if err != nil {
		panic(err)
	}

	//get the posted data into a User struct
	params := json.NewDecoder(req.Body)
	var user User
	err = params.Decode(&user)
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
	result, _ := FindUser(user.Username, user.Password)

	if result == nil {
		rw.WriteHeader(401)
		rw.Write([]byte("401 Unauthorized"))
	} else {
		accessToken := getToken(result.Id)
		rw.Header().Set("Token", accessToken.Token)
		rw.WriteHeader(200)

		homeUser := HomeUser{Username: result.Username, First: result.First, Last: result.Last}
		marshaled, err := json.Marshal(homeUser)
		if err != nil {
			rw.WriteHeader(500)
			rw.Write([]byte("Internal server error"))
		}

		rw.Write(marshaled)
	}
}

func validateToken(req *http.Request) (*User, error) {
	accessToken := AccessToken{UserId: bson.NewObjectId()}
	tokenText := req.Header.Get("Token")
	accessToken.Token = tokenText

	session, err := mgo.Dial("localhost")
	if err != nil {
		return nil, err
	}

	defer session.Close()

	c := session.DB("wordpass").C("Tokens")
	err = c.Find(bson.M{"token": accessToken.Token}).One(&accessToken)
	if err != nil {
		return nil, err
	}

	current := time.Now()
	if accessToken.LastAccessed.Before(current) {
		dif := current.Sub(accessToken.LastAccessed).Minutes()
		if dif > 15 {
			return nil, fmt.Errorf("Token has expired")
		} else {
			accessToken.LastAccessed = time.Now()
			err = c.Update(bson.M{"token": accessToken.Token}, accessToken)
			if err != nil {
				return nil, err
			}
		}
	} else {
		return nil, fmt.Errorf("Token has expired")
	}

	user := &User{}
	userC := session.DB("wordpass").C("Users")
	err = userC.Find(bson.M{"_id": accessToken.UserId}).One(user)
	return user, err
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

	_, err = c.RemoveAll(bson.M{"userid": userId})

	if err != nil {
		panic(err)
	}

	err = c.Insert(&accessToken)

	if err != nil {
		panic(err)
	}

	return accessToken
}

func testFindUser() {
	key := []byte(RecordSalt) // 32 bytes
	result, _ := FindUser("test", "testPW")

	if result != nil {
		decrypted, err := decrypt(key, []byte(result.EncryptedPasswords))
		if err != nil {
			panic(err)
		}
		fmt.Printf("Decrypted: %s\n", decrypted)
	}
}

func GetFullKey(passwordKey string) []byte {
	userKey := []byte(passwordKey)

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

	return fullKey
}

func (u *User) EncryptRecords() {
	fullKey := GetFullKey(u.PasswordKey)

	passes, err := json.Marshal(u.Passwords)
	if err != nil {
		panic(err)
	}

	ciphertext, err := encrypt(fullKey, passes)
	if err != nil {
		panic(err)
	}

	u.EncryptedPasswords = string(ciphertext)
}

func FindUser(username string, password string) (*User, error) {
	session, err := mgo.Dial("localhost")
	if err != nil {
		return nil, err
	}
	defer session.Close()

	user := User{}
	c := session.DB("wordpass").C("Users")
	err = c.Find(bson.M{"username": username,
		"password": password}).One(&user)

	if err != nil {
		return nil, err
	}

	return &user, nil
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
