package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"math/rand"

	"hydrodash/mongDrive"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var u = uint8(rand.Intn(255))

var (
	
	key = []byte{239,  57, 183,  33, 121, 175, 214,  u,  52, 235,  33, 167,  74,  91, 153,  39 }
	store = sessions.NewCookieStore(key)
)

var authDict map[string]mongDrive.User

var templates = template.Must(template.ParseFiles(
	"../Templates/index.html",
	"../Templates/login.html",
	"../Templates/agents.html",
	"../Templates/reports.html",
))

// a authenticated c cookie, u username, e email
type Session struct {
	Auth     bool
	Cookie   []byte
	Username string
	Email    string
}

type Agent struct {
	Name        string
	Mongo       *mongo.Client
	ReportFunc  func(*mongo.Client, string, string, chan map[string]mongDrive.Report)
	NamesFunc   func(*mongo.Client, string, chan map[string]bool)
	ReportErr   chan map[string]bool
	Reports     chan map[string]mongDrive.Report
}

func (a Agent) reportList() {
	go a.NamesFunc(a.Mongo, a.Name, a.ReportErr)
}

func (a Agent) singleReport(reportName string) {
	go a.ReportFunc(a.Mongo, a.Name, reportName, a.Reports)
}

var agents = make(map[string]Agent)
var agentList []string

func renderTemplate(w http.ResponseWriter, tmpl string, data any) {
	//fmt.Println(data)
	err := templates.ExecuteTemplate(w, tmpl+".html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Handler Functions///////////////////////////////////////////////////////////////////////
func indexHanlder(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("index handler called")
	renderTemplate(w, "index", agentList)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("Login handler Called")
	renderTemplate(w, "login", "")
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Println("Validate Handler Called")
	session, _ := store.Get(r, "cookie-name")
	_ = r.ParseForm()
	fmt.Println(r.Form["Username"])
	fmt.Println(r.Body)
	fmt.Println(r.PostForm)
    session.Values["authenticated"] = true
    session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func agentHandler(w http.ResponseWriter, r *http.Request, agentName string) {
	a := strings.Split(agentName, "/")[2]
	agent := agents[a]
	agent.reportList()
	orgNames := map[string][]string{"Error": nil, "Working": nil}
	
	//Parse reports into lists of working or not
	for key, elm := range <- agent.ReportErr{
		if elm{
			orgNames["Error"] = append(orgNames["Error"], key)
		}else{
			orgNames["Working"] = append(orgNames["Working"], key)
		}
	}


	data := struct {
		Db      string
		Error []string
		Working []string
	}{
		Db:      a,
		Error: orgNames["Error"],
		Working: orgNames["Working"],
	}
	renderTemplate(w, "agents", data)
}

func reportHandler(w http.ResponseWriter, r *http.Request, path []string) {
	agentName := path[2]
	reportName := path[3]
	agents[agentName].singleReport(reportName)
	var report = <-agents[agentName].Reports
	renderTemplate(w, "reports", report[reportName])
}

// Handler Makers//////////////////////////////////////////////////////////////////////////////
func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "cookie-name")

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			fmt.Println("Redirecting per auth")
			http.Redirect(w, r, "/login/", http.StatusFound)
		}
		fn(w, r)
	}
}

func makeReportHandler(fn func(http.ResponseWriter, *http.Request, []string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "cookie-name")

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			fmt.Println("Redirecting per auth")
			http.Redirect(w, r, "/login/", http.StatusFound)
		}
		path := string(r.URL.Path)
		pathArr := strings.Split(path, "/")
		if pathArr[2] == "" {
			http.NotFound(w, r)
			return
		}
		fn(w, r, pathArr)
	}
}

func makeAgentHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "cookie-name")

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			fmt.Println("Redirecting per auth")
			http.Redirect(w, r, "/login/", http.StatusFound)
		}
		agent := string(r.URL.Path)
		if agent == "" {
			http.NotFound(w, r)
			return
		}
		fn(w, r, agent)
	}
}

// Main//////////////////////////////////////////////////////////////////////////////////////////////////////
func main() {
	//Use .env for consection string
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	var db_chan = make(chan []string)
	//var rep_chan = make(chan map[string]mongDrive.Report, 0)
	var srv_chan = make(chan []mongDrive.Server)
	//var repName_chan = make(chan []string, 0)
	mongo_uri := os.Getenv("MONGOSTRING")
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Fatal(err)
	}

	authDict = mongDrive.GetAuths(client)


	go mongDrive.GetDBNames(client, db_chan)
	dbList := db_chan

	//Get Agents and report should extract to function to update every 900,000 ms
	for _, db := range <-dbList {
		split := strings.Split(db, "_")
		if len(split) > 1 && split[1] != "server" {
			agentList = append(agentList, db)
			//Each agent gets its own set of chanels and goroutines that we start here.
			newAgent := Agent{Name: db, Mongo: client, ReportFunc: mongDrive.GetAgentReports, NamesFunc: mongDrive.GetAgentReportList,  Reports: make(chan map[string]mongDrive.Report), ReportErr: make(chan map[string]bool)}
			agents[db] = newAgent
		}
	}

	go mongDrive.GetServerReports(client, srv_chan)
	//srvLst := srv_chan

	http.HandleFunc("/", makeHandler(indexHanlder))
	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/validate/", validateHandler)
	http.HandleFunc("/agents/", makeAgentHandler(agentHandler))
	http.HandleFunc("/reports/", makeReportHandler(reportHandler))

	log.Fatal(http.ListenAndServe(":8809", nil))
}
