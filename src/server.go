package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"context"


	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"hydrodash/mongDrive"
)

var templates = template.Must(template.ParseFiles("../Templates/index.html", "../Templates/login.html"))

// a authenticated c cookie, u username, e email
type Session struct {
	A bool
	C []byte
	U string
	E string
}

var curSession = Session{A: false}

func renderTemplate(w http.ResponseWriter, tmpl string) {

	err := templates.ExecuteTemplate(w, tmpl+".html", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func indexHanlder(w http.ResponseWriter, r *http.Request) {
	fmt.Println("index handler called")
	renderTemplate(w, "index")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Login handler Called")
	renderTemplate(w, "login")
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Validate Handler Called")
	curSession.A = true
	http.Redirect(w, r, "/", http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !curSession.A {
			fmt.Println("Redirecting per auth")
			http.Redirect(w, r, "/login/", http.StatusFound)
		}
		fn(w, r)
	}
}

func main() {
	//Use .env for consection string
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	var db_chan = make(chan []string, 0)
	var rep_chan = make(chan []mongDrive.Report, 0)
	mongo_uri := os.Getenv("MONGOSTRING")
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Fatal(err)
	}

	go mongDrive.GetDBNames(client, db_chan)
	dbList := db_chan

	for _, db := range <-dbList {
		fmt.Println(db)
	
	}


	go mongDrive.GetAgentReports(client, "agent_Go4", rep_chan )
	repLst := rep_chan

	for _, rep := range <-repLst {
		fmt.Println(rep)
	
	}



	
	//http.HandleFunc("/", makeHandler(indexHanlder))
	//http.HandleFunc("/login/", loginHandler)
	//http.HandleFunc("/validate/", validateHandler)

	//log.Fatal(http.ListenAndServe(":8809", nil))
}
