// internal/kb/models.go
package kb

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// KBItem represents a knowledge base item
type KBItem struct {
	ID          string    `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Content     string    `json:"content" db:"content"`
	TLDR        string    `json:"tldr" db:"tldr"`
	OriginalURL string    `json:"original_url" db:"original_url"`
	ImpactScore float64   `json:"impact_score" db:"impact_score"`
	Tags        []string  `json:"tags"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Tag represents a knowledge base tag
type Tag struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// ItemTag represents the many-to-many relationship between KBItem and Tag
type ItemTag struct {
	ItemID string `json:"item_id" db:"item_id"`
	TagID  int    `json:"tag_id" db:"tag_id"`
}

// StringSlice is a custom type for storing string slices in SQLite
type StringSlice []string

// Value implements the driver.Valuer interface
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}

	switch v := value.(type) {
	case string:
		return json.Unmarshal([]byte(v), s)
	case []byte:
		return json.Unmarshal(v, s)
	default:
		return fmt.Errorf("cannot scan type %T into StringSlice", value)
	}
}

// IngestRequest represents the request body for knowledge ingestion
type IngestRequest struct {
	Content string `json:"content"`
}

// IngestResponse represents the response for knowledge ingestion
type IngestResponse struct {
	Status      string  `json:"status"`
	ItemID      string  `json:"item_id,omitempty"`
	Message     string  `json:"message"`
	ImpactScore float64 `json:"impact_score,omitempty"`
}

// TagFilter represents a filter for knowledge items by tags
type TagFilter struct {
	Tags   []string `json:"tags"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

// PaginatedKBItems represents a paginated list of knowledge items
type PaginatedKBItems struct {
	Items      []KBItem `json:"items"`
	Total      int      `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"per_page"`
	TotalPages int      `json:"total_pages"`
}

// TagCloudItem represents a tag with usage count for the tag cloud
type TagCloudItem struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Validate validates the KBItem fields
func (item *KBItem) Validate() error {
	if item.ID == "" {
		return fmt.Errorf("id is required")
	}
	if item.Title == "" {
		return fmt.Errorf("title is required")
	}
	if item.Content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

// IsURL checks if the content is a URL
func (req *IngestRequest) IsURL() bool {
	content := strings.TrimSpace(req.Content)
	return len(content) >= 4 && strings.HasPrefix(strings.ToLower(content), "http")
}

// GetContentType returns the type of content (url or raw_text)
func (req *IngestRequest) GetContentType() string {
	if req.IsURL() {
		return "url"
	}
	return "raw_text"
}
