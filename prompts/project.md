[TODO]

[WIP]

[READY]

[READY]

[TODO]

### 16. 统一时区准入 (Timezone Unification)

### 20. 刷新状态持久化 (Refresh State Persistence)
- 在 handleDashboard (main.go:1986) 中，增加对 Cookie active_tab 的读取
- 修改后端渲染逻辑：若 Cookie 值为 kb，则初始 HTML 中 assets-tab 设为 display:none，kb-tab 设为 display:block
- 状态：✅ 已完成

### 16. 统一时区准入 (Timezone Unification)
- 废弃冗余的 isSilentPeriod() 函数
- 强制所有审计入口（包括邮件发送逻辑）调用 isMarketOpen(asset)
- 确保北京时间周一中午时，由于 America/New_York 处于闭市态，系统不再处理 USO 等标的
- 状态：🔄 执行中

### 17. 数据库连接修复 (DB Connection Fix)
- 修正 main.go:1199-1207
- 确保 DashboardData 接口正确使用已初始化的全局 DB 实例
- 状态：⏳ 待执行

### 18. 异步清理机制 (Async Cleanup)
- 在 kb_handler.go 中为 jobStore 增加 TTL（过期时间）逻辑
- 防止内存泄漏
- 状态：⏳ 待执行

[READY]

[WIP]

### 6. 知识库系统扩展 (Knowledge Base System)

**6.1 数据库架构扩展**
- 在 ironcore.db 中新增三张表：
  - `kb_items`: 存储标题、Markdown正文、UUID、TLDR、原链接、审计分、时间戳
  - `tags`: 唯一名称
  - `item_tags`: 建立多对多关联
- 确保 kb_items.id 使用字符串类型，存储 UUID 或基于标题生成的 Slug
- 状态：✅ 已完成

**6.2 迁移脚本 scripts/migrate_v1.py**
- 遍历逻辑：扫描旧 ~/knowledge 目录下的所有 .md 文件
- 路径转标签：将文件所在的父级目录名（如 ai/agent/）自动转化为数据库标签 ai 和 agent
- 内容解析：解析 Markdown 的 Frontmatter，提取 title 和 original_url；若无，则以文件名为标题
- 图片迁移：识别正文中的本地图片路径，将图片移动到 storage/assets/ 目录下（重命名为 MD5 哈希值以防重复），并同步更新 Markdown 中的图片引用链接
- 入库执行：将清洗后的数据写入 SQLite
- 状态：✅ 已完成

**6.3 Go 后端 Ingest 路由**
- 在 main.go 中新增 POST /api/kb/ingest 接口（需 JWT 保护）
- 逻辑：接收 content 字段。使用正则判定：若是 http 开头则标记为 type: url，否则标记为 type: raw_text
- 将任务通过本地 Socket 或直接调用命令行的形式发送给 sentinel.py 处理
- 状态：✅ 已完成

**6.4 升级 sentinel.py 知识审计能力**
- 新增 process_kb_ingest 函数，集成现有 LLMScorer 逻辑
- Gemini Prompt 约束：要求 AI 返回 JSON：包含 title、tags (自动根据内容判定)、tldr、以及一个针对铁核资产标的的 impact_score (0-1)
- 如果是 URL，调用抓取逻辑（可复用旧 fetch.py 核心代码）；如果是纯文本，直接脱水
- 入库操作：处理完后，将最终 Markdown 和标签写入 kb_items
- 状态：✅ 已完成

### 7. 免鉴权预览与前端改造 (Share & Frontend)

**7.1 免鉴权预览路由 GET /share/:id**
- 在 main.go 的路由注册中，将此路径排除在 AuthMiddleware 之外
- 渲染逻辑：从数据库读取 Markdown，后端利用 github.com/yuin/goldmark 将其转为 HTML
- 前端模板：返回一个极简的静态页面，支持移动端阅读、代码高亮（建议集成 Prism.js 或类似物）
- 状态：✅ 已完成

**7.2 前端 Dashboard 改造**
- 增加"知识库"选项卡
- 录入组件：设计一个"万能投递框"，支持粘贴 URL 或长文本。点击提交后，显示 Loading 状态直到 AI 审计完成
- 标签云过滤：从 tags 表拉取数据，点击标签即可过滤下方的知识点列表
- 状态：✅ 已完成

### 7. 免鉴权预览与前端改造 (Share & Frontend)

**7.1 免鉴权预览路由 GET /share/:id**
- 在 main.go 的路由注册中，将此路径排除在 AuthMiddleware 之外
- 渲染逻辑：从数据库读取 Markdown，后端利用 github.com/yuin/goldmark 将其转为 HTML
- 前端模板：返回一个极简的静态页面，支持移动端阅读、代码高亮（建议集成 Prism.js 或类似物）

