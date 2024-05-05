# poki_assignment

## Approach

### Understanding of the problem
Client sends a POST request with `seed` and `value` int64 values where `seed` is the number with which client generates random number. With same `seed` value, the order of random numbers are always same.<br><br>
When server recieves the payload, it created the player with provided `seed` value and generates a random number independently. When both generated random numbers match, score increases.<br><br>
As per server code, it will stop processing requests when first mismatch of values happen or timer duration expires.<br><br>
Which means the the order of generation of random number with `seed` value (and the order of subsequent sending of requests) must match with the order of generation of random numbers on the server. And this should happen within the given duration. The client has to get all requests right and without getting a single value wrong.

### Approach
Firing requests in sequential order is very slow to win the game (server waits for 100 ms before returning response). Hence I tried to use concurrency by using goroutines and schedule them using **Worker-pool** strategy.<br><br>

- Create and store `numClient` http clients to use them in circular order.
- Pre-compute and store `MaxJobs` number of requests in a slice so that we save time at the time of making the POST request.
- Create a `dispatcher` which will create `MaxWorkers` number of workers, each having its own channel to recieve the job.
- `dispatcher` has `WorkerPool`, a channel of `chan int` which has individual input channel of each worker.
- There is a separate `JobQueue` channel to send the actual job to. `dispatcher` waits to recieve jobs on this channel
- Once a job is recieved on `JobQueue` channel, the `dispatcher` takes out a channel from `WorkerPool` and sends the job on that channel.
- The worker who owns that individual channel processes the request concurrently.
- -  Each worker first acquires a mutex lock. Inside mutex - 
  -  worker gets the index (to maintain order of request)
  -  Takes the pre-computed request at that index from the array
  -  send HTTP POST request concurrently
  -  wait for few milliseconds to ensure the order
  -  release mutex lock

 ### Result
 I tried many approaches but could not ensure the order of request reaching the server. Waiting for few milliseconds after firing each request seems to allow some deterministic behavior of order of execution of goroutines but still not able to win the game. Other than that, couldn't seem to find a deterministic way to ensure order without relying on delay in sending requests via goroutines.<br><br>
On Mac M3 laptop, i was able to reach ~11000 with `WaitTime` of 1 ms. Please change `WaitTime` at line 116 to play with different delay value (in ms).
