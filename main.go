package main

import (
	"os"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rapulu/url-shortner/db"
	hashids "github.com/speps/go-hashids"
	"go.mongodb.org/mongo-driver/bson"
)

//URL structures
type URL struct {
	LongURL  string `json:"long_url,omitempty"`
	ShortURL string `json:"short_url,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	
	if port == ""{
		port = "9090"
	}

	mx := mux.NewRouter()
	
	mx.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./index.html")
	}).Methods(http.MethodGet)
	
	mx.HandleFunc("/", HandleIncomingRequest).Methods(http.MethodPost)

	mx.HandleFunc("/{id}", FindURLEndpoint).Methods(http.MethodGet)

	s := http.StripPrefix("/frontend/", http.FileServer(http.Dir("frontend")))

	mx.PathPrefix("/frontend/").Handler(s)

	http.ListenAndServe(":"+port, mx)

}

//FindURLEndpoint finds the url and redirect to original url
func FindURLEndpoint(resp http.ResponseWriter, req *http.Request){
	code := mux.Vars(req)["id"]
	url :=  &URL{}

	client, err := db.GetMongoClient()

	if err != nil {
		fmt.Fprintf(resp, "InsertOne ERROR: %v \n", err)
		return // safely exit script on error
	}

	col := client.Database("testDB").Collection("url")

	//return the inserted document
	err = col.FindOne(context.TODO(), bson.M{"shorturl": code}).Decode(url)
	if err != nil {
		fmt.Fprintf(resp, "Findone ERROR: %v \n", err)
		return // safely exit script on error
	}

	link := "http://"+url.LongURL
	http.Redirect(resp, req, link, 302)
	return
}

//HandleIncomingRequest Handles incoming request from users
func HandleIncomingRequest(w http.ResponseWriter, r *http.Request) {
	url := &URL{}
	url.ShortURL, _ = generateCode()
	dec := json.NewDecoder(r.Body)
	dec.Decode(url)

	client, err := db.GetMongoClient()

	if err != nil {
		fmt.Fprintf(w, "InsertOne ERROR: %v \n", err)
		return // safely exit script on error
	}

	col := client.Database("testDB").Collection("url")

	result, insertErr := col.InsertOne(context.TODO(), url)

	if insertErr != nil {
		fmt.Fprintf(w, "InsertOne ERROR: %v \n", insertErr)
		return // safely exit script on error
	}

	// get the inserted ID string
	newID := result.InsertedID

	//return the inserted document
	err = col.FindOne(context.TODO(), bson.M{"_id": newID}).Decode(url)
	if err != nil {
		fmt.Fprintf(w, "Findone ERROR: %v \n", err)
		return // safely exit script on error
	}
	enc := json.NewEncoder(w)
	enc.Encode(url)
}

func generateCode() (string, error) {
	hd := hashids.NewData()

	hash, err := hashids.NewWithData(hd)
	if err != nil {
		return "", errors.New("Something went wrong generating code")
	}

	t := time.Now()
	hashed, err := hash.Encode([]int{int(t.Unix())})
	if err != nil {
		return "", errors.New("Something went wrong generating code")
	}
	return hashed, nil
}
