# Blog

## 2026-03-08

### Daily Summary

**IronCore 2.0 核心功能交付**: 完成警报系统周末检测、动态配置中心、数据采集增强(换手率)、集合竞价监控、新闻事件分析、战术建议引擎、并发安全加固及知识库系统(KB)完整实现，包含安全修复和 Build 脚本优化。

## 2026-03-09

### Raw Entry

- **KB Dashboard 交互优化**: 实现 Tab 状态持久化(localStorage)、顶部栏 UI 降级(Micro-Status-Bar)、最近列表 API(/api/kb/recent, /api/kb/list)、预览模态框、标签云点击交互。
- **安全加固**: 修复 kb_handler 输入验证(fmt.Sscanf→strconv.Atoi, limit≤100)、错误信息泄露(改为日志记录)、XSS 防护(data-*属性+事件委托)、DOM Clobbering 防护(iframe sandbox 隔离)、tag 参数正则验证。
- **市场时区审计系统**: 实现按资产时区的智能审计调度，支持美股(America/New_York)和A股(Asia/Shanghai)交易时段检测，休市期间自动跳过3-Sigma审计和竞价审计，Dashboard显示分市场状态(🇺🇸/🇨🇳)。
- **时区统一准入修复**: 废弃冗余的 isSilentPeriod() 函数，统一使用 isMarketOpen(asset) 按资产判断市场状态，确保美股标的在北京时间周一中午（美股周日休市）不被错误处理。
- **数据库连接修复**: 修复 DashboardData 接口重复创建数据库连接的问题，新增 NewDBFromConn() 和 NewDashboardHandlerFromDB() 函数，实现全局 DB 实例复用，避免连接泄漏。
- **异步清理机制**: 为 kb_handler.go 的 jobStore 添加 TTL(24小时) 过期逻辑和每小时后台清理协程，使用 sync.Once 确保协程只启动一次，防止内存泄漏。
- **UI 重构与简化**: 删除冗余 internal/kb/dashboard.go (415行)，将活跃 UI 逻辑收拢至 main.go 内嵌模板；实现刷新状态持久化(Cookie active_tab)，后端根据 Cookie 值切换初始 Tab 显示；简化预览逻辑，移除 iframe 直接使用 div 渲染，添加 sanitizeHtml() XSS 防护函数，使用 {{js .Name}} 模板转义防止注入。


### Raw Entry

- **KB Dashboard 渲染与交互**: 完成 DashboardData 结构体扩展(KBItems 字段)，实现后端渲染 KB 条目网格卡片，支持 ImpactScore 高亮显示(≥0.8)和 TLDR 摘要，后端模板条件渲染解决前端 hydration 问题。
- **Sidebar 详情面板**: 实现右侧滑出式详情面板，通过 fetch('/share/:id') 加载内容并使用 DOMParser 解析提取标题和正文，集成 sanitizeHtml() XSS 防护，提供"生成外部链接"按钮复制 /share/:id 链接到剪贴板。
- **全局搜索功能**: 实现 SearchKBItems() 支持标题、内容、TLDR、标签多字段模糊搜索，修复时间解析错误显式处理、N+1 查询改为批量预加载、strconv.Atoi 参数错误日志记录，前端实现 performSearch() 实时搜索和 renderSearchResults() 结果渲染。


### Raw Entry

- **SVG Favicon 设计**: 为 report-v2 项目创建全新的 SVG 格式 Favicon，采用深蓝到亮青的对角线渐变，白色"R"字母配合向上趋势线的品牌标识。
- **Gemini Prompt 打标升级**: 新增 "Strict Multi-word Term Bond" 约束规则，强制多词术语使用连字符（如 open-source），禁止拆分专有名词，优先从 IronCore 核心观察名单选择标签，限制 3-5 个标签。
- **标签清洗脚本**: 实现标签聚合与清洗功能，支持合并同义词（ai-agent/aiagent/agent → ai-agent）、修复断词（open + source → open-source）、过滤低频标签（count<2 且非监控关键词），保护核心关键词不被删除。
- **标签云 UI 优化**: 实现智能折叠（默认显示前20个标签，展开更多按钮）、权重排序（按关联文章数量降序）、核心关键词高亮（Transformer/VIX/Fed 等使用青色加粗显示）、长尾噪音过滤（隐藏 count=1 的非核心标签）。
- **详情页排版优化**: Sidebar 详情面板使用 Georgia 衬线字体、1.8 行高提升阅读体验，添加代码块、引用、表格、图片等 Markdown 元素的样式支持，登录页面 UI 风格统一为深色主题。

