package task_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
	"opspilot/task-api/internal/platform/response"
	"opspilot/task-api/internal/task"
)

func setupTestServer() (http.Handler, *task.MemoryRepository) {
	repo := task.NewMemoryRepository()
	router := task.NewRouter(task.NewService(repo))
	return router, repo
}

func TestHealthCheck(t *testing.T) {
	router, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}

func TestTaskEndpoints_CRUD(t *testing.T) {
	router, _ := setupTestServer()

	// 1. Initial GET /tasks should return empty array []
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET /tasks expected 200 OK, got %d", rec.Code)
		}
		var tasks []task.Task
		if err := json.NewDecoder(rec.Body).Decode(&tasks); err != nil {
			t.Fatalf("failed to decode tasks: %v", err)
		}
		if len(tasks) != 0 {
			t.Fatalf("expected empty array, got %d items", len(tasks))
		}
	}

	// 2. POST /tasks create first task
	var createdTask1 task.Task
	{
		body := `{"title":"Task 1","description":"Initial task"}`
		req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /tasks expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		loc := rec.Header().Get("Location")
		if loc == "" || !strings.HasPrefix(loc, "/tasks/") {
			t.Errorf("expected Location header starting with /tasks/, got %q", loc)
		}

		if err := json.NewDecoder(rec.Body).Decode(&createdTask1); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if createdTask1.ID == "" {
			t.Error("expected non-empty task ID")
		}
		if !task.IsValidID(createdTask1.ID) {
			t.Errorf("expected valid UUID, got %q", createdTask1.ID)
		}
		if createdTask1.Title != "Task 1" {
			t.Errorf("expected title 'Task 1', got %q", createdTask1.Title)
		}
		if createdTask1.Status != task.StatusPending {
			t.Errorf("expected default status 'pending', got %q", createdTask1.Status)
		}
	}

	// 3. POST /tasks create second task with custom status
	var createdTask2 task.Task
	{
		body := `{"title":"Task 2","status":"in_progress"}`
		req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /tasks expected 201 Created, got %d", rec.Code)
		}

		if err := json.NewDecoder(rec.Body).Decode(&createdTask2); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if createdTask2.Status != task.StatusInProgress {
			t.Errorf("expected status 'in_progress', got %q", createdTask2.Status)
		}
	}

	// 4. GET /tasks list all tasks (should contain 2 items)
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET /tasks expected 200 OK, got %d", rec.Code)
		}

		var tasks []task.Task
		if err := json.NewDecoder(rec.Body).Decode(&tasks); err != nil {
			t.Fatalf("failed to decode tasks: %v", err)
		}

		if len(tasks) != 2 {
			t.Fatalf("expected 2 tasks, got %d", len(tasks))
		}
	}

	// 5. GET /tasks/{id} fetch task 1
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+createdTask1.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("GET /tasks/{id} expected 200 OK, got %d", rec.Code)
		}

		var fetched task.Task
		if err := json.NewDecoder(rec.Body).Decode(&fetched); err != nil {
			t.Fatalf("failed to decode task: %v", err)
		}

		if fetched.ID != createdTask1.ID || fetched.Title != createdTask1.Title {
			t.Errorf("fetched task %+v mismatch with created %+v", fetched, createdTask1)
		}
	}

	// 6. PUT /tasks/{id} update task 1
	{
		body := `{"title":"Task 1 Updated","description":"Updated desc","status":"completed"}`
		req := httptest.NewRequest(http.MethodPut, "/tasks/"+createdTask1.ID, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("PUT /tasks/{id} expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var updated task.Task
		if err := json.NewDecoder(rec.Body).Decode(&updated); err != nil {
			t.Fatalf("failed to decode updated task: %v", err)
		}

		if updated.Title != "Task 1 Updated" || updated.Status != task.StatusCompleted {
			t.Errorf("unexpected updated task: %+v", updated)
		}
	}

	// 7. DELETE /tasks/{id} delete task 1
	{
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+createdTask1.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("DELETE /tasks/{id} expected 204 No Content, got %d", rec.Code)
		}
	}

	// 8. GET /tasks/{id} after deletion should return 404 Not Found
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+createdTask1.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("GET deleted /tasks/{id} expected 404 Not Found, got %d", rec.Code)
		}
	}

	// 9. DELETE /tasks/{id} again should return 404 Not Found
	{
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+createdTask1.ID, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("DELETE non-existent /tasks/{id} expected 404 Not Found, got %d", rec.Code)
		}
	}
}

