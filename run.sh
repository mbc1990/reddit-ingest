#!/bin/bash
ulimit -n 1000000
./reddit-ingest conf.json
