package main

import "log"
import "net/http"
import "encoding/json"
import "encoding/base64"
import "sync"
import "bytes"
import "io/ioutil"
import "fmt"

type SubredditResponse struct {
}

type CommentsResponse struct {
}

type JobInfo struct {
	URL      string // URL to fetch
	PageType string // "subreddit" or "comments"
}

type RedditIngester struct {
	Conf        *Configuration
	WorkQueue   chan JobInfo
	BaseURL     string
	Wg          *sync.WaitGroup
	AccessToken string
}

func (r *RedditIngester) Worker() {
	for info := range r.WorkQueue {
		if info.PageType == "subreddit" {
			url := "https://oauth.reddit.com/r/cryptocurrencies"
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("Authorization", "Bearer "+r.AccessToken)
			req.Header.Add("User-Agent", r.Conf.ClientId+"by "+r.Conf.Username)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Println(string(body))
		} else if info.PageType == "comments" {
			// TODO: Traverse comments trees
			// TODO: Extract time, content, unique id
			// TODO: Write to postgres if unique
		} else {
			log.Fatal("Unexpected job type")
		}

		// TODO: If we want to terminate a worker early, handle that message here
		// TODO: if terminate then call r.wg.Done()
	}
}

// Entry point of a single run
// This program will be run as a cron job and will handle its own deduplication
func (r *RedditIngester) Run() {
	for _, subreddit := range r.Conf.TargetSubreddits {
		job := JobInfo{}
		job.URL = r.BaseURL + subreddit
		job.PageType = "subreddit"
		r.WorkQueue <- job
	}
	r.Wg.Wait()
}

type AuthResponse struct {
	Access_token string
	Error        int
}

// Goes through the reddit OAuth flow
func (r *RedditIngester) Authenticate() {
	url := "https://www.reddit.com/api/v1/access_token"
	client := &http.Client{}
	bodyToSend := bytes.NewBuffer([]byte("grant_type=client_credentials&\\device_id=1"))
	req, _ := http.NewRequest("POST", url, bodyToSend)

	toEncode := []byte(r.Conf.ClientId + ":" + r.Conf.Secret)
	toSend := base64.StdEncoding.EncodeToString(toEncode)
	req.Header.Add("Authorization", "Basic "+toSend)
	req.Header.Add("User-Agent", r.Conf.ClientId+"by "+r.Conf.Username)
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	authResp := new(AuthResponse)
	json.Unmarshal(body, &authResp)
	r.AccessToken = authResp.Access_token
}

func NewRedditIngester(conf *Configuration) *RedditIngester {
	r := new(RedditIngester)
	r.Conf = conf
	r.BaseURL = "https://www.reddit.com/r/"

	var wg sync.WaitGroup
	r.Wg = &wg

	// Reddit has an unauthenticated API but it's far too rate limited
	r.Authenticate()

	// Create and populate worker queue
	r.WorkQueue = make(chan JobInfo)
	for i := 0; i < r.Conf.NumWorkers; i++ {
		r.Wg.Add(1)
		go r.Worker()
	}
	return r
}
