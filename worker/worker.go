package worker

import (
	"crypto"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var availableJobs = []string{
	"echo",
}

var jobs = make(map[string]Job) // Temporary until i decide to use a database
var mux = sync.RWMutex{}

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

	hash := crypto.SHA256.New()
	hash.Write([]byte(strconv.Itoa(rand.Int())))
	sum := hash.Sum(nil)
	jobId := hex.EncodeToString(sum[:])[:16]

	job.JobId = jobId
	job.Status = "queued"
	job.Result = "nil"

	mux.Lock()
	jobs[jobId] = job
	mux.Unlock()

	w.Header().Set("Content-Type", "application/json")
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

	mux.RLock()
	if _, ok := jobs[job_id]; !ok {
		w.WriteHeader(http.StatusNoContent)
		mux.RUnlock()
		return
	}
	mux.RUnlock()
}

func DoJob() {
	for {
		var job Job
		var job_id string
		var found = false
		mux.Lock()
		for id_k, job_v := range jobs {
			if job_v.Status == "queued" {
				job_v.Status = "in_progress"
				jobs[id_k] = job_v
				job = job_v
				job_id = id_k
				found = true
				break
			}
		}
		mux.Unlock()

		if !found {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		switch {
		case job.Type == "echo":
			job.Result = job.Payload
			job.Status = "completed"
			fmt.Printf("%v\n", job)
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
		default:
			job.Result = "unknown job type"
			job.Status = "failed"
			fmt.Printf("%v\n", job)
		}

		mux.Lock()
		jobs[job_id] = job
		mux.Unlock()
	}
}
