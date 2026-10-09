package worker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

/*
PostJob test
POST /jobs -> CODE
Test for potential malformed JSON or invalid codes
*/

var successJob = "test-job-success"
var failJob = "test-job-fail"
var deleteJob = "test-job-delete"

func TestPostJob_Success(t *testing.T) {
	body := `{"type":"echo","payload":"testing"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/jobs",
		strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	PostJob(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d",
			http.StatusCreated, rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected content type %s, got %s",
			"application/json", rec.Header().Get("Content-Type"))
	}

	job := Job{}
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
		t.Fatalf("expected to decode job, json: %s", rec.Body.String())
	}

	switch {
	case job.JobId == "":
		t.Fatalf("expected job_id to be set, got %s", job.JobId)
	case job.Result != "nil":
		t.Fatalf("expected result to be nil, got %s", job.Result)
	case job.Status != "queued":
		t.Fatalf("expected status to be queued, got %s", job.Status)
	case job.Type != "echo":
		t.Fatalf("expected type to be echo, got %s", job.Type)
	case job.Payload != "testing":
		t.Fatalf("expected payload to be testing, got %s", job.Payload)
	}

	mux.RLock()
	_, ok := jobs[job.JobId]
	mux.RUnlock()

	if !ok {
		t.Fatalf("expected job to be in jobs map, got nil")
	}

	t.Cleanup(func() {
		mux.Lock()
		defer mux.Unlock()
		delete(jobs, job.JobId)
	})
}

func TestPostJob_InvalidJSON(t *testing.T) {
	body := `{type:"echo", payload:"testing"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/jobs",
		strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	PostJob(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestPostJob_UnsupportedType(t *testing.T) {
	body := `{"type":"unsupported","payload":"testing"}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/jobs",
		strings.NewReader(body),
	)
	rec := httptest.NewRecorder()

	PostJob(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

/*
GetJob test
GET /jobs/:id -> CODE
Test for potential malformed JSON or invalid codes
*/

func TestGetJob_Success(t *testing.T) {
	mux.Lock()
	jobs[successJob] = Job{
		JobId:   successJob,
		Type:    "echo",
		Payload: "success job test",
		Status:  "completed",
		Result:  "success job test",
	}
	mux.Unlock()

	req := httptest.NewRequest(
		http.MethodGet,
		"/jobs/"+successJob,
		nil,
	)
	req.SetPathValue("id", successJob)
	rec := httptest.NewRecorder()

	GetJob(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected content type %s, got %s",
			"application/json", rec.Header().Get("Content-Type"))
	}

	job := Job{}
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
		t.Fatalf("expected to decode job, json: %s", rec.Body.String())
	}

	if job.JobId != successJob {
		t.Fatalf("expected job id %s, got %s", successJob, job.JobId)
	}

	mux.RLock()
	v, ok := jobs[job.JobId]
	mux.RUnlock()

	if !ok {
		t.Fatalf("expected job to be in jobs map, got nil")
	} else if v != job {
		t.Fatalf("expected job %v, got %v", job, v)
	}

	t.Cleanup(func() {
		mux.Lock()
		defer mux.Unlock()
		delete(jobs, successJob)
	})
}

func TestGetJob_NotFound(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/jobs/"+failJob,
		nil,
	)
	req.SetPathValue("id", failJob)
	rec := httptest.NewRecorder()

	GetJob(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestDeleteJob_Success(t *testing.T) {
	mux.Lock()
	jobs[deleteJob] = Job{
		JobId:   deleteJob,
		Type:    "echo",
		Payload: "delete job test",
		Status:  "failed",
		Result:  "hopefully gonna delete this",
	}
	mux.Unlock()

	t.Cleanup(func() {
		mux.Lock()
		defer mux.Unlock()
		delete(jobs, deleteJob)
	})

	req := httptest.NewRequest(
		http.MethodDelete,
		"/jobs/"+deleteJob,
		nil,
	)
	req.SetPathValue("id", deleteJob)
	rec := httptest.NewRecorder()

	DeleteJob(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status code %d, got %d", http.StatusNoContent, rec.Code)
	}

	mux.RLock()
	_, ok := jobs[deleteJob]
	mux.RUnlock()

	if ok {
		t.Fatalf("job %s should have been deleted", deleteJob)
	}

}

func TestDeleteJob_NotFound(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodDelete,
		"/jobs/"+deleteJob,
		nil,
	)
	req.SetPathValue("id", deleteJob)
	rec := httptest.NewRecorder()

	DeleteJob(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}
