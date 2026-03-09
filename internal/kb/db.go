// internal/kb/db.go
package kb

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the knowledge base database operations
type DB struct {
	conn *sql.DB
}

// NewDB creates a new knowledge base database instance
func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// NewDBFromConn creates a new knowledge base database instance from existing connection
// This is useful when you want to reuse an existing database connection
func NewDBFromConn(conn *sql.DB) *DB {
	return &DB{conn: conn}
}

// InitSchema creates the necessary tables for the knowledge base
func (db *DB) InitSchema() error {
	// Create kb_items table with string ID (UUID or Slug)
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS kb_items (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			tldr TEXT,
			original_url TEXT,
			impact_score REAL DEFAULT 0.0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create kb_items table: %w", err)
	}

	// Create index for kb_items
	_, err = db.conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_kb_items_created_at 
		ON kb_items(created_at DESC)
	`)
	if err != nil {
		return fmt.Errorf("failed to create kb_items index: %w", err)
	}

	_, err = db.conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_kb_items_impact_score 
		ON kb_items(impact_score DESC)
	`)
	if err != nil {
		return fmt.Errorf("failed to create impact_score index: %w", err)
	}

	// Create tags table with unique name constraint
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create tags table: %w", err)
	}

	// Create item_tags junction table for many-to-many relationship
	_, err = db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS item_tags (
			item_id TEXT NOT NULL,
			tag_id INTEGER NOT NULL,
			PRIMARY KEY (item_id, tag_id),
			FOREIGN KEY (item_id) REFERENCES kb_items(id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create item_tags table: %w", err)
	}

	// Create index for item_tags
	_, err = db.conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_item_tags_tag_id 
		ON item_tags(tag_id)
	`)
	if err != nil {
		return fmt.Errorf("failed to create item_tags index: %w", err)
	}

	return nil
}

// CreateKBItem creates a new knowledge base item
func (db *DB) CreateKBItem(item *KBItem) error {
	if err := item.Validate(); err != nil {
		return err
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	_, err := db.conn.Exec(
		`INSERT INTO kb_items (id, title, content, tldr, original_url, impact_score, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Title, item.Content, item.TLDR, item.OriginalURL, item.ImpactScore, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert kb_item: %w", err)
	}

	// Create or get tags and establish relationships
	if len(item.Tags) > 0 {
		if err := db.LinkTagsToItem(item.ID, item.Tags); err != nil {
			return fmt.Errorf("failed to link tags: %w", err)
		}
	}

	return nil
}

// GetKBItem retrieves a knowledge base item by ID
func (db *DB) GetKBItem(id string) (*KBItem, error) {
	var item KBItem
	var createdAt, updatedAt string

	err := db.conn.QueryRow(
		`SELECT id, title, content, tldr, original_url, impact_score, created_at, updated_at 
		FROM kb_items WHERE id = ?`,
		id,
	).Scan(&item.ID, &item.Title, &item.Content, &item.TLDR, &item.OriginalURL, &item.ImpactScore, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("kb_item not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query kb_item: %w", err)
	}

	item.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	item.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	// Load tags for this item
	tags, err := db.GetTagsByItemID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	item.Tags = tags

	return &item, nil
}