func TestTaskEndpoints_ValidationAndErrorCases(t *testing.T) {
	router, _ := setupTestServer()

	tests := []struct {
		name       string
		method     string
		url        string
		body       string
		wantStatus int
	}{
		{
			name:       "POST malformed json syntax",
			method:     http.MethodPost,
			url:        "/tasks",
			body:       `{"title": "incomplete json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "POST empty request body",
			method:     http.MethodPost,
			url:        "/tasks",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "POST unknown field rejected",
			method:     http.MethodPost,
			url:        "/tasks",
			body:       `{"title":"Valid Title","unknown_property":"bad"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "POST trailing data rejected",
			method:     http.MethodPost,
			url:        "/tasks",
			body:       `{"title":"Valid Title"}{"title":"Second"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "POST missing title rejected",
			method:     http.MethodPost,
			url:        "/tasks",
			body:       `{"description":"No title provided"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "POST invalid status rejected",
			method:     http.MethodPost,
			url:        "/tasks",
			body:       `{"title":"Task","status":"invalid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "GET non-existent task with valid UUID returns 404",
			method:     http.MethodGet,
			url:        "/tasks/00000000-0000-0000-0000-000000000000",
			body:       "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET invalid UUID format returns 400",
			method:     http.MethodGet,
			url:        "/tasks/invalid-uuid-string",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "PUT non-existent task with valid UUID returns 404",
			method:     http.MethodPut,
			url:        "/tasks/00000000-0000-0000-0000-000000000000",
			body:       `{"title":"Title","status":"completed"}`,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "PUT invalid UUID format returns 400",
			method:     http.MethodPut,
			url:        "/tasks/invalid-uuid-string",
			body:       `{"title":"Title","status":"completed"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "PUT malformed json returns 400",
			method:     http.MethodPut,
			url:        "/tasks/00000000-0000-0000-0000-000000000000",
			body:       `{"title": bad json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "PUT validation failure returns 400",
			method:     http.MethodPut,
			url:        "/tasks/00000000-0000-0000-0000-000000000000",
			body:       `{"title":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "DELETE invalid UUID format returns 400",
			method:     http.MethodDelete,
			url:        "/tasks/not-a-valid-uuid",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "DELETE non-existent task with valid UUID returns 404",
			method:     http.MethodDelete,
			url:        "/tasks/00000000-0000-0000-0000-000000000000",
			body:       "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "Method not allowed returns 405",
			method:     http.MethodPatch,
			url:        "/tasks",
			body:       `{}`,
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader *bytes.Reader
			if tc.body != "" {
				bodyReader = bytes.NewReader([]byte(tc.body))
			} else {
				bodyReader = bytes.NewReader(nil)
			}

			req := httptest.NewRequest(tc.method, tc.url, bodyReader)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status code = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}

			// Ensure error response has proper JSON structure when status >= 400 and != 405
			if rec.Code >= 400 && rec.Code != http.StatusMethodNotAllowed {
				var errResp response.ErrorResponse
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("expected JSON error response: %v", err)
				}
				if errResp.Error == "" {
					t.Errorf("expected non-empty error message, got: %+v", errResp)
				}
			}
		})
	}
}

func TestTaskEndpoints_PayloadTooLarge(t *testing.T) {
	router, _ := setupTestServer()

	// Create payload > 1 MB (1048576 bytes)
	largeDescription := strings.Repeat("A", 1048576+100)
	body := `{"title":"Large Task","description":"` + largeDescription + `"}`

	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for oversized payload, got %d", rec.Code)
	}
}

func TestTaskEndpoints_ConcurrentRequests(t *testing.T) {
	router, _ := setupTestServer()
	const workers = 20

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(idx int) {
			defer wg.Done()

			// Concurrently create
			body := `{"title":"Concurrent Task"}`
			req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Errorf("concurrent POST failed: %d", rec.Code)
				return
			}

			var created task.Task
			if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
				t.Errorf("concurrent decode failed: %v", err)
				return
			}

			// Concurrently get
			getReq := httptest.NewRequest(http.MethodGet, "/tasks/"+created.ID, nil)
			getRec := httptest.NewRecorder()
			router.ServeHTTP(getRec, getReq)
			if getRec.Code != http.StatusOK {
				t.Errorf("concurrent GET failed: %d", getRec.Code)
				return
			}

			// Concurrently list
			listReq := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			listRec := httptest.NewRecorder()
			router.ServeHTTP(listRec, listReq)
			if listRec.Code != http.StatusOK {
				t.Errorf("concurrent LIST failed: %d", listRec.Code)
				return
			}
		}(i)
	}

	wg.Wait()
}

