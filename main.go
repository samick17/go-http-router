package main

import (
    "bufio"
	"fmt"
	"os"
	"net/http"
	"io"
	"./env"
	"./routes"
)

type commandActionCallback func(command string)

func prompt(fn commandActionCallback) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		fn(text)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func main() {
	var goEnv = env.Variables["go_env"]
	var debugMode = false
	if goEnv == "production" {
		debugMode = false
	} else {
		debugMode = true
	}
	fmt.Println("debugMode ", debugMode)
	routes.SetDebugMode(debugMode)
	router := routes.GetInstance()
	router.GET("/api/v2", func(w http.ResponseWriter, r *http.Request, p routes.Params) {
		w.WriteHeader(200)
		io.WriteString(w, "{\"data\":\"success!\"}")
		})
	router.POST("/api/v2/model/:modelId", func(w http.ResponseWriter, r *http.Request, p routes.Params) {
		name := p.Get("modelId")
		io.WriteString(w, "{\"data\":\""+name+"\"}")
		})
	err := http.ListenAndServe(":8270", router)
	if err != nil {
		panic(":" + err.Error())
	}
}
