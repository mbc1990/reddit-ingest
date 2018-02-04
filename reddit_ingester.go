package main

import "log"

// import "strconv"
import "net/http"
import "encoding/json"
import "encoding/base64"
import "sync"
import "bytes"
import "io/ioutil"
import "fmt"

// TODO: Replace this with ResponsePrimative
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
type ResponsePrimative struct {
	Kind string
	Data struct {
		Created_UTC int
		Body        string
		Title       string
		Selftext    string
		Children    *[]ResponsePrimative
		Replies     *ResponsePrimative
	}
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

// Parses a response tree for comments and writes them to postgres
func (r *RedditIngester) ParseTreeForComments(tree *ResponsePrimative) {
	// TODO: This method just prints the comments right now. Next step is writing them to postgres
	switch tree.Kind {
	case "t3":
		fmt.Println(tree.Data.Title)
		fmt.Println(tree.Data.Selftext)
	case "t1":
		fmt.Println(tree.Data.Body)
		// Don't recurse if it's an empty struct (leaf node)
		if *tree.Data.Replies != (ResponsePrimative{}) {
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
			subredditResponse := new(SubredditResponse)
			json.Unmarshal(body, &subredditResponse)
			for _, story := range subredditResponse.Data.Children {
				url := story.Data.Permalink
				ji := new(JobInfo)
				ji.URL = url
				ji.PageType = "comments"
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
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			commentResponse := make([]ResponsePrimative, 0)
			json.Unmarshal(body, &commentResponse)

			// A comment response is an array of trees, so send each off
			// to the recursive tree parser
			for _, topLevelNode := range commentResponse {
				r.ParseTreeForComments(&topLevelNode)
			}

		} else {
			log.Fatal("Unexpected job type")
		}
	}
}

// Entry point of a single run
// This program will be run as a cron job and will handle its own deduplication
func (r *RedditIngester) Run() {
	for _, subreddit := range r.Conf.TargetSubreddits {
		job := JobInfo{}
		job.URL = r.BaseURL + "r/" + subreddit
		job.PageType = "subreddit"
		r.WorkQueue <- job
	}
	r.Wg.Wait()
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

func NewRedditIngester(conf *Configuration) *RedditIngester {
	r := new(RedditIngester)
	r.Conf = conf
	r.BaseURL = "https://oauth.reddit.com/"

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
