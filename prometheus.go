package main

import "github.com/prometheus/client_golang/prometheus"

var commentsCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "comment_saved",
	Help: "When a new comment is saved",
})

var duplicatesGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "duplicate_comment",
	Help: "When a comment is a duplicate",
})
