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

var workQueueGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "work_queue",
	Help: "Number of jobs in the work queue",
})

var reAuthGauge = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "re_auth",
	Help: "Every time we reauthenticate",
})
