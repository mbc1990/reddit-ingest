package main

import "encoding/json"
import "github.com/prometheus/client_golang/prometheus"
import "net/http"
import "fmt"
import "time"
import "os"

// Configuration struct that conf json file is read into
type Configuration struct {
	PGHost           string
	PGPort           int
	PGUser           string
	PGPassword       string
	PGDbname         string
	TargetSubreddits []string // Subreddits whose content will be consumed
	NumWorkers       int
	Username         string
	Password         string
	Secret           string
	ClientId         string
	PrometheusPort   string
	RunEverySeconds  int
}

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		fmt.Println("Usage: ./main <absolute path to configuration file>")
		return
	}
	file, _ := os.Open(args[0])
	decoder := json.NewDecoder(file)
	var conf = Configuration{}
	err := decoder.Decode(&conf)
	if err != nil {
		fmt.Println("error:", err)
	}

	prometheus.MustRegister(commentsCounter)
	prometheus.MustRegister(duplicatesGauge)
	http.Handle("/metrics", prometheus.Handler())
	go http.ListenAndServe(conf.PrometheusPort, nil)

	ing := NewRedditIngester(&conf)

	// Run forever, sleeping between runs
	for {
		go ing.Run()
		time.Sleep(time.Duration(conf.RunEverySeconds) * time.Second)
	}
}
