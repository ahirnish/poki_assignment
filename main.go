package main

import (
  "fmt"
  "bytes"
  "net/http"
  "io/ioutil"
  "math/rand"
  "sync"
  "time"
  "os"
  "strconv"
)


var JobQueue = make(chan int)

var wg sync.WaitGroup

type Worker struct {
	WorkerPool  chan chan int
	JobChannel  chan int
	quit    	chan bool
    quitChannel chan chan bool
}

func NewWorker(workerPool chan chan int) Worker {
	return Worker{
		WorkerPool: workerPool,
		JobChannel: make(chan int),
		quit:       make(chan bool)}
}

func (w Worker) Start() {
	go func() {
		for {
			// register the current worker into the worker queue.
			w.WorkerPool <- w.JobChannel

			select {
			case job := <-w.JobChannel:
				// we have received a work request.
				worker_job(job)

			case <-w.quit:
				// we have received a signal to stop
				return
			}
		}
	}()
}

func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

func worker_job(index int) {
    callMutex.Lock()
    defer wg.Done()
    defer callMutex.Unlock()
    client := clients[globalIndex%numClient]
    request := requests[globalIndex]
    globalIndex = globalIndex + 1
    go client.Do(request)
    time.Sleep(time.Duration(WaitTime) * time.Millisecond)
}

type Dispatcher struct {
	// A pool of workers channels that are registered with the dispatcher
	WorkerPool chan chan int
    MaxWorkers int
    Workers []Worker
}

func NewDispatcher(maxWorkers int) *Dispatcher {
	pool := make(chan chan int, maxWorkers)
	return &Dispatcher{WorkerPool: pool, MaxWorkers: maxWorkers}
}

func (d *Dispatcher) Run() {
    // starting n number of workers
	for i := 0; i < d.MaxWorkers; i++ {
		worker := NewWorker(d.WorkerPool)
		worker.Start()
        d.Workers = append(d.Workers, worker)
	}

	go d.dispatch()
}

func (d *Dispatcher) Stop() {
    for _, worker := range d.Workers {
        worker.Stop()
    }
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-JobQueue:
			// a job request has been received
            jobChannel := <-d.WorkerPool
            jobChannel <- job
		}
	}
}

var callMutex sync.Mutex
var globalIndex = 0

const numClient = 10
const MaxWorkers = 50
const MaxJobs = 15000
const WaitTime = 1

var clients []*http.Client
var requests []*http.Request


func main() {
    if len(os.Args) != 2 {
        fmt.Println("Usage: go run main.go <seed_value>")
        return
    }
    seed, errInput := strconv.Atoi(os.Args[1])
    if errInput != nil {
        fmt.Println("Error parsing input - ", errInput)
        return
    }
    fmt.Println("sending requests...")
    gen := rand.New(rand.NewSource(int64(seed)))
    start := time.Now()
    url := "http://localhost:1337"
    
    jsonData, err := ioutil.ReadFile("request_template.json")
    if err != nil {
        fmt.Println("Error reading request template:", err)
        return
    }

    for i:=0;i<numClient;i++{
    	clients = append(clients, &http.Client{})    	
    }


    for i:=0;i<MaxJobs;i++ {
    	value := gen.Int63()
        data := string(jsonData)
        data = fmt.Sprintf(data, seed, value)
        requestBody := bytes.NewReader([]byte(data))
        req, err := http.NewRequest(http.MethodPost, url, requestBody)
        if err != nil {
            fmt.Println("Error creating request:", err)
            return
        }
        req.Header.Set("Content-Type", "application/json")
        requests = append(requests, req)	
    }

    dispatcher := NewDispatcher(MaxWorkers)
    dispatcher.Run()

    for i:=0;i<MaxJobs;i++ {
        wg.Add(1)
        JobQueue <- i
    }
    
    wg.Wait()
    dispatcher.Stop()
    fmt.Printf("Total time to finish : %s \n", time.Since(start).String())
}