func TestSecuredTaskEndpoints(t *testing.T) {
	repo := task.NewMemoryRepository()
	svc := task.NewService(repo)
	tokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", 1*time.Hour)
	revStore := auth.NewMemoryRevocationStore()

	router := task.NewSecuredRouter(svc, tokenSvc, revStore, nil)

	// Helper to generate tokens
	adminT1, _, _ := tokenSvc.GenerateToken(auth.User{ID: "admin-1", TenantID: "tenant-1", Role: auth.RoleAdmin})
	memberT1, _, _ := tokenSvc.GenerateToken(auth.User{ID: "member-1", TenantID: "tenant-1", Role: auth.RoleMember})
	viewerT1, _, _ := tokenSvc.GenerateToken(auth.User{ID: "viewer-1", TenantID: "tenant-1", Role: auth.RoleViewer})
	adminT2, _, _ := tokenSvc.GenerateToken(auth.User{ID: "admin-2", TenantID: "tenant-2", Role: auth.RoleAdmin})

	// 1. Health check is public (unauthenticated)
	{
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("health check expected 200, got %d", rec.Code)
		}
	}

	// 2. Missing authorization header on /tasks -> 401 Unauthorized
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for missing auth, got %d", rec.Code)
		}
	}

	// 3. Invalid/malformed Bearer token -> 401 Unauthorized
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.structure")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for malformed token, got %d", rec.Code)
		}
	}

	// 4. Expired token -> 401 Unauthorized
	{
		expiredTokenSvc, _ := auth.NewTokenService("test-secret-at-least-32-bytes-long", "opspilot", "opspilot-api", -1*time.Second)
		expToken, _, _ := expiredTokenSvc.GenerateToken(auth.User{ID: "exp", TenantID: "tenant-1", Role: auth.RoleAdmin})

		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+expToken)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for expired token, got %d", rec.Code)
		}
	}

	// 5. Viewer role cannot create task -> 403 Forbidden
	{
		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"Viewer Task"}`))
		req.Header.Set("Authorization", "Bearer "+viewerT1)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when Viewer tries to POST /tasks, got %d", rec.Code)
		}
	}

	// 6. Admin in Tenant 1 creates a task -> 201 Created
	var createdID string
	{
		req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"Tenant 1 Secret Task"}`))
		req.Header.Set("Authorization", "Bearer "+adminT1)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for Admin, got %d, body: %s", rec.Code, rec.Body.String())
		}
		var tObj task.Task
		_ = json.Unmarshal(rec.Body.Bytes(), &tObj)
		createdID = tObj.ID
	}

	// 7. Viewer in Tenant 1 can read the task -> 200 OK
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+createdID, nil)
		req.Header.Set("Authorization", "Bearer "+viewerT1)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for Viewer GET /tasks/{id}, got %d", rec.Code)
		}
	}

	// 8. Member in Tenant 1 can update the task -> 200 OK
	{
		req := httptest.NewRequest(http.MethodPut, "/tasks/"+createdID, bytes.NewBufferString(`{"title":"Updated Title","status":"in_progress"}`))
		req.Header.Set("Authorization", "Bearer "+memberT1)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for Member PUT /tasks/{id}, got %d", rec.Code)
		}
	}

	// 9. Member in Tenant 1 CANNOT delete the task -> 403 Forbidden
	{
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+createdID, nil)
		req.Header.Set("Authorization", "Bearer "+memberT1)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden when Member tries to DELETE /tasks/{id}, got %d", rec.Code)
		}
	}

	// 10. Cross-Tenant Access Rejection: Admin in Tenant 2 CANNOT read Tenant 1's task by ID -> 404 Not Found
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks/"+createdID, nil)
		req.Header.Set("Authorization", "Bearer "+adminT2)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for cross-tenant GET /tasks/{id}, got %d", rec.Code)
		}
	}

	// 11. Cross-Tenant Access Rejection: Admin in Tenant 2 CANNOT update Tenant 1's task by ID -> 404 Not Found
	{
		req := httptest.NewRequest(http.MethodPut, "/tasks/"+createdID, bytes.NewBufferString(`{"title":"Hack Title","status":"completed"}`))
		req.Header.Set("Authorization", "Bearer "+adminT2)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for cross-tenant PUT /tasks/{id}, got %d", rec.Code)
		}
	}

	// 12. Cross-Tenant Access Rejection: Admin in Tenant 2 CANNOT delete Tenant 1's task by ID -> 404 Not Found
	{
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+createdID, nil)
		req.Header.Set("Authorization", "Bearer "+adminT2)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected 404 Not Found for cross-tenant DELETE /tasks/{id}, got %d", rec.Code)
		}
	}

	// 13. Cross-Tenant Isolation: Admin in Tenant 2 listing /tasks sees empty list
	{
		req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
		req.Header.Set("Authorization", "Bearer "+adminT2)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK for list tasks, got %d", rec.Code)
		}
		var tasks []task.Task
		_ = json.Unmarshal(rec.Body.Bytes(), &tasks)
		if len(tasks) != 0 {
			t.Errorf("tenant 2 should see 0 tasks, got %d", len(tasks))
		}
	}

	// 14. Admin in Tenant 1 can successfully delete the task -> 204 No Content
	{
		req := httptest.NewRequest(http.MethodDelete, "/tasks/"+createdID, nil)
		req.Header.Set("Authorization", "Bearer "+adminT1)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204 No Content for Admin DELETE /tasks/{id}, got %d", rec.Code)
		}
	}
}
