package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gorilla/mux"
	"github.com/terakilobyte/rhetor/container"
	"github.com/terakilobyte/rhetor/filesystem"
)

const minPort int = 3000
const maxPort int = 3999
const portOffset int = 2000

var usedPorts = make(map[int]string)
var studentContainerMapping = make(map[string]string)
var aws *session.Session

// APIResponse represents possible API responses
type APIResponse struct {
	ContainerID string `json:"containerID,omitempty"`
	DevPort     string `json:"devPort,omitempty"`
	AppPort     string `json:"appPort,omitempty"`
	Error       string `json:"error,omitempty"`
	Destroyed   bool   `json:"destroyed,omitempty"`
}

// HandleProvisionRequest handles calling the provisioning service
func HandleProvisionRequest(w http.ResponseWriter, r *http.Request) {
	fs, err := filesystem.New("terakilobyte", "M220P")
	if err != nil {
		fmt.Println(err.Error())
	}
	var request container.ProvisionRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	port := getRandomPort()
	request.DevPort = strconv.Itoa(port)
	request.AppPort = strconv.Itoa(port + 2000)
	request.FS = fs
	request.AWS = aws
	cid, err := container.Provision(request)
	var apiResponse APIResponse
	if err != nil {
		apiResponse.Error = err.Error()
		json.NewEncoder(w).Encode(apiResponse)
		return
	}
	apiResponse.ContainerID = cid
	apiResponse.DevPort = request.DevPort
	apiResponse.AppPort = request.AppPort

	// This operation should live on the container servers. If the server dies,
	// all of the containers will die with it.
	usedPorts[port] = request.StudentID

	// This should be centralized to MongoDB
	// A future iteration would have this be stored in MongoDB
	// and the central load balancer would handle storing data in the
	// studentContainerMapping
	// This should be separate in case the load balancing server crashes but the
	// container servers do not.
	studentContainerMapping[request.StudentID] = cid
	json.NewEncoder(w).Encode(apiResponse)
}

// HandleDestroyRequest handles calling the destroy service
func HandleDestroyRequest(w http.ResponseWriter, r *http.Request) {
	fs, err := filesystem.New("terakilobyte", "M220P")
	if err != nil {
		fmt.Println(err.Error())
	}
	var request container.DestroyRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	request.FS = fs
	request.AWS = aws
	var apiResponse APIResponse
	if err := container.Destroy(request); err != nil {
		apiResponse.Error = err.Error()
	} else {
		delete(usedPorts, request.Port)
		delete(studentContainerMapping, request.StudentID)
		apiResponse.Destroyed = true
	}
	json.NewEncoder(w).Encode(apiResponse)
}

func getRandomPort() int {
	port := rand.Intn(maxPort-minPort) + minPort
	if _, ok := usedPorts[port]; ok {
		return getRandomPort()
	}
	return port
}

func getIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func main() {
	sess, err := filesystem.Init()
	if err != nil {
		panic(err)
	}
	aws = sess
	rand.Seed(time.Now().Unix())
	router := mux.NewRouter()
	router.HandleFunc("/provision", HandleProvisionRequest).Methods("POST")
	router.HandleFunc("/destroy", HandleDestroyRequest).Methods("POST")
	log.Fatal(http.ListenAndServe(":8000", router))
}
