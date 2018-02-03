package main

import "encoding/json"
import "fmt"
import "os"

// Configuration struct that conf json file is read into
type Configuration struct {
	PGHost           string
	PGPort           int
	PGUser           string
	PGPassword       string
	PGDbname         string
	TargetSubreddits []string // Subreddits whose content will be consumed
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

	ing := NewRedditIngester(&conf)
	ing.Run()
}
