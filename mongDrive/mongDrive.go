package mongDrive

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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
	Name	string
	Cpu		[]int16
	Drives	[]map[string]float32
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
// Returns list of maps with for given agent with common structure
func GetAgentReports(client *mongo.Client, db string, reports chan []Report) {

	reportCol := client.Database(db).Collection("Report")
	cur, err := reportCol.Find(context.TODO(), bson.D{}) //Get all the reports
	if err != nil {
		log.Fatal(err)
	}
	var repLst []Report

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
		repLst = append(repLst, newReport)
	}

	reports <- repLst

}

// Return Server Data about all current hydrologik servers
// Returns a list of structs about each given server 
func GetServerReports(client *mongo.Client, servers chan []Server)
