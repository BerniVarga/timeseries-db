# Timeseries DB implementation with MongoDB

## Code structure
The code consist of 3 components:
* db - contains a docker compose setup for running a MongoDB instance, that will stores the timeseries data
* ingestion - a script that is run over the DB (ingestion is run from within the docker container, after the file is copied over)
* api - an api that allows querying the data for specific time ranges and aggregations by frequency period

## How to run it

Each module describes the running instructions needed, but in order for you to run the code, you need:
* Golang installed (>1.7)
* Docker installed
* bash shell

1. Start the docker container from within the `db` directory with:  
`./run.sh`  
After you finished, stop the container with:  
`./stop.sh`


2. Run the ingestion script - any time you consider it. 
In an ideal future there would be a cronjob set up that runs it every 5 minutes to observe/save behaviour continously.
The ingestion script should also be run from the `ingestion` folder: 
`./ingest.sh`

3. Start the API. 
The Api can be started even before the ingestion layer has run.  
`go run main.go`

Some interesting queries that you can run:

`curl "localhost:8080/metrics/concurrency?start=1501681460&end=1650843741&frequency=years" | jq`   
Returns all the saved concurency metrics within the given time range, with aggregations done /years, displaying the avarage for each. Results could look like:
```
[
  {
    "timestamp": "2017-01-01T00:00:00Z",
    "concurrency": 169070
  },
  {
    "timestamp": "2022-01-01T00:00:00Z",
    "concurrency": 263415
  }
]
```


`curl "localhost:8080/metrics/cpu_load?start=1501681460&end=1650843741&frequency=minutes"`  
`curl "localhost:8080/metrics?start=1501681460&end=1650843741&frequency=minutes"`  

`curl "localhost:8080/metrics/average?start=1501681460&end=1650843741"`  
`curl "localhost:8080/metrics/cpu_load/average?start=1501681460&end=1650843741"`  
`curl "localhost:8080/metrics/concurrency/average?start=1501681460&end=1650843741"`  


## Future TODO list/known limitation:

DB level:
- Authentication to Mongo should use secrets
- Authentication for the script executer needs to be set up

API level TODOs:
* change the Logging - use Logger
* revise parameters - http server, mongo client
* extend query methods with more aggregation; query for a day (without using the range)
* Mongo - data from Mongo; read into a channel and considering streaming it to the client or offer pagination
* write more tests (cover more testcases, and add tests for the handler repo)
* add a health endpoint

Ingestion Level:
* create a cron job
* make ingestion through a client, instead of copying the script inside the container and running it there