**7.2 前端 Dashboard 改造**
- 增加"知识库"选项卡
- 录入组件：设计一个"万能投递框"，支持粘贴 URL 或长文本。点击提交后，显示 Loading 状态直到 AI 审计完成
- 标签云过滤：从 tags 表拉取数据，点击标签即可过滤下方的知识点列表
- 状态：⏳ 待执行

[WIP]

[READY]

### 12. 资产属性扩展 (Asset Timezone Config)
- 修改 Config 结构体，为每个资产类别增加 market_timezone 字段
- 在 config.yaml 中，为 global_macro 指定 America/New_York，为 china_power_grid 指定 Asia/Shanghai
- 状态：✅ 已完成

### 13. 抑制周末错误警报 (Market Open Detection)
- 增加 isMarketOpen(asset) 函数，识别特定市场的节假日和周末
- 支持 America/New_York（美股 09:30-16:00）和 Asia/Shanghai（A股 09:30-11:30, 13:00-15:00）
- 状态：✅ 已完成

### 14. 审计调度器改造 (Trading Hours Audit)
- 在 performAudit 函数中，对每个资产调用 isMarketOpen() 检查市场状态
- 仅当资产市场开放时才触发 3-Sigma 和竞价审计
- 状态：✅ 已完成

### 15. UI 状态同步 (Dashboard Silent Period Display)
- 新增 MarketStatusInfo 结构体和 calculateMarketStatuses() 函数
- Dashboard 顶部状态栏按市场分流显示：🇨🇳 交易中 / 🇺🇸 静默期（周末）
- 状态：✅ 已完成

### 13. 审计调度器改造 (Trading Hours Audit)
- 在 main.go 的审计循环中，调用 time.LoadLocation 获取资产对应的时区
- 逻辑判定：仅当该资产所在时区的 Now() 处于其法定交易时段内时，才触发 3-Sigma 和竞价审计
- 状态：⏳ 待执行

### 14. 抑制周末错误警报 (Market Open Detection)
- 增加 isMarketOpen(asset) 函数，识别特定市场的节假日和周末
- 例如：北京时间周一早上，系统应对 global_macro 标的保持静默，直到北京时间周一晚间美股开盘
- 状态：⏳ 待执行

### 15. UI 状态同步 (Dashboard Silent Period Display)
- Dashboard 顶部的"静默期"状态应按资产分流显示
- 例如："🇨🇳 交易中 | 🇺🇸 静默期（周末）"
- 状态：⏳ 待执行

[READY]

[DONE]

### [2026-03-09] Favicon、Gemini Prompt、标签清洗、标签云 UI 优化
**Status**: ✅ 已完成并通过代码审查
**Patches**: review_Task1_SVG_Favicon.patch, review_Task2_Gemini_Prompt_Upgrade.patch, review_Task3_Tag_Cleanup_Script.patch, review_Task4_5_6_7_MainGo_Updates.patch

**核心变更：**
1. **SVG Favicon 设计**
   - 为 report-v2 项目创建全新 SVG 格式 Favicon
   - 深蓝到亮青对角线渐变背景
   - 白色"R"字母配合向上趋势线品牌标识

2. **Gemini Prompt 打标升级**
   - 新增 "Strict Multi-word Term Bond" 约束规则
   - 强制多词术语使用连字符（open-source, ai-agent）
   - 禁止拆分专有名词
   - 优先从 IronCore 核心观察名单选择标签
   - 限制 3-5 个标签

3. **标签清洗脚本**
   - 实现标签聚合与清洗功能
   - 合并同义词（ai-agent/aiagent/agent → ai-agent）
   - 修复断词（open + source → open-source）
   - 过滤低频标签（count<2 且非监控关键词）
   - 保护核心关键词不被删除

4. **标签云 UI 优化**
   - 智能折叠：默认显示前20个标签
   - 权重排序：按关联文章数量降序
   - 核心关键词高亮：Transformer/VIX/Fed 等使用青色加粗
   - 长尾噪音过滤：隐藏 count=1 的非核心标签

5. **详情页排版优化**
   - Sidebar 详情面板使用 Georgia 衬线字体
   - 1.8 行高提升阅读体验
   - 添加代码块、引用、表格、图片等 Markdown 元素样式
   - 登录页面 UI 风格统一为深色主题

