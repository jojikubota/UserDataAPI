/*
Class:		CMPE 273-01
Assignment:	assignment2
Name:		Joji Kubota
Email:		joji.kubota@sjsu.edu
SID:		010404602
*/

package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"

	"github.com/drone/routes"
	"github.com/mkilling/goejdb"
	"github.com/naoina/toml"
	"labix.org/v2/mgo/bson"
)

// Struct to hold toml configurations
type tomlConfig struct {
	Database struct {
		File_name string
		Port_num  int
	}
	Replication struct {
		Rpc_server_port_num int
		Replica             []string
	}
}

// Replica server listener
type Listener int

// Map _id, map[oid]email (oid = BSON object identifier)
var oidMap map[string]string

// Global config object
var config tomlConfig

// Helper to setup TOML configuration
func ConfigureTOML(tomlFile string) {
	// Open toml config file
	f, err := os.Open(tomlFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Read in the file into config object
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	if err := toml.Unmarshal(buf, &config); err != nil {
		panic(err)
	}
}

// Helper to setup RPC server
func LaunchRpcServer() {
	// Setup replica server
	addy, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(config.Replication.Rpc_server_port_num))
	if err != nil {
		panic(err)
	}

	// Set listener
	inbound, err := net.ListenTCP("tcp", addy)
	if err != nil {
		panic(err)
	}

	// Launch
	log.Println("RPC Server Listening to... http://0.0.0.0:" + strconv.Itoa(config.Replication.Rpc_server_port_num))
	listener := new(Listener)
	rpc.Register(listener)
	rpc.Accept(inbound)

}

// Respond to client Post Replicate Request
func (l *Listener) ReplicatePost(profile map[string]interface{}, ack *bool) error {
	// Open db
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	// Get or create collection 'profiles'
	coll, _ := jb.CreateColl("profiles", nil)

	// Save profile in db
	email := profile["email"].(string)
	bsProfile, _ := bson.Marshal(profile)
	oidMap[email], _ = coll.SaveBson(bsProfile)

	// Close db
	jb.Close()

	return nil
}

// Respond to client Put Replicate Request
func (l *Listener) ReplicatePut(profile map[string]interface{}, ack *bool) error {
	// Open db
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	// Get or create collection 'profiles'
	coll, _ := jb.GetColl("profiles")

	// Remove exisiting profile
	email := profile["email"].(string)
	coll.RmBson(oidMap[email])
	delete(oidMap, email)

	// Save new profile in db
	bsProfile, _ := bson.Marshal(profile)
	oidMap[email], _ = coll.SaveBson(bsProfile)

	// Close db
	jb.Close()

	return nil
}

// Respond to client Del Replicate Request
func (l *Listener) ReplicateDel(profile map[string]interface{}, ack *bool) error {
	// Open db
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER)
	if dbErr != nil {
		os.Exit(1)
	}

	// Get or create collection 'profiles'
	coll, _ := jb.GetColl("profiles")

	// Remove exisiting profile
	email := profile["email"].(string)
	coll.RmBson(oidMap[email])
	delete(oidMap, email)

	// Close db
	jb.Close()

	return nil
}

// Helper to update replica
func UpdateReplica(method string, profile map[string]interface{}) {

	// Call rpc server
	numReplica := len(config.Replication.Replica)
	for i := 0; i < numReplica; i++ {

		client, err := rpc.Dial(
			"tcp", config.Replication.Replica[i])
		if err != nil {
			panic(err)
		}

		// //**** FOR TESTING ****//
		// client, err := rpc.Dial("tcp", "localhost:3002")
		// if err != nil {
		// 	log.Fatal(err)
		// }

		if method == "POST" {
			var reply bool
			err = client.Call("Listener.ReplicatePost", profile, &reply)
			if err != nil {
				log.Fatal(err)
			}
		} else if method == "PUT" {
			var reply bool
			err = client.Call("Listener.ReplicatePut", profile, &reply)
			if err != nil {
				log.Fatal(err)
			}
		} else if method == "DEL" {
			var reply bool
			err = client.Call("Listener.ReplicateDel", profile, &reply)
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Println("Calling Replica at... http://" + config.Replication.Replica[i])
	}
}

// Main funciton
func main() {
	// Setup toml config
	tomlFile := os.Args[1]
	ConfigureTOML(tomlFile)

	// Setup Oid tracker (oid = BSON object identifier)
	oidMap = make(map[string]string)

	// Register to gob (encoder/decoder)
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(bson.ObjectId(""))

	// Setup rpc server
	go LaunchRpcServer()

	// Setup new rounter for rest api
	mux := routes.New()

	// Handle different rest calls
	mux.Post("/profile", PostProfile)
	mux.Get("/profile/:email", GetProfile)
	mux.Put("/profile/:email", PutProfile)
	mux.Del("/profile/:email", DeleteProfile)

	// Run rest api server
	http.Handle("/", mux)
	log.Println("REST API Listening to... http://localhost:" + strconv.Itoa(config.Database.Port_num))
	http.ListenAndServe(":"+strconv.Itoa(config.Database.Port_num), nil)
}

// POST
func PostProfile(w http.ResponseWriter, r *http.Request) {
	// Read in the http body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	// Close
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	// Unmarshal http body.
	var profile map[string]interface{}
	if err := json.Unmarshal(body, &profile); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}
	// fmt.Println(profile)

	// Open db
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	// Get or create collection 'profiles'
	coll, _ := jb.CreateColl("profiles", nil)

	// Check if the profile exists
	email := profile["email"].(string)
	res, _ := coll.Find(`{"email": "` + email + `"}`)
	if len(res) != 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusConflict)
		jb.Close()
		return
	}
	// Save profile in db
	bsProfile, _ := bson.Marshal(profile)
	oidMap[email], _ = coll.SaveBson(bsProfile)

	// Close db
	jb.Close()

	// Replicate
	UpdateReplica("POST", profile)

	// Set header
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
}

