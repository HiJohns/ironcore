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