### [2026-03-09] KB Dashboard 渲染、Sidebar、全局搜索功能
**Status**: ✅ 已完成并通过代码审查
**Patches**: review_TaskA_DashboardDB_Fix.patch, review_TaskB_KBItems_Render.patch, review_TaskD_Global_Search.patch

**核心变更：**
1. **DashboardData 初始化修复**
   - 扩展 DashboardData 结构体添加 KBItems 字段
   - handleDashboard 函数中复用全局 DB 实例获取最近 10 条 KB 条目
   - 添加 nil 检查避免空指针异常

2. **KB 条目后端渲染**
   - 实现后端模板渲染 KB 条目网格卡片
   - 支持 ImpactScore 高亮显示(≥0.8 显示徽章)
   - TLDR 摘要展示和格式化日期显示

3. **Sidebar 详情面板**
   - 右侧滑出式详情面板，点击卡片打开
   - 通过 fetch('/share/:id') 加载 goldmark 转换后的 HTML
   - 集成 sanitizeHtml() XSS 防护
   - "生成外部链接"按钮支持复制 /share/:id 链接

4. **全局搜索功能**
   - 搜索栏支持关键词实时搜索
   - SearchKBItems() 支持标题、内容、TLDR、标签多字段模糊搜索
   - 修复时间解析错误显式处理、N+1 查询改为批量预加载
   - strconv.Atoi 参数错误添加日志记录

### [2026-03-09] UI 重构与简化 (删除 dashboard.go, Cookie 状态持久化, 预览逻辑简化)
**Status**: ✅ 已完成并通过代码审查
**Patches**: review_UI_Refactoring.patch

**核心变更：**
1. **清理冗余文件 (Cleanup Redundant Files)**
   - 删除未被引用的 internal/kb/dashboard.go (415行)
   - 将活跃 UI 逻辑收拢至 main.go 内嵌模板

2. **刷新状态持久化 (Refresh State Persistence)**
   - 新增 DashboardData 结构体包装 AuditStatus 和 ActiveTab
   - handleDashboard 读取 Cookie active_tab，验证值仅为 "assets" 或 "kb"
   - 后端模板条件渲染根据 ActiveTab 切换初始 Tab 显示

3. **预览逻辑简化 (Preview Logic Simplification)**
   - 移除预览 Modal 中的 iframe，直接使用 div 渲染
   - 新增 sanitizeHtml() 函数净化 HTML 内容，防止 XSS 攻击
   - 使用 {{js .Name}} 模板函数进行 JavaScript 上下文转义
   - 两处模板执行添加错误处理和日志记录

### [2026-03-09] 时区统一、数据库连接修复、异步清理机制
**Status**: ✅ 已完成并通过代码审查
**Patches**: review_IronCore_2.0_Fixes.patch

**核心变更：**
1. **时区统一准入 (Timezone Unification)**
   - 废弃冗余的 `isSilentPeriod()` 函数
   - 统一使用 `isMarketOpen(asset)` 按资产判断市场状态
   - 修复北京时间周一中午美股标的不被错误处理的问题

2. **数据库连接修复 (DB Connection Fix)**
   - 新增 `NewDBFromConn()` 和 `NewDashboardHandlerFromDB()` 函数
   - DashboardData 接口复用全局 DB 实例，避免连接泄漏

3. **异步清理机制 (Async Cleanup)**
   - jobStore 添加 TTL(24小时) 过期逻辑
   - 每小时后台清理协程，使用 `sync.Once` 确保只启动一次
   - 锁策略优化：RLock 读取，仅过期时升级 Lock，双重检查防竞争

### [2026-03-08] 知识库系统与动态配置引擎安全修复
**Status**: ✅ 已完成并通过代码审查
**Patches**: review_KB_*.patch (8 files), review_Config_Engine.patch

**核心变更：**
1. **知识库系统 (KB System)**
   - 数据库架构：kb_items, tags, item_tags 三表设计
   - Go后端：internal/kb/ 包（models.go, db.go, kb_handler.go, share_handler.go, dashboard.go）
   - 功能：万能投递框、AI审计、标签云、免鉴权分享 (/share/:id)
   - 安全：临时文件传递替代环境变量、10MB内容限制、异步状态查询

2. **安全修复**
   - 硬编码凭据改为环境变量（IRONCORE_ADMIN_PASS, IRONCORE_SESSION_SECRET）
   - 命令注入风险修复（临时文件方案）
   - 索引越界修复（strings.HasPrefix）
   - 错误处理完善（RowsAffected、fmt.Sscanf、模板渲染）

3. **动态配置引擎**
   - Config结构体支持热重载
   - 资产列表、阈值、关键词外置到 config.yaml
   - /api/reload-config 端点实现配置刷新

