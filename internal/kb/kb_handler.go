// internal/kb/kb_handler.go
package kb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// maxContentSize is the maximum allowed content size (10MB)
const maxContentSize = 10 * 1024 * 1024

// jobTTL is the time-to-live for ingestion jobs (24 hours)
const jobTTL = 24 * time.Hour

// cleanupInterval is how often to run cleanup (every hour)
const cleanupInterval = time.Hour

// ingestJob represents an async ingestion job
type ingestJob struct {
	ID        string
	Status    string
	Message   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// jobStore stores ingestion job statuses
var (
	jobStore       = make(map[string]*ingestJob)
	jobMux         sync.RWMutex
	cleanupStarted sync.Once
)

// getJob retrieves a job by ID, returns nil if expired
// Uses RLock for reads, only upgrades to Lock when deletion is needed
func getJob(id string) (*ingestJob, bool) {
	// Phase 1: Read with RLock
	jobMux.RLock()
	job, exists := jobStore[id]
	if !exists {
		jobMux.RUnlock()
		return nil, false
	}

	// Fast path: job not expired, return immediately
	if time.Since(job.UpdatedAt) <= jobTTL {
		jobMux.RUnlock()
		return job, true
	}
	jobMux.RUnlock()

	// Phase 2: Job expired, need to delete - upgrade to Lock
	jobMux.Lock()
	defer jobMux.Unlock()

	// Double-check after acquiring write lock (another goroutine might have deleted it)
	if job, exists := jobStore[id]; exists {
		if time.Since(job.UpdatedAt) > jobTTL {
			delete(jobStore, id)
			log.Printf("[KB] Expired job %s cleaned up during get", id)
		} else {
			// Job was renewed during lock upgrade
			return job, true
		}
	}
	return nil, false
}

// cleanupExpiredJobs removes expired jobs from the store
func cleanupExpiredJobs() {
	jobMux.Lock()
	defer jobMux.Unlock()
	now := time.Now()
	expiredCount := 0
	for id, job := range jobStore {
		if now.Sub(job.UpdatedAt) > jobTTL {
			delete(jobStore, id)
			expiredCount++
		}
	}
	if expiredCount > 0 {
		log.Printf("[KB] Cleaned up %d expired ingestion jobs", expiredCount)
	}
}

// startJobCleanupRoutine starts a background goroutine to periodically clean up expired jobs
func startJobCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredJobs()
		}
	}()
}

// updateJob updates a job's status
func updateJob(id, status, message string) {
	jobMux.Lock()
	defer jobMux.Unlock()
	if job, exists := jobStore[id]; exists {
		job.Status = status
		job.Message = message
		job.UpdatedAt = time.Now()
	}
}

// createJob creates a new ingestion job
func createJob(id string) *ingestJob {
	// Ensure cleanup routine is started (only once)
	cleanupStarted.Do(startJobCleanupRoutine)

	jobMux.Lock()
	defer jobMux.Unlock()
	job := &ingestJob{
		ID:        id,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	jobStore[id] = job
	return job
}

// Handler handles knowledge base HTTP requests
type Handler struct {
	db *DB
}

// NewHandler creates a new knowledge base handler
func NewHandler(dbPath string) (*Handler, error) {
	db, err := NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &Handler{db: db}, nil
}

// Close closes the handler's database connection
func (h *Handler) Close() error {
	return h.db.Close()
}

// RegisterRoutes registers all KB routes with the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Protected routes
	mux.HandleFunc("/api/kb/ingest", authMiddleware(h.HandleIngest))
	mux.HandleFunc("/api/kb/items", authMiddleware(h.HandleListItems))
	mux.HandleFunc("/api/kb/recent", authMiddleware(h.HandleRecentItems))
	mux.HandleFunc("/api/kb/list", authMiddleware(h.HandleListByTag))
	mux.HandleFunc("/api/kb/tags", authMiddleware(h.HandleListTags))
	mux.HandleFunc("/api/kb/status", authMiddleware(h.HandleGetStatus))
	mux.HandleFunc("/api/kb/search", authMiddleware(h.HandleSearch))
}

