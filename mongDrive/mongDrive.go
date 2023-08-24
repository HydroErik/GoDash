package mongDrive

import (
	"context"
	"fmt"
	"log"

	//"os"
	//"reflect"
	"time"

	//"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type Report struct {
	Name        string
	Err_flag    bool
	LastReview  time.Time
	LastRun     time.Time
	Source      string
	ErrStr      string
	Destination string
}

type Server struct {
	Name      string
	Cpu       []float64
	Drives    map[string][]float64
	TimeStamp []time.Time
}

type User struct {
	Email    string
	Username string
	Password string
	Name     string
}

func EncryptPass(old_pass string) (string, error) {
	//encoded := base64.StdEncoding.EncodeToString([]byte(old_pass))
	encoded, err := bcrypt.GenerateFromPassword([]byte(old_pass), bcrypt.DefaultCost)
	return string(encoded), err
}

// Retrieve the names of the databases in the mongo instance
// takes connection string and returns array of string names and error
func GetDBNames(client *mongo.Client, db_names chan []string) {

	dbs, err := client.ListDatabaseNames(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}

	db_names <- dbs
}

// Get the list of site returns for a given agent DB
// When report name passsed in return single record
// Returns list of maps with for given agent with common structure
func GetAgentReports(client *mongo.Client, db string, reportName string, reports chan map[string]Report) {

	reportCol := client.Database(db).Collection("Report")

	//Need to pre-declare to avoid nill operator
	var cur *mongo.Cursor
	var err error

	if reportName != "" {
		cur, err = reportCol.Find(context.TODO(), bson.D{{"name", reportName}}) //Get a single report
	} else {
		cur, err = reportCol.Find(context.TODO(), bson.D{}) //Get all report
	}
	if err != nil {
		log.Fatal(err)
	}
	repDic := make(map[string]Report)

	for cur.Next(context.TODO()) {
		var reportRaw bson.M
		err := cur.Decode(&reportRaw)
		if err != nil {
			fmt.Println(err)
		}

		var newReport = Report{
			Name:        reportRaw["name"].(string),
			Err_flag:    reportRaw["Error Flag"].(bool),
			LastReview:  reportRaw["Last Review"].(primitive.DateTime).Time(),
			LastRun:     reportRaw["LastRun"].(primitive.DateTime).Time(),
			Destination: reportRaw["destination"].(string),
			Source:      reportRaw["source"].(string),
			ErrStr:      reportRaw["error"].(string),
		}
		repDic[reportRaw["name"].(string)] = newReport
	}
	reports <- repDic

}

// Get the list of reports for a given agent database
// Takes a client the db_name, and a chanel for the return []string
func GetAgentReportList(client *mongo.Client, db string, reportNames chan map[string]bool) {
	reportCol := client.Database(db).Collection("Report")
	opts := options.Find().SetProjection(bson.D{{"name", 1}, {"Error Flag", 1}})
	cur, err := reportCol.Find(context.TODO(), bson.D{}, opts) //Get all the reports
	if err != nil {
		log.Fatal(err)
	}
	var ret = make(map[string]bool)
	for cur.Next(context.TODO()) {

		var value bson.M
		err := cur.Decode(&value)
		if err != nil {
			log.Fatal(err)
		}
		ret[value["name"].(string)] = value["Error Flag"].(bool)
	}
	reportNames <- ret
}

// Return Server Data about all current hydrologik servers
// Returns a list of structs about each given server
func GetServerReports(client *mongo.Client, servers chan []Server) {
	var srvLst []Server
	srvDb := client.Database("agent_server")

	colLst, err := srvDb.ListCollectionNames(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}

	for _, col := range colLst {
		if col == "log" {
			continue
		}
		srv := srvDb.Collection(col)
		cur, err := srv.Find(context.TODO(), bson.D{})
		if err != nil {
			log.Fatal(err)
		}

		var cpu []float64
		drives := make(map[string][]float64)
		var timeStamps []time.Time

		for cur.Next(context.TODO()) {
			var reportRaw bson.M
			err := cur.Decode(&reportRaw)
			if err != nil {
				log.Fatal(err)
			}

			cpu = append(cpu, reportRaw["cpu_perc"].(float64))
			timeStamps = append(timeStamps, reportRaw["_id"].(primitive.DateTime).Time())
			//keys := reflect.ValueOf(reportRaw["drives"].(primitive.M)).MapKeys()

			for i, dic := range reportRaw["drives"].(primitive.M) {

				val := dic.(float64)
				drives[i] = append(drives[i], val)
			}
		}
		var newSrv = Server{Name: col, Cpu: cpu, Drives: drives, TimeStamp: timeStamps}
		srvLst = append(srvLst, newSrv)
	}

	servers <- srvLst
}

// Return dict of authenticated user from db
func GetAuths(client *mongo.Client) map[string]User {
	reportCol := client.Database("GoUsers").Collection("gocredentials")
	cur, err := reportCol.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}

	var userMap = make(map[string]User)

	for cur.Next(context.TODO()) {
		dbUser := bson.M{}
		err := cur.Decode(&dbUser)
		if err != nil {
			log.Fatal(err)
		}
		newUser := User{
			Email:    dbUser["email"].(string),
			Username: dbUser["username"].(string),
			Password: dbUser["password"].(string),
			Name:     dbUser["name"].(string),
		}
		userMap[newUser.Username] = newUser
	}
	return userMap

}

/*
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	//var agnetChan = make(chan []string, 0)
	var agentRep = make(chan map[string]Report)
	mongo_uri := os.Getenv("MONGOSTRING")
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Fatal(err)
	}

	//go GetAgentReportList(client, "agent_Go4", agnetChan)
	//var val = agnetChan
	//fmt.Println(<-val)

	go GetAgentReports(client, "agent_Go4", "JOPOUTCO:STAGE", agentRep)
	var val = agentRep
	fmt.Println(<-val)
}
*/