### 6-7. 知识库系统与分享功能 (KB System & Share) - 已完成 ✅
- 数据库架构：kb_items, tags, item_tags 三表设计
- 迁移脚本：scripts/migrate_v1.py 支持 Markdown 导入和图片迁移
- Go 后端：internal/kb/ 包实现模块化架构
  - models.go: 数据结构定义
  - db.go: 数据库操作封装
  - kb_handler.go: POST /api/kb/ingest (JWT 保护)
  - share_handler.go: GET /share/:id (免鉴权)
  - dashboard.go: 知识库 Dashboard 组件
- Sentinel 集成：新增 KBIngestProcessor 类处理 AI 审计
- 前端：Tab 导航 + 万能投递框 + 标签云过滤
- 日志规范：[KB] New Knowledge Ingested: "标题", Impact: 0.85

1. 解耦硬编码，建立动态配置中心 (Config Engine)
   - 重构 IronCore 的资产观察列表
   - 动态化：将 collector.py 和 main.go 中硬编码的 Ticker 列表（如 SRVR, 600406.SS 等）全部迁移到外部 config.yaml 或 assets.json 文件中
   - 分组管理：支持按分类配置，如 Global_Macro (Yahoo), China_Power_Grid (efinance), Sentinel_Keywords (News) 等
   - 热加载：Go 服务启动时读取该配置，并提供一个内部函数支持在不重启服务的情况下刷新观察名单
   - 状态：✅ 已完成 (配置系统已存在，已验证完整性)

2. 集成 Sentinel 哨兵模块与 AI 审计逻辑 (Sentinel Engine)
   - 新增 sentinel.py 独立采集模块
   - 新闻抓取：对接 NewsAPI 或 GNews，根据配置中的关键词（如：Hormuz, Ga2O3, Transformer Shortage）抓取全球主流媒体标题
   - AI 评分逻辑：对抓取的标题调用 LLM API，生成 0.0-1.0 的 ImpactScore
   - 数据交互：将评分结果存入 ironcore.db 的新表 news_events，并与相关资产的 Ticker 建立关联
   - 联动审计：修改 Go 引擎的 runAuditLoop，将 ImpactScore > 0.8 作为触发 3-Sigma 告警的加权因子
   - 状态：✅ 已完成

3. 修改 isSilentPeriod 逻辑及采集频率 (Auction Mode)
   - 取消静默：移除 9:00-9:30 的完全静默，将其定义为 High_Frequency_Auction_Mode
   - 竞价侦测：在 09:25 采集一次关键标的（如 159326.SZ）的集合竞价成交量
   - 异常触发：如果 09:25 的 Volume > 过去 5 日均值的 2 倍，立即在 Web Dashboard 标记"🔥 换血资金进场"，并发送特级告警
   - 状态：✅ 已完成

4. 数据库 Schema 升级与 Go 结构体对齐 (Data Schema)
   - SQLite 更新：
     - market_data 增加 turnover_rate（换手率）字段
     - 新建 news_events 表：timestamp, symbol, title, impact_score, sentiment, logic_summary
   - Go Struct 升级：在 AssetStatus 中增加 SentimentScore (float64) 和 LatestNews (string) 字段
   - 持久化逻辑：确保 collector.py 在存入价格的同时，能通过 API 同步 sentinel.py 的最新审计结论
   - 状态：✅ 已完成

5. 可视化仪表盘增强 (UI/UX Optimization)
   - 升级 plotter.py 和 Web 界面
   - 标注新闻事件：在相关性趋势图或价格线上，用小圆点标注 ImpactScore > 0.8 的新闻发生点，实现"图文合一"
   - 实时状态面板：在 /dashboard 增加一个"地缘政治风险灯"，根据 Sentinel 的平均评分显示：Green (Calm), Yellow (Tension), Red (Crisis)
   - 操作建议输出：根据 3-Sigma 异动 + 产业链共振 + AI 审计结果，在 API 返回值中生成一段人类可读的 TacticalAdvice（例如："检测到物理基建共振且地缘评分高，建议加仓 159326"）
   - 状态：✅ 已完成

## 🚨 Critical Safety Rules

**绝对禁止删除 ironcore.db**
- `ironcore.db` 是 IronCore 系统的核心数据资产，包含历史价格数据、新闻事件记录和审计日志
- 在任何情况下（包括测试、清理、重置）都严禁删除或清空该数据库文件
- 如需备份，应使用 `cp ironcore.db ironcore.db.backup.$(date +%Y%m%d)` 方式创建副本
- 违规删除将导致所有历史数据和训练结果永久丢失
