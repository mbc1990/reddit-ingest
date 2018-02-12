package main

import "log"

import "strconv"
import "net/http"
import "encoding/json"
import "encoding/base64"

import "time"
import "bytes"
import "io/ioutil"
import "fmt"

type RedditIngester struct {
	Conf           *Configuration
	WorkQueue      chan JobInfo
	BaseURL        string
	AccessToken    string
	PostgresClient *PostgresClient
}

// TODO: Replace this with ResponsePrimitive
type SubredditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Selftext  string
				Permalink string
			}
		}
	}
}

// The reddit api returns trees of these objects which are identified by their "kind" field
type ResponsePrimitive struct {
	Kind string
	Data struct {
		Score                   int
		Created_utc             float64
		Id                      string
		Body                    string
		Subreddit_name_prefixed string
		Title                   string
		Selftext                string
		Children                *[]ResponsePrimitive
		Replies                 *ResponsePrimitive
	}
}

type JobInfo struct {
	URL      string // URL to fetch
	PageType string // "subreddit" or "comments"
}

// Parses a response tree for comments and writes them to postgres
func (r *RedditIngester) ParseTreeForComments(tree *ResponsePrimitive) {
	switch tree.Kind {
	case "t3":
		// TODO: Send these to postgres
		fmt.Println(tree.Data.Title)
		fmt.Println(tree.Data.Selftext)
	case "t1":
		// Insert into postgres if comment hasn't been seen before
		if !r.PostgresClient.CommentExists(tree.Data.Id) {
			commentsCounter.Inc()
			r.PostgresClient.InsertComment(tree.Data.Id, tree.Data.Subreddit_name_prefixed,
				tree.Data.Body, int(tree.Data.Created_utc))
			fmt.Println("ID: " + tree.Data.Id)
			fmt.Println("Created at: " + strconv.Itoa(int(tree.Data.Created_utc)))
			fmt.Println("Score: " + strconv.Itoa(tree.Data.Score))
			fmt.Println("Body: " + tree.Data.Body)
			fmt.Println("----------------------------------")
		} else {
			duplicatesGauge.Inc()
		}
		// Don't recurse if it's an empty struct (leaf node)
		if *tree.Data.Replies != (ResponsePrimitive{}) {
			r.ParseTreeForComments(tree.Data.Replies)
		}
	case "Listing":
		for _, child := range *tree.Data.Children {
			r.ParseTreeForComments(&child)
		}
	default:
		fmt.Println("Unexpected object type: " + tree.Kind)
	}
}

func (r *RedditIngester) Worker() {
	for info := range r.WorkQueue {
		// TODO: Refactor request construction out of the if/then cases
		if info.PageType == "subreddit" {
			url := info.URL
			fmt.Println("Getting stories for " + url)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("Authorization", "Bearer "+r.AccessToken)
			req.Header.Add("User-Agent", r.Conf.ClientId+"by "+r.Conf.Username)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			body, _ := ioutil.ReadAll(resp.Body)
			subredditResponse := new(SubredditResponse)
			err = json.Unmarshal(body, &subredditResponse)
			if err != nil {
				log.Fatal(err)
			}
			resp.Body.Close()
			fmt.Println("Response status: " + resp.Status)
			for _, story := range subredditResponse.Data.Children {
				url := story.Data.Permalink
				ji := new(JobInfo)
				ji.URL = url
				ji.PageType = "comments"
				fmt.Println("Adding " + url + " to queue")
				r.WorkQueue <- *ji
			}

		} else if info.PageType == "comments" {
			url := r.BaseURL + info.URL
			fmt.Println("Getting comments for " + url)
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Add("Authorization", "Bearer "+r.AccessToken)
			req.Header.Add("User-Agent", r.Conf.ClientId+"by "+r.Conf.Username)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}
			fmt.Println("Response status: " + resp.Status)
			body, _ := ioutil.ReadAll(resp.Body)
			commentResponse := make([]ResponsePrimitive, 0)
			json.Unmarshal(body, &commentResponse)
			resp.Body.Close()

			// A comment response is an array of trees, so send each off
			// to the recursive tree parser
			for _, topLevelNode := range commentResponse {
				r.ParseTreeForComments(&topLevelNode)
			}

		} else {
			fmt.Println("Unexpected job type")
		}
	}
}

// Entry point of a single run
// This program will be run as a cron job and will handle its own deduplication
func (r *RedditIngester) Run() {
	fmt.Println("Attempting run...")
	for _, subreddit := range r.Conf.TargetSubreddits {
		job := JobInfo{}
		job.URL = r.BaseURL + "r/" + subreddit
		job.PageType = "subreddit"
		r.WorkQueue <- job
	}
}

type AuthResponse struct {
	Access_token string
	Error        int
}

// Goes through the reddit Basic Authentication flow
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

func (r *RedditIngester) LogChannelInfo() {
	for {
		workQueueGauge.Add(float64(len(r.WorkQueue)))
		time.Sleep(time.Duration(5) * time.Second)
	}
}

func NewRedditIngester(conf *Configuration) *RedditIngester {
	r := new(RedditIngester)
	r.Conf = conf
	r.BaseURL = "https://oauth.reddit.com/"

	// Postgres
	r.PostgresClient = NewPostgresClient(r.Conf.PGHost, r.Conf.PGPort,
		r.Conf.PGUser, r.Conf.PGPassword, r.Conf.PGDbname)

	// Reddit has an unauthenticated API but it's far too rate limited
	r.Authenticate()

	// Create and populate worker queue
	r.WorkQueue = make(chan JobInfo, 50000)
	for i := 0; i < r.Conf.NumWorkers; i++ {
		go r.Worker()
	}
	go r.LogChannelInfo()
	return r
}
