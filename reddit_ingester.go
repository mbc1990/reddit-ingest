package main

import "log"
import "sync"

type SubredditResponse struct {
}

type CommentsResponse struct {
}

type JobInfo struct {
	URL      string // URL to fetch
	PageType string // "subreddit" or "comments"
}

type RedditIngester struct {
	Conf      *Configuration
	WorkQueue chan JobInfo
	BaseURL   string
	Wg        *sync.WaitGroup
}

func (r *RedditIngester) Worker() {
	for info := range r.WorkQueue {
		if info.PageType == "subreddit" {
			// TODO: Get page
			// TODO: parse URLs
			// TODO: Send comments jobs to work queue

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
		job.URL = r.BaseURL + subreddit + ".json"
		job.PageType = "subreddit"
		r.WorkQueue <- job
	}
}

func NewRedditIngester(conf *Configuration) *RedditIngester {
	r := new(RedditIngester)
	r.Conf = conf
	r.BaseURL = "https://www.reddit.com/r/"
	r.WorkQueue = make(chan JobInfo)

	// Populate worker queue
	for i := 0; i < r.Conf.NumWorkers; i++ {
		r.Wg.Add(1)
		go r.Worker()
	}
	return r
}
