package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	shortLivingToken = ""
)

const (
	filename = "/mnt/api/config.txt"
)

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
	if r.Method != "GET" {
		returnError(w, "Method not supported!", http.StatusMethodNotAllowed)
		return
	}

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

// run the server
func runServer() {
	http.HandleFunc("/ocm_token", getToken)
	http.HandleFunc("/write_kafka_config", writeKafkaConfig)
	err := http.ListenAndServe(":8099", nil)
	if err != nil {
		log.Fatalf("Unable to start the server: %s", err.Error())
	}
}

type configStruct struct {
	BootstrapUrl string
	Username     string
	Password     string
}

func writeKafkaConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		returnError(w, "Method not supported!", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		returnError(w, fmt.Sprintf("Unable to read request body: %s", err), http.StatusInternalServerError)
		return
	}

	var c configStruct
	err = json.Unmarshal(body, &c)
	if err != nil {
		returnError(w, fmt.Sprintf("Unable to unmarshal request body: %s", err), http.StatusInternalServerError)
		return
	}

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
	if !strings.Contains(currentConfig, c.BootstrapUrl) {
		configString := fmt.Sprintf("---\nbootstrapURL: \"%s\"\nusername: %s\npassword: %s\n", c.BootstrapUrl, c.Username, c.Password)
		if _, err := file.WriteString(configString); err != nil {
			returnError(w, fmt.Sprintf("Unable to write to file: %s", err), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent) // respond with 204 on successful config write
	defer file.Close()
}

func returnError(w http.ResponseWriter, err string, statusCode int) {
	w.WriteHeader(statusCode) //nolint
	w.Write([]byte(err))      //nolint
}

// login to ocm and serve an endpoint to retrieve short living ocm tokens
func main() {
	loginToOcm()
	runServer()
}
