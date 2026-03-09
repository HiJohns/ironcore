// internal/kb/dashboard.go
package kb

import (
	"html/template"
	"net/http"
)

// DashboardHandler handles dashboard UI requests
type DashboardHandler struct {
	db *DB
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(dbPath string) (*DashboardHandler, error) {
	db, err := NewDB(dbPath)
	if err != nil {
		return nil, err
	}
	return &DashboardHandler{db: db}, nil
}

// NewDashboardHandlerFromDB creates a new dashboard handler from existing DB instance
func NewDashboardHandlerFromDB(db *DB) *DashboardHandler {
	return &DashboardHandler{db: db}
}

// Close closes the handler's database connection
func (h *DashboardHandler) Close() error {
	return h.db.Close()
}

// GetKnowledgeBaseHTML returns the HTML for the knowledge base tab
func (h *DashboardHandler) GetKnowledgeBaseHTML() string {
	return knowledgeBaseTabHTML
}

// HandleKnowledgeBaseData handles AJAX requests for knowledge base data
func (h *DashboardHandler) HandleKnowledgeBaseData(w http.ResponseWriter, r *http.Request) {
	// Get tags for cloud
	tags, err := h.db.GetAllTags()
	if err != nil {
		http.Error(w, "Failed to load tags", http.StatusInternalServerError)
		return
	}

	// Get recent items
	items, err := h.db.ListKBItems(nil, 10, 0)
	if err != nil {
		http.Error(w, "Failed to load items", http.StatusInternalServerError)
		return
	}

	// Build HTML response
	data := struct {
		Tags  []TagCloudItem
		Items []KBItem
	}{
		Tags:  tags,
		Items: items.Items,
	}

	tmpl, err := template.New("kb-data").Parse(kbDataTemplate)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, data)
}