// UpdateKBItem updates an existing knowledge base item
func (db *DB) UpdateKBItem(item *KBItem) error {
	if err := item.Validate(); err != nil {
		return err
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	result, err := db.conn.Exec(
		`UPDATE kb_items SET title = ?, content = ?, tldr = ?, original_url = ?, impact_score = ?, updated_at = ? 
		WHERE id = ?`,
		item.Title, item.Content, item.TLDR, item.OriginalURL, item.ImpactScore, now, item.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update kb_item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("kb_item not found: %s", item.ID)
	}

	// Update tags
	if err := db.UnlinkTagsFromItem(item.ID); err != nil {
		return fmt.Errorf("failed to unlink old tags: %w", err)
	}

	if len(item.Tags) > 0 {
		if err := db.LinkTagsToItem(item.ID, item.Tags); err != nil {
			return fmt.Errorf("failed to link new tags: %w", err)
		}
	}

	return nil
}

// DeleteKBItem deletes a knowledge base item
func (db *DB) DeleteKBItem(id string) error {
	// Delete will cascade to item_tags due to foreign key constraint
	result, err := db.conn.Exec("DELETE FROM kb_items WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete kb_item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("kb_item not found: %s", id)
	}

	return nil
}

// CreateOrGetTag creates a new tag or returns existing one
func (db *DB) CreateOrGetTag(name string) (int, error) {
	// Normalize tag name
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return 0, fmt.Errorf("tag name cannot be empty")
	}

	// Try to insert, ignore if exists
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO tags (name) VALUES (?)`,
		name,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create tag: %w", err)
	}

	// Get the tag ID
	var tagID int
	err = db.conn.QueryRow(
		`SELECT id FROM tags WHERE name = ?`,
		name,
	).Scan(&tagID)
	if err != nil {
		return 0, fmt.Errorf("failed to get tag id: %w", err)
	}

	return tagID, nil
}

// LinkTagsToItem creates relationships between an item and its tags
func (db *DB) LinkTagsToItem(itemID string, tagNames []string) error {
	for _, name := range tagNames {
		tagID, err := db.CreateOrGetTag(name)
		if err != nil {
			return fmt.Errorf("failed to create/get tag '%s': %w", name, err)
		}

		_, err = db.conn.Exec(
			`INSERT OR IGNORE INTO item_tags (item_id, tag_id) VALUES (?, ?)`,
			itemID, tagID,
		)
		if err != nil {
			return fmt.Errorf("failed to link tag '%s' to item: %w", name, err)
		}
	}
	return nil
}

// UnlinkTagsFromItem removes all tag relationships for an item
func (db *DB) UnlinkTagsFromItem(itemID string) error {
	_, err := db.conn.Exec("DELETE FROM item_tags WHERE item_id = ?", itemID)
	if err != nil {
		return fmt.Errorf("failed to unlink tags: %w", err)
	}
	return nil
}

// GetTagsByItemID retrieves all tags for a specific item
func (db *DB) GetTagsByItemID(itemID string) ([]string, error) {
	rows, err := db.conn.Query(
		`SELECT t.name FROM tags t 
		JOIN item_tags it ON t.id = it.tag_id 
		WHERE it.item_id = ? 
		ORDER BY t.name`,
		itemID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, name)
	}

	return tags, rows.Err()
}

// ListKBItems retrieves paginated knowledge base items with optional tag filtering
func (db *DB) ListKBItems(tags []string, limit, offset int) (*PaginatedKBItems, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var query string
	var args []interface{}

	if len(tags) > 0 {
		// Filter by tags - items must have ALL specified tags
		query = `
			SELECT k.id, k.title, k.content, k.tldr, k.original_url, k.impact_score, k.created_at, k.updated_at
			FROM kb_items k
			JOIN item_tags it ON k.id = it.item_id
			JOIN tags t ON it.tag_id = t.id
			WHERE t.name IN (` + placeholders(len(tags)) + `)
			GROUP BY k.id
			HAVING COUNT(DISTINCT t.name) = ?
			ORDER BY k.created_at DESC
			LIMIT ? OFFSET ?
		`
		for _, tag := range tags {
			args = append(args, strings.ToLower(tag))
		}
		args = append(args, len(tags), limit, offset)
	} else {
		query = `
			SELECT id, title, content, tldr, original_url, impact_score, created_at, updated_at
			FROM kb_items
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query kb_items: %w", err)
	}
	defer rows.Close()

	var items []KBItem
	for rows.Next() {
		var item KBItem
		var createdAt, updatedAt string
		err := rows.Scan(&item.ID, &item.Title, &item.Content, &item.TLDR, &item.OriginalURL, &item.ImpactScore, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan kb_item: %w", err)
		}
		item.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		item.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Load tags for each item
	for i := range items {
		tags, err := db.GetTagsByItemID(items[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for item %s: %w", items[i].ID, err)
		}
		items[i].Tags = tags
	}

	// Get total count
	total, err := db.GetTotalCount(tags)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	page := offset/limit + 1

	return &PaginatedKBItems{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// GetTotalCount returns the total number of items, optionally filtered by tags
func (db *DB) GetTotalCount(tags []string) (int, error) {
	var query string
	var args []interface{}
	var count int

	if len(tags) > 0 {
		query = `
			SELECT COUNT(DISTINCT k.id)
			FROM kb_items k
			JOIN item_tags it ON k.id = it.item_id
			JOIN tags t ON it.tag_id = t.id
			WHERE t.name IN (` + placeholders(len(tags)) + `)
			GROUP BY k.id
			HAVING COUNT(DISTINCT t.name) = ?
		`
		for _, tag := range tags {
			args = append(args, strings.ToLower(tag))
		}
		args = append(args, len(tags))
	} else {
		query = `SELECT COUNT(*) FROM kb_items`
	}

	err := db.conn.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count kb_items: %w", err)
	}

	return count, nil
}

// GetAllTags retrieves all tags with their usage counts for the tag cloud
func (db *DB) GetAllTags() ([]TagCloudItem, error) {
	rows, err := db.conn.Query(`
		SELECT t.name, COUNT(it.item_id) as count
		FROM tags t
		LEFT JOIN item_tags it ON t.id = it.tag_id
		GROUP BY t.id, t.name
		ORDER BY count DESC, t.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	var tags []TagCloudItem
	for rows.Next() {
		var item TagCloudItem
		if err := rows.Scan(&item.Name, &item.Count); err != nil {
			return nil, fmt.Errorf("failed to scan tag: %w", err)
		}
		tags = append(tags, item)
	}

	return tags, rows.Err()
}

// GenerateSlug creates a URL-friendly slug from a title
func GenerateSlug(title string) string {
	// Simple slug generation - can be enhanced
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "\\", "-")

	// Remove non-alphanumeric characters except dash
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	slug = result.String()

	// Remove consecutive dashes
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim dashes
	slug = strings.Trim(slug, "-")

	if slug == "" {
		// Fallback to MD5 hash
		hash := md5.Sum([]byte(title))
		slug = hex.EncodeToString(hash[:8])
	}

	return slug
}

// placeholders generates SQL placeholders for IN clause
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}
