package main

import (
	"log"
	"fmt"
	"os"
	"net/http"
	"encoding/json"

	"github.com/gorilla/mux"
)

type GitHubPushEventPayload struct {
	Action		string		`json:"action"`
	Release struct {
		Tarball_URL	string 	`json:"tarball_url"`
		Tag_Name	string	`json:"tag_name"`
	} 				`json:"release"`
	Repository struct {
		FullName 	string	`json:"full_name"`
		Name		string	`json:"name"`
		CloneUrl	string	`json:"clone_url"`
	}				`json:repository"`
}
var payloadChannel chan GitHubPushEventPayload
func main() {
	if (os.Getenv("WEBHOOK_SECRET") == "") {
		log.Panicln("We don't support unsecured webhooks!")
	}else if (os.Getenv("DOCKERHUB_NAME") == ""){
		log.Panicln("We need a login Name for hub.docker.com")
	}else if (os.Getenv("DOCKERHUB_PW") == ""){
		log.Panicln("We need a passwort for hub.docker.com")
	}
	if (os.Getenv("DOCKERHUB_ORG") == ""){
		log.Println("Setting DOCKERHUB_ORG to DOCKERHUB_NAME")
		os.Setenv("DOCKERHUB_ORG", os.Getenv("DOCKERHUB_NAME"))
	}

	payloadChannel = make(chan GitHubPushEventPayload, 10)
	go loop()
	router := mux.NewRouter()

	router.Use(loggingMiddleware)
	router.HandleFunc("/", BuildHookReceiver).Methods("POST")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080) , router))
}

func BuildHookReceiver (w http.ResponseWriter, r *http.Request){
	payload := GitHubPushEventPayload{}
	log.Println(r.Body)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&payload)
	if (err != nil){
		log.Printf("Error decoding Github push payload: %s", err)
		return
	}
	log.Printf("Received action %s in repository %s for tag %s", payload.Action, payload.Repository.FullName, payload.Release.Tag_Name)
	if (payload.Action != "released") {
		log.Printf("Release Action %s is not tracked to deployment", payload.Action)
		return
	}
	payloadChannel <- payload
	log.Println("Added Tag to Queue")
}

func loop(){
	log.Println("Worker started")
	for {
		select {
		case payload := <- payloadChannel:
			log.Printf("Worker starting tag %s in repository %s\n", payload.Release.Tag_Name, payload.Repository.FullName)
			releaseRepository(payload)
			log.Printf("Worker finished tag %s in repository %s\n", payload.Release.Tag_Name, payload.Repository.FullName)
		}
	}
}
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[WEBHOOK] %s: %s\n", r.Method, r.RequestURI)
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, r)
	})
}