// knowledgeBaseTabHTML is the HTML for the knowledge base tab in the dashboard
const knowledgeBaseTabHTML = `
<div id="kb-tab" class="section" style="display: none;">
    <h2>📚 知识库 (Knowledge Base)</h2>
    
    <!-- Universal Ingest Box -->
    <div class="kb-ingest-box">
        <h3>📝 万能投递框</h3>
        <textarea id="kb-content" placeholder="在此粘贴 URL 或长文本..." rows="4"></textarea>
        <button id="kb-submit" class="sync-btn" onclick="submitKB()">
            <span id="kb-btn-text">🚀 提交 AI 审计</span>
            <span id="kb-loading" style="display: none;">⏳ 处理中...</span>
        </button>
        <div id="kb-result" class="kb-result" style="display: none;"></div>
    </div>
    
    <!-- Tag Cloud -->
    <div class="kb-tag-cloud">
        <h3>🏷️ 标签云</h3>
        <div id="tag-cloud-container">
            <p>加载中...</p>
        </div>
    </div>
    
    <!-- Knowledge Items List -->
    <div class="kb-items-list">
        <h3>📖 最近录入</h3>
        <div id="kb-items-container">
            <p>加载中...</p>
        </div>
    </div>
</div>

<style>
.kb-ingest-box {
    background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
    padding: 20px;
    border-radius: 10px;
    margin-bottom: 30px;
    border: 1px solid #333;
}
.kb-ingest-box h3 {
    color: #00d4ff;
    margin-bottom: 15px;
}
.kb-ingest-box textarea {
    width: 100%;
    padding: 12px;
    border-radius: 6px;
    border: 1px solid #444;
    background: #0f0f1e;
    color: #eee;
    font-size: 14px;
    resize: vertical;
    font-family: inherit;
}
.kb-ingest-box textarea:focus {
    outline: none;
    border-color: #00d4ff;
}
.kb-result {
    margin-top: 15px;
    padding: 12px;
    border-radius: 6px;
    background: #0f3460;
}
.kb-result.success {
    background: rgba(40, 167, 69, 0.2);
    border: 1px solid #28a745;
}
.kb-result.error {
    background: rgba(220, 53, 69, 0.2);
    border: 1px solid #dc3545;
}
.kb-tag-cloud {
    margin-bottom: 30px;
}
.kb-tag-cloud h3 {
    color: #00d4ff;
    margin-bottom: 15px;
}
.tag-cloud {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
}
.tag-item {
    background: #2d3748;
    color: #e2e8f0;
    padding: 6px 14px;
    border-radius: 20px;
    cursor: pointer;
    transition: all 0.2s;
    font-size: 13px;
    border: 1px solid #4a5568;
}
.tag-item:hover {
    background: #667eea;
    border-color: #667eea;
    transform: translateY(-2px);
}
.tag-item.active {
    background: #00d4ff;
    color: #1a1a2e;
    border-color: #00d4ff;
}
.tag-count {
    opacity: 0.7;
    margin-left: 5px;
    font-size: 11px;
}
.kb-items-list h3 {
    color: #00d4ff;
    margin-bottom: 15px;
}
.kb-item {
    background: #16213e;
    padding: 15px;
    border-radius: 8px;
    margin-bottom: 10px;
    border-left: 3px solid #667eea;
    transition: all 0.2s;
}
.kb-item:hover {
    background: #1e2a4a;
    transform: translateX(5px);
}
.kb-item-title {
    font-weight: bold;
    color: #fff;
    margin-bottom: 8px;
    font-size: 15px;
}
.kb-item-meta {
    display: flex;
    gap: 15px;
    font-size: 12px;
    color: #888;
    flex-wrap: wrap;
}
.kb-item-tags {
    display: flex;
    gap: 5px;
}
.kb-item-tag {
    background: rgba(102, 126, 234, 0.2);
    color: #667eea;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
}
.kb-impact {
    background: #ff6b6b;
    color: white;
    padding: 2px 8px;
    border-radius: 4px;
    font-weight: bold;
}
.kb-item-tldr {
    margin-top: 10px;
    padding: 10px;
    background: rgba(102, 126, 234, 0.1);
    border-radius: 6px;
    font-size: 13px;
    color: #ccc;
    line-height: 1.5;
}
.kb-share-link {
    color: #00d4ff;
    text-decoration: none;
    margin-left: 10px;
}
.kb-share-link:hover {
    text-decoration: underline;
}
</style>

<script>
let selectedTags = [];

function submitKB() {
    const content = document.getElementById('kb-content').value.trim();
    if (!content) {
        alert('请输入内容');
        return;
    }
    
    const btn = document.getElementById('kb-submit');
    const btnText = document.getElementById('kb-btn-text');
    const loading = document.getElementById('kb-loading');
    const result = document.getElementById('kb-result');
    
    btn.disabled = true;
    btnText.style.display = 'none';
    loading.style.display = 'inline';
    
    fetch('/api/kb/ingest', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({content: content})
    })
    .then(r => r.json())
    .then(data => {
        result.style.display = 'block';
        if (data.status === 'processing') {
            result.className = 'kb-result success';
            result.innerHTML = '✅ ' + data.message + '<br>Item ID: ' + data.item_id;
            document.getElementById('kb-content').value = '';
            // Refresh list after a delay
            setTimeout(loadKBData, 3000);
        } else {
            result.className = 'kb-result error';
            result.innerHTML = '❌ 提交失败: ' + (data.message || 'Unknown error');
        }
    })
    .catch(err => {
        result.style.display = 'block';
        result.className = 'kb-result error';
        result.innerHTML = '❌ 网络错误: ' + err.message;
    })
    .finally(() => {
        btn.disabled = false;
        btnText.style.display = 'inline';
        loading.style.display = 'none';
    });
}

function loadKBData() {
    fetch('/api/kb/dashboard-data')
    .then(r => r.text())
    .then(html => {
        const container = document.createElement('div');
        container.innerHTML = html;
        
        // Update tag cloud
        const tagContainer = document.getElementById('tag-cloud-container');
        const tagCloud = container.querySelector('.tag-cloud');
        if (tagCloud && tagContainer) {
            tagContainer.innerHTML = '';
            tagContainer.appendChild(tagCloud);
        }
        
        // Update items list
        const itemsContainer = document.getElementById('kb-items-container');
        const itemsList = container.querySelector('.kb-items-container');
        if (itemsList && itemsContainer) {
            itemsContainer.innerHTML = '';
            itemsContainer.appendChild(itemsList);
        }
    })
    .catch(err => {
        console.error('Failed to load KB data:', err);
    });
}

function filterByTag(tag) {
    const index = selectedTags.indexOf(tag);
    if (index > -1) {
        selectedTags.splice(index, 1);
    } else {
        selectedTags.push(tag);
    }
    
    // Update UI
    document.querySelectorAll('.tag-item').forEach(el => {
        if (selectedTags.includes(el.dataset.tag)) {
            el.classList.add('active');
        } else {
            el.classList.remove('active');
        }
    });
    
    // Reload items with filter
    loadKBItems();
}

function loadKBItems() {
    let url = '/api/kb/items?limit=20';
    if (selectedTags.length > 0) {
        url += '&tags=' + encodeURIComponent(selectedTags.join(','));
    }
    
    fetch(url)
    .then(r => r.json())
    .then(data => {
        const container = document.getElementById('kb-items-container');
        if (data.items && data.items.length > 0) {
            container.innerHTML = data.items.map(item => 
                '<div class="kb-item">' +
                    '<div class="kb-item-title">' + escapeHtml(item.title) +
                        '<a href="/share/' + item.id + '" class="kb-share-link" target="_blank">🔗 分享</a>' +
                    '</div>' +
                    '<div class="kb-item-meta">' +
                        '<span>影响分: <span class="kb-impact">' + item.impact_score.toFixed(2) + '</span></span>' +
                        '<span>' + new Date(item.created_at).toLocaleString() + '</span>' +
                        (item.tags ? '<div class="kb-item-tags">' + item.tags.map(t => '<span class="kb-item-tag">' + escapeHtml(t) + '</span>').join('') + '</div>' : '') +
                    '</div>' +
                    (item.tldr ? '<div class="kb-item-tldr">' + escapeHtml(item.tldr) + '</div>' : '') +
                '</div>'
            ).join('');
        } else {
            container.innerHTML = '<p>暂无数据</p>';
        }
    })
    .catch(err => {
        console.error('Failed to load items:', err);
    });
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Load data when tab is shown
document.addEventListener('DOMContentLoaded', function() {
    const kbTabBtn = document.getElementById('kb-tab-btn');
    if (kbTabBtn) {
        kbTabBtn.addEventListener('click', loadKBData);
    }
});
</script>`

