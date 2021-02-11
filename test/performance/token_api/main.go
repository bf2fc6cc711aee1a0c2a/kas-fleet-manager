package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

var (
	shortLivingToken = ""
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
		w.WriteHeader(http.StatusNotFound)          //nolint
		w.Write([]byte("Method is not supported.")) //nolint
		return
	}

	if shortLivingToken == "" || !isTokenValid(string(shortLivingToken)) {
		tokenOutput, err := exec.Command("ocm", "token").Output()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError) //nolint
			w.Write([]byte(err.Error()))                  //nolints
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
	err := http.ListenAndServe(":8099", nil)
	if err != nil {
		log.Fatalf("Unable to start the server: %s", err.Error())
	}
}

// login to ocm and serve an endpoint to retrieve short living ocm tokens
func main() {
	loginToOcm()
	runServer()
}