// HandleIngest handles the POST /api/kb/ingest endpoint
func (h *Handler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Content) == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// Check content size limit
	if len(req.Content) > maxContentSize {
		http.Error(w, fmt.Sprintf("Content exceeds maximum size of %d bytes", maxContentSize), http.StatusBadRequest)
		return
	}

	// Determine content type
	contentType := req.GetContentType()

	// Generate UUID for this ingestion
	itemID := uuid.New().String()

	// Create job entry for status tracking
	createJob(itemID)

	// Invoke sentinel.py for processing
	// This runs asynchronously and will populate the database
	go h.processIngestAsync(itemID, req.Content, contentType)

	// Return immediate response
	resp := IngestResponse{
		Status:  "processing",
		ItemID:  itemID,
		Message: fmt.Sprintf("Knowledge ingestion started. Content type: %s", contentType),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// processIngestAsync invokes sentinel.py to process the knowledge ingestion
func (h *Handler) processIngestAsync(itemID, content, contentType string) {
	// Create a temporary file for content instead of using environment variables
	tmpDir := os.TempDir()
	contentFile := filepath.Join(tmpDir, fmt.Sprintf("kb_ingest_%s.txt", itemID))

	// Write content to temporary file
	if err := ioutil.WriteFile(contentFile, []byte(content), 0600); err != nil {
		log.Printf("[KB] Failed to write content file for item %s: %v", itemID, err)
		updateJob(itemID, "failed", fmt.Sprintf("Failed to write content: %v", err))
		return
	}
	// Clean up temp file after processing
	defer os.Remove(contentFile)

	// Build command to invoke sentinel.py with file path
	cmd := exec.Command("python3", "sentinel.py", "--kb-ingest", itemID, contentType, contentFile)

	// Set minimal environment variables (no content)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KB_ITEM_ID=%s", itemID),
		fmt.Sprintf("KB_CONTENT_TYPE=%s", contentType),
		fmt.Sprintf("KB_CONTENT_FILE=%s", contentFile),
	)

	updateJob(itemID, "processing", "Sentinel is analyzing the content")

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[KB] Failed to process ingest for item %s: %v\nOutput: %s", itemID, err, string(output))
		updateJob(itemID, "failed", fmt.Sprintf("Processing failed: %v", err))
		return
	}

	updateJob(itemID, "completed", "Knowledge item processed successfully")
	log.Printf("[KB] Successfully processed ingest for item %s", itemID)
}

// HandleListItems handles GET /api/kb/items endpoint
func (h *Handler) HandleListItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters with validation
	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > 100 {
				limit = 100 // Maximum limit to prevent DoS
			} else {
				limit = parsed
			}
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Parse tags filter
	var tags []string
	if t := r.URL.Query().Get("tags"); t != "" {
		tags = strings.Split(t, ",")
	}

	// Retrieve items
	items, err := h.db.ListKBItems(tags, limit, offset)
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve KB items: %v", err)
		http.Error(w, "Failed to retrieve items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// HandleRecentItems handles GET /api/kb/recent endpoint - returns last 20 items
func (h *Handler) HandleRecentItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Retrieve recent 20 items without tag filter
	items, err := h.db.ListKBItems(nil, 20, 0)
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve recent KB items: %v", err)
		http.Error(w, "Failed to retrieve items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// HandleListByTag handles GET /api/kb/list?tag=xxx endpoint
func (h *Handler) HandleListByTag(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse and validate tag parameter
	tag := r.URL.Query().Get("tag")
	if tag == "" || len(tag) > 50 {
		http.Error(w, "Invalid tag parameter", http.StatusBadRequest)
		return
	}
	// Validate tag format: allow Unicode letters, numbers, and common separators
	// This supports tags like "open-source", "ai-agent", "人工智能", "VIX"
	validTagPattern := regexp.MustCompile(`^[\p{L}\p{N}_\-\s\.]+$`)
	if !validTagPattern.MatchString(tag) {
		http.Error(w, "Invalid tag format", http.StatusBadRequest)
		return
	}

	// Parse limit parameter with validation
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > 100 {
				limit = 100 // Maximum limit to prevent DoS
			} else {
				limit = parsed
			}
		}
	}

	// Retrieve items with single tag filter
	items, err := h.db.ListKBItems([]string{tag}, limit, 0)
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve KB items by tag: %v", err)
		http.Error(w, "Failed to retrieve items", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

// HandleListTags handles GET /api/kb/tags endpoint
func (h *Handler) HandleListTags(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tags, err := h.db.GetAllTags()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to retrieve tags: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

// HandleGetStatus handles GET /api/kb/status endpoint
func (h *Handler) HandleGetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemID := r.URL.Query().Get("id")
	if itemID == "" {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	job, exists := getJob(itemID)
	if !exists {
		// Check if item exists in database
		_, err := h.db.GetKBItem(itemID)
		if err != nil {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}
		// Item exists in DB, job completed
		job = &ingestJob{
			ID:     itemID,
			Status: "completed",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// HandleSearch handles GET /api/kb/search?q=keyword endpoint
func (h *Handler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keyword := r.URL.Query().Get("q")
	if keyword == "" {
		http.Error(w, "Missing search query", http.StatusBadRequest)
		return
	}

	// Parse pagination params with explicit error handling
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		} else if err != nil {
			log.Printf("[WARN] Invalid limit parameter '%s': %v", l, err)
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		} else if err != nil {
			log.Printf("[WARN] Invalid offset parameter '%s': %v", o, err)
		}
	}

	results, err := h.db.SearchKBItems(keyword, limit, offset)
	if err != nil {
		log.Printf("[KB] Search failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// LogIngestion logs a knowledge ingestion event
func LogIngestion(title string, impactScore float64) {
	log.Printf("[KB] New Knowledge Ingested: \"%s\", Impact: %.2f", title, impactScore)
}