// kbDataTemplate is the template for KB data AJAX response
const kbDataTemplate = "{{range .Tags}}" +
	"<span class=\"tag-item\" data-tag=\"{{.Name}}\" onclick=\"filterByTag('{{.Name}}')\">" +
	"{{.Name}}<span class=\"tag-count\">{{.Count}}</span></span>" +
	"{{else}}<p style=\"color: #888;\">暂无标签</p>{{end}}" +
	"<div class=\"kb-items-container\">" +
	"{{range .Items}}" +
	"<div class=\"kb-item\"><div class=\"kb-item-title\">{{.Title}}" +
	"<a href=\"/share/{{.ID}}\" class=\"kb-share-link\" target=\"_blank\">🔗 分享</a></div>" +
	"<div class=\"kb-item-meta\"><span>影响分: <span class=\"kb-impact\">{{printf \"%.2f\" .ImpactScore}}</span></span>" +
	"<span>{{.CreatedAt.Format \"2006-01-02 15:04:05\"}}</span>" +
	"{{if .Tags}}<div class=\"kb-item-tags\">{{range .Tags}}<span class=\"kb-item-tag\">{{.}}</span>{{end}}</div>{{end}}</div>" +
	"{{if .TLDR}}<div class=\"kb-item-tldr\">{{.TLDR}}</div>{{end}}</div>" +
	"{{else}}<p style=\"color: #888;\">暂无知识条目</p>{{end}}</div>"

// GetKBTabButtonHTML returns the HTML for the KB tab button
func GetKBTabButtonHTML() string {
	return `<button class="tab-btn" id="kb-tab-btn" onclick="showTab('kb-tab')">📚 知识库</button>`
}
