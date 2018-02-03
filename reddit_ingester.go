package main

import "sync"

type RedditIngester struct {
	Conf *Configuration
	Wg   *sync.WaitGroup
}

func (r *RedditIngester) Run() {
	// TODO: Fetch all the stuff
}

func (r *RedditIngester) Connect() {
	// TODO: Authentication logic
}

func NewRedditIngester(conf *Configuration) *RedditIngester {
	r := new(RedditIngester)

	// Do any necessary authentication
	r.Connect()

	r.Conf = conf
	return r
}
