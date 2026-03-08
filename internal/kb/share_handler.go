// internal/kb/share_handler.go
package kb

import (
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// ShareHandler handles share page requests
type ShareHandler struct {
	db *DB
}

// NewShareHandler creates a new share handler
func NewShareHandler(dbPath string) (*ShareHandler, error) {
	db, err := NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &ShareHandler{db: db}, nil
}

// Close closes the handler's database connection
func (h *ShareHandler) Close() error {
	return h.db.Close()
}

// RegisterRoutes registers share routes (no auth required)
func (h *ShareHandler) RegisterRoutes(mux *http.ServeMux) {
	// Public route - no authentication required
	mux.HandleFunc("/share/", h.HandleShare)
}

// HandleShare handles GET /share/:id endpoint
func (h *ShareHandler) HandleShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/share/")
	id := strings.TrimSpace(path)

	if id == "" {
		http.Error(w, "Invalid share ID", http.StatusBadRequest)
		return
	}

	// Retrieve item from database
	item, err := h.db.GetKBItem(id)
	if err != nil {
		http.Error(w, "Knowledge item not found", http.StatusNotFound)
		return
	}

	// Convert markdown to HTML
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	var htmlContent strings.Builder
	if err := markdown.Convert([]byte(item.Content), &htmlContent); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := struct {
		Title       string
		TLDR        string
		Content     template.HTML
		Tags        []string
		OriginalURL string
		ImpactScore float64
		CreatedAt   string
	}{
		Title:       item.Title,
		TLDR:        item.TLDR,
		Content:     template.HTML(htmlContent.String()),
		Tags:        item.Tags,
		OriginalURL: item.OriginalURL,
		ImpactScore: item.ImpactScore,
		CreatedAt:   item.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	// Render template
	tmpl, err := template.New("share").Parse(shareTemplate)
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[KB] Failed to render share template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

// shareTemplate is the HTML template for the share page
const shareTemplate = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - IronCore KB</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/themes/prism-tomorrow.min.css">
    <style>
        * {
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background: #fafafa;
        }
        .container {
            background: white;
            border-radius: 8px;
            padding: 30px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
        h1 {
            color: #1a1a2e;
            border-bottom: 2px solid #00d4ff;
            padding-bottom: 10px;
            margin-bottom: 20px;
        }
        .tldr {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 15px;
            border-radius: 8px;
            margin-bottom: 25px;
            font-size: 15px;
        }
        .tldr-label {
            font-weight: bold;
            font-size: 12px;
            text-transform: uppercase;
            opacity: 0.8;
            margin-bottom: 5px;
        }
        .meta {
            display: flex;
            flex-wrap: wrap;
            gap: 15px;
            margin-bottom: 25px;
            padding: 10px 0;
            border-bottom: 1px solid #eee;
            font-size: 14px;
            color: #666;
        }
        .meta-item {
            display: flex;
            align-items: center;
            gap: 5px;
        }
        .tags {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
        }
        .tag {
            background: #e3f2fd;
            color: #1976d2;
            padding: 4px 12px;
            border-radius: 16px;
            font-size: 13px;
            text-decoration: none;
        }
        .impact-badge {
            background: #ff6b6b;
            color: white;
            padding: 4px 10px;
            border-radius: 4px;
            font-weight: bold;
        }
        .content {
            margin-top: 30px;
        }
        .content h2 {
            color: #1a1a2e;
            margin-top: 30px;
            margin-bottom: 15px;
        }
        .content h3 {
            color: #333;
            margin-top: 25px;
            margin-bottom: 12px;
        }
        .content p {
            margin-bottom: 15px;
        }
        .content code {
            background: #f4f4f4;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.9em;
        }
        .content pre {
            background: #2d2d2d;
            color: #f8f8f2;
            padding: 20px;
            border-radius: 8px;
            overflow-x: auto;
            margin: 20px 0;
        }
        .content pre code {
            background: none;
            padding: 0;
        }
        .content blockquote {
            border-left: 4px solid #00d4ff;
            margin: 20px 0;
            padding-left: 20px;
            color: #666;
            font-style: italic;
        }
        .content ul, .content ol {
            margin-bottom: 15px;
            padding-left: 25px;
        }
        .content li {
            margin-bottom: 8px;
        }
        .content a {
            color: #667eea;
            text-decoration: none;
        }
        .content a:hover {
            text-decoration: underline;
        }
        .content img {
            max-width: 100%;
            height: auto;
            border-radius: 8px;
            margin: 20px 0;
        }
        .content table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
        }
        .content th, .content td {
            border: 1px solid #ddd;
            padding: 12px;
            text-align: left;
        }
        .content th {
            background: #f5f5f5;
            font-weight: bold;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            text-align: center;
            font-size: 13px;
            color: #999;
        }
        .original-link {
            display: inline-block;
            margin-top: 10px;
            color: #667eea;
        }
        @media (max-width: 600px) {
            body {
                padding: 10px;
            }
            .container {
                padding: 20px;
            }
            h1 {
                font-size: 24px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{.Title}}</h1>
        
        {{if .TLDR}}
        <div class="tldr">
            <div class="tldr-label">TL;DR</div>
            {{.TLDR}}
        </div>
        {{end}}
        
        <div class="meta">
            {{if .Tags}}
            <div class="meta-item">
                <span>标签:</span>
                <div class="tags">
                    {{range .Tags}}
                    <span class="tag">{{.}}</span>
                    {{end}}
                </div>
            </div>
            {{end}}
            <div class="meta-item">
                <span>影响分:</span>
                <span class="impact-badge">{{printf "%.2f" .ImpactScore}}</span>
            </div>
            <div class="meta-item">
                <span>时间:</span>
                <span>{{.CreatedAt}}</span>
            </div>
        </div>
        
        <div class="content">
            {{.Content}}
        </div>
        
        {{if .OriginalURL}}
        <div class="footer">
            <a href="{{.OriginalURL}}" class="original-link" target="_blank">查看原始来源 ↗</a>
            <p>由 IronCore Knowledge Base 生成</p>
        </div>
        {{else}}
        <div class="footer">
            <p>由 IronCore Knowledge Base 生成</p>
        </div>
        {{end}}
    </div>
    
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/components/prism-core.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/prism/1.29.0/plugins/autoloader/prism-autoloader.min.js"></script>
</body>
</html>`
