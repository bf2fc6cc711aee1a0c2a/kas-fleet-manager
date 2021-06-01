package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	shortLivingToken       = ""
	kafkaCreateContainerId = ""
)

const (
	configFilename    = "/mnt/api/config.txt"
	kafkaIdsFilemane  = "/mnt/api/kafkas.txt"
	svcAccIdsFilemane = "/mnt/api/service_accounts.txt"
)

type configStruct struct {
	BootstrapUrl string
	Username     string
	Password     string
}

type kafkaIdStruct struct {
	KafkaId string
}

type svcAccIdStruct struct {
	ServiceAccountId string
}

type containerIdStruct struct {
	ContainerId string
}

// login to ocm via shell (OCM_OFFLINE_TOKEN env var required)
func loginToOcm() {
	token := os.Getenv("OCM_OFFLINE_TOKEN")
	if token == "" {
		log.Fatal("No offline token provided!")
	}
	_, err := exec.Command("ocm", "login", "--url=https://api.stage.openshift.com/", fmt.Sprintf("--token=%s", token)).Output()
	if err != nil {
		log.Print(err)
	}
}

// get short living token and return its value via http response
func getToken(w http.ResponseWriter, r *http.Request) {
	checkHttpMethod(w, r, "GET")

	if shortLivingToken == "" || !isTokenValid(string(shortLivingToken)) {
		tokenOutput, err := exec.Command("ocm", "token").Output()
		if err != nil {
			returnError(w, fmt.Sprintf("Unable to return ocm token: %s", err), http.StatusInternalServerError)
		}
		shortLivingToken = string(tokenOutput)
	}
	fmt.Fprint(w, shortLivingToken)
}

// verify short living token validity
func isTokenValid(tokenOutput string) bool {
	token, err := jwt.Parse(string(tokenOutput), nil)
	if token == nil {
		log.Print(err)
		return false
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	expTimestamp := int64(claims["exp"].(float64))
	now := time.Now()
	currentTime := now.Unix()
	return expTimestamp > currentTime
}

func checkKafkaCreateContainerId(w http.ResponseWriter, r *http.Request) {
	checkHttpMethod(w, r, "POST")

	var c containerIdStruct

	unmarshalBody(w, r, &c)

	// assign kafkaCreateContainerId, if empty
	if kafkaCreateContainerId == "" && c.ContainerId != "" {
		kafkaCreateContainerId = c.ContainerId
	}
	// return containerID
	fmt.Fprint(w, kafkaCreateContainerId)
}

// run the server
func runServer() {
	ctx, cancel := context.WithCancel(context.Background())

	http.HandleFunc("/ocm_token", getToken)
	http.HandleFunc("/write_kafka_config", writeKafkaConfig)
	http.HandleFunc("/write_kafka_id", writeKafkaId)
	http.HandleFunc("/write_svc_acc_id", writeSvcAccId)
	http.HandleFunc("/kafka_create_container_id", checkKafkaCreateContainerId)

	srv := &http.Server{Addr: ":8099"}
	go func() {
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatalf("Unable to start the server: %s", err.Error())
		}
	}()

	time.AfterFunc(getRunningTime(), func() { cancel() })

	<-ctx.Done()

	// shutdown the server after perf test is complete
	err := srv.Shutdown(context.Background())
	if err != nil {
		log.Fatalf("Unable to stop the server: %s", err.Error())
	}
}

func checkHttpMethod(w http.ResponseWriter, r *http.Request, method string) {
	if r.Method != method {
		returnError(w, "Method not supported!", http.StatusMethodNotAllowed)
		return
	}
}

func unmarshalBody(w http.ResponseWriter, r *http.Request, str interface{}) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		returnError(w, fmt.Sprintf("Unable to read request body: %s", err), http.StatusInternalServerError)
		return
	}
	err = json.Unmarshal(body, &str)
	if err != nil {
		returnError(w, fmt.Sprintf("Unable to unmarshal request body: %s", err), http.StatusInternalServerError)
		return
	}
}

func writeToFile(w http.ResponseWriter, filename string, checkExists string, config string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		returnError(w, fmt.Sprintf("Unable to open/create config file: %s", err), http.StatusInternalServerError)
		return
	}

	// reading currently saved config
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		returnError(w, fmt.Sprintf("Unable to read config file: %s", err), http.StatusInternalServerError)
		return
	}

	// Convert []byte to string and print to screen
	currentConfig := string(content)

	// checking if current config contains config to be persisted
	if !strings.Contains(currentConfig, checkExists) {
		if _, err := file.WriteString(config); err != nil {
			returnError(w, fmt.Sprintf("Unable to write to file: %s", err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent) // respond with 204 on successful config write
	defer file.Close()
}

func writeSvcAccId(w http.ResponseWriter, r *http.Request) {
	checkHttpMethod(w, r, "POST")

	var c svcAccIdStruct

	unmarshalBody(w, r, &c)

	configString := fmt.Sprintf("%s\n", c.ServiceAccountId)

	writeToFile(w, svcAccIdsFilemane, c.ServiceAccountId, configString)
}

func writeKafkaId(w http.ResponseWriter, r *http.Request) {
	checkHttpMethod(w, r, "POST")

	var c kafkaIdStruct

	unmarshalBody(w, r, &c)

	configString := fmt.Sprintf("%s\n", c.KafkaId)

	writeToFile(w, kafkaIdsFilemane, c.KafkaId, configString)
}

func writeKafkaConfig(w http.ResponseWriter, r *http.Request) {
	checkHttpMethod(w, r, "POST")

	var c configStruct

	unmarshalBody(w, r, &c)

	configString := fmt.Sprintf("---\nbootstrapURL: \"%s\"\nusername: %s\npassword: %s\n", c.BootstrapUrl, c.Username, c.Password)

	writeToFile(w, configFilename, c.BootstrapUrl, configString)
}

func returnError(w http.ResponseWriter, err string, statusCode int) {
	w.WriteHeader(statusCode) //nolint
	w.Write([]byte(err))      //nolint
}

func getRunningTime() time.Duration {
	runTimeStr := os.Getenv("PERF_TEST_RUN_TIME")
	if !strings.HasSuffix(string(runTimeStr), "m") { // <number_of_minutes>m format expected
		log.Fatal(fmt.Printf("Unable to get run time length variable %s", runTimeStr))
	}
	runes := []rune(runTimeStr)
	runTimeMinutes := string(runes[0 : len(runTimeStr)-1])
	if parsedMinutes, err := strconv.Atoi(runTimeMinutes); err == nil {
		return time.Duration(parsedMinutes*60+30) * time.Second // 30 seconds extra to allow perf test to complete
	} else {
		return 0 * time.Second
	}
}

// login to ocm and serve an endpoint to retrieve short living ocm tokens
func main() {
	loginToOcm()
	getRunningTime()
	runServer()
}
