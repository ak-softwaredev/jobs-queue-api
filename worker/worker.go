package worker

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/expr-lang/expr"
)

var availableJobs = []string{
	"echo",
	"eval",
}

var jobs = make(map[string]Job) // Temporary until i decide to use a database
var mux = sync.RWMutex{}
var jobQueue = make(chan string, 1024)

var submit_mux sync.RWMutex
var shuttingDown bool

type Job struct {
	JobId   string `json:"job_id"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Status  string `json:"status"`
	Result  string `json:"result"`
}

func GetJob(w http.ResponseWriter, r *http.Request) {
	job_id := r.PathValue("id")

	mux.RLock()
	job, ok := jobs[job_id]
	mux.RUnlock()

	if !ok {
		http.Error(w, "job does not exist", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(job)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func PostJob(w http.ResponseWriter, r *http.Request) {
	submit_mux.RLock()
	defer submit_mux.RUnlock()

	if shuttingDown {
		http.Error(w, "service shutting down", http.StatusServiceUnavailable)
		return
	}

	job := Job{}
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	isAllowed := false
	for _, v := range availableJobs {
		if v == job.Type {
			isAllowed = true
			break
		}
	}
	if !isAllowed {
		http.Error(w, "job type not allowed", http.StatusBadRequest)
		return
	}

	jobId := rand.Text()[:16]

	mux.Lock()
	for jobs[jobId] != (Job{}) {
		jobId = rand.Text()[:16]
	}

	job.JobId = jobId
	job.Status = "queued"
	job.Result = ""

	jobs[jobId] = job
	mux.Unlock()

	select {
	case jobQueue <- jobId:
		// job added, no action
	default:
		mux.Lock()
		delete(jobs, jobId)
		mux.Unlock()

		log.Printf("job %s not added to queue: channel full?", jobId)
		http.Error(w, "job queue full", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}

func DeleteJob(w http.ResponseWriter, r *http.Request) {
	job_id := r.PathValue("id")

	mux.RLock()
	if _, ok := jobs[job_id]; !ok {
		http.Error(w, "job does not exist", http.StatusNotFound)
		mux.RUnlock()
		return
	}
	mux.RUnlock()

	mux.Lock()
	delete(jobs, job_id)
	mux.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

func DoJob() {
	for jobId := range jobQueue {
		jobCycle(jobId)
	}
}

func jobCycle(job_id string) bool {
	mux.Lock()
	job, ok := jobs[job_id]
	if !ok {
		mux.Unlock()
		log.Printf("job %s does not exist!", job_id)
		return false
	}
	job.Status = "in_progress"
	jobs[job_id] = job
	mux.Unlock()

	switch {
	case job.Type == "echo":
		job.echo()
	// case job.Type == "cmd":
	// 	cmd := exec.Command("sh", "-c", job.Payload)
	// 	output, err := cmd.CombinedOutput()
	// 	if err != nil {
	// 		job.Result = fmt.Sprintf("%s\n", err)
	// 		job.Status = "failed"
	// 		return
	// 	}
	// 	job.Result = string(output)
	// 	job.Status = "completed"
	case job.Type == "eval":
		job.evaluate()
	default:
		job.Result = "unknown job type"
		job.Status = "failed"
		fmt.Printf("%v\n", job)
	}

	mux.Lock()
	jobs[job_id] = job
	mux.Unlock()

	return true
}

// Stuff regarding the function above; Commands

func (job *Job) echo() {
	job.Result = job.Payload
	job.Status = "completed"
}

func (job *Job) evaluate() {
	code, err := expr.Compile(job.Payload, expr.AsBool())
	if err != nil {
		log.Println(err.Error())
		job.Result = "invalid expression"
		job.Status = "failed"
		return
	}

	result, err := expr.Run(code, nil)
	if err != nil {
		log.Println(err.Error())
		job.Result = "invalid expression"
		job.Status = "failed"
		return
	}

	if _, ok := result.(bool); !ok {
		log.Println(result)
		job.Result = "invalid expression"
		job.Status = "failed"
		return
	}

	job.Result = fmt.Sprint(result)
	job.Status = "completed"
}

func BeginShutdown() {
	submit_mux.Lock()
	shuttingDown = true
	submit_mux.Unlock()
}

func CloseJobQueue() {
	close(jobQueue)
}