// GET
func GetProfile(w http.ResponseWriter, r *http.Request) {
	// Parse params
	params := r.URL.Query()
	email := params.Get(":email")

	// Open db
	jb, err := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if err != nil {
		os.Exit(1)
	}

	// Get or create collection 'profiles'
	coll, _ := jb.GetColl("profiles")

	// Search the db
	res, _ := coll.Find(`{"email": "` + email + `"}`)
	// Not found
	if len(res) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		jb.Close()
		return
	} else { // Found. Send it as http response
		for _, bs := range res {
			var profile map[string]interface{}
			bson.Unmarshal(bs, &profile)
			// fmt.Println(profile)
			delete(profile, "_id") // remove extra field
			if err := json.NewEncoder(w).Encode(profile); err != nil {
				panic(err)
			}
			// routes.ServeJson(w, profile)
		}
	}

	// Close db
	jb.Close()

	// Set header
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

// PUT
func PutProfile(w http.ResponseWriter, r *http.Request) {
	// Parse params
	params := r.URL.Query()
	email := params.Get(":email")

	// Read in the http body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	// Unmarshal the body
	var newProfile map[string]interface{}
	if err := json.Unmarshal(body, &newProfile); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) // unprocessable entity
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	// Open db
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	// Get or create collection 'profiles'
	coll, _ := jb.GetColl("profiles")

	// Check if the profile exists
	var oldProfile map[string]interface{} // Used later in else
	res, _ := coll.Find(`{"email": "` + email + `"}`)
	if len(res) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		jb.Close()
		return
	} else { // Found
		for _, bs := range res {
			bson.Unmarshal(bs, &oldProfile)
		}
	}

	// Update data
	// ** oldProfide eventually becomes updateProfile **
	for key, _ := range newProfile {
		for k, _ := range oldProfile {
			if key == k {
				oldProfile[k] = newProfile[key]
			}
		}
	}
	fmt.Println(oldProfile)
	fmt.Println(newProfile)

	// Remove exisiting profile
	coll.RmBson(oidMap[email])
	delete(oidMap, email)

	// Save updated profile in db
	bsProfile, _ := bson.Marshal(oldProfile)
	oidMap[email], _ = coll.SaveBson(bsProfile)

	// Close db
	jb.Close()

	// Replicate
	UpdateReplica("PUT", oldProfile)

	// Set header
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusNoContent)

}

// DELETE
func DeleteProfile(w http.ResponseWriter, r *http.Request) {
	// Parse params
	params := r.URL.Query()
	email := params.Get(":email")

	// Open db
	jb, err := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER)
	if err != nil {
		os.Exit(1)
	}

	// Get or create collection 'profiles'
	coll, _ := jb.GetColl("profiles")

	// Check if the profile exists
	res, _ := coll.Find(`{"email": "` + email + `"}`)
	if len(res) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		jb.Close()
		return
	}

	// Remove profile
	coll.RmBson(oidMap[email])
	delete(oidMap, email)

	// Close db
	jb.Close()

	// Replicate
	var profile = make(map[string]interface{})
	profile["email"] = email
	UpdateReplica("DEL", profile)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusNoContent)

}
