package scheduler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makePreconditions wraps a single precondition in the slice required by
// EvaluatePreconditions.
func makePreconditions(p Precondition) []Precondition { return []Precondition{p} }

// TestAfterJobDependencySucceeded verifies that after_job returns true when the
// named job's last run has status success.
func TestAfterJobDependencySucceeded(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("dep-job")); err != nil {
		t.Fatal(err)
	}
	r := &Run{JobName: "dep-job", StartTime: time.Now(), Status: RunStatusSuccess}
	if err := s.InsertRun(r); err != nil {
		t.Fatal(err)
	}

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionAfterJob, JobName: "dep-job"}),
		s, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true (dependency succeeded), got false")
	}
}

// TestAfterJobDependencyFailed verifies that after_job returns false when the
// last run of the named job has status failure.
func TestAfterJobDependencyFailed(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("fail-dep")); err != nil {
		t.Fatal(err)
	}
	r := &Run{JobName: "fail-dep", StartTime: time.Now(), Status: RunStatusFailure}
	if err := s.InsertRun(r); err != nil {
		t.Fatal(err)
	}

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionAfterJob, JobName: "fail-dep"}),
		s, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false (dependency failed), got true")
	}
}

// TestAfterJobDependencyNeverRan verifies that after_job returns false when the
// named job has no runs yet.
func TestAfterJobDependencyNeverRan(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("no-run-dep")); err != nil {
		t.Fatal(err)
	}

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionAfterJob, JobName: "no-run-dep"}),
		s, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false (job never ran), got true")
	}
}

// TestAfterJobDependencyDoesNotExist verifies that after_job returns false (not
// an error) when the named job does not exist in the store.
func TestAfterJobDependencyDoesNotExist(t *testing.T) {
	s := NewStore(newTestDB(t))

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionAfterJob, JobName: "ghost-job"}),
		s, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for non-existent dependency, got true")
	}
}

// TestFileExistsPresent verifies that file_exists returns true when the file is
// present inside the project sandbox.
func TestFileExistsPresent(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "myfile.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionFileExists, Path: "myfile.txt"}),
		nil, root)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true (file present), got false")
	}
}

// TestFileExistsAbsent verifies that file_exists returns false when the file does
// not exist.
func TestFileExistsAbsent(t *testing.T) {
	root := t.TempDir()

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionFileExists, Path: "missing.txt"}),
		nil, root)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false (file absent), got true")
	}
}

// TestFileExistsPathTraversal verifies that a path traversal attempt is rejected
// with an error rather than silently resolving outside the project root.
func TestFileExistsPathTraversal(t *testing.T) {
	root := t.TempDir()

	_, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionFileExists, Path: "../../etc/passwd"}),
		nil, root)
	if err == nil {
		t.Error("expected sandbox violation error for path traversal, got nil")
	}
}

// TestHTTPOk200 verifies that http_ok returns true when the endpoint responds 200.
func TestHTTPOk200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionHTTPOk, URL: srv.URL}),
		nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true for 200 response, got false")
	}
}

// TestHTTPOk500 verifies that http_ok returns false when the endpoint responds 500.
func TestHTTPOk500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionHTTPOk, URL: srv.URL}),
		nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for 500 response, got true")
	}
}

// TestHTTPOkUnreachable verifies that http_ok returns false (not a fatal error)
// when the host is unreachable, and does not hang indefinitely.
func TestHTTPOkUnreachable(t *testing.T) {
	// Use a context with a generous deadline to catch hangs.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Point at a port on localhost that is not listening.
	ok, err := EvaluatePreconditions(ctx,
		makePreconditions(Precondition{Kind: PreconditionHTTPOk, URL: "http://127.0.0.1:19999"}),
		nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for unreachable host, got true")
	}
}

// TestShellExitZero verifies that a shell command exiting 0 returns true.
func TestShellExitZero(t *testing.T) {
	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionShell, Command: "true"}),
		nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected true for exit 0, got false")
	}
}

// TestShellExitNonZero verifies that a shell command exiting non-zero returns false.
func TestShellExitNonZero(t *testing.T) {
	ok, err := EvaluatePreconditions(context.Background(),
		makePreconditions(Precondition{Kind: PreconditionShell, Command: "false"}),
		nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for exit 1, got true")
	}
}

// TestShellTimeout verifies that a long-running shell command is terminated by
// the precondition's internal timeout and returns false without hanging.
func TestShellTimeout(t *testing.T) {
	// evalShell uses a 30s internal timeout; we use a short command timeout via
	// a context that expires in 2 seconds so the test finishes promptly.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use a command that sleeps longer than the context timeout.
	ok, err := EvaluatePreconditions(ctx,
		makePreconditions(Precondition{Kind: PreconditionShell, Command: "sleep 60"}),
		nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected false for timed-out shell command, got true")
	}
}
