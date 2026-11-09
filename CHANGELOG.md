# Changelog

## 0.11.0

- `/status` 改为固定静态资源，完全移除会话数据注入和基于 GET 查询参数的状态变更。
- 状态查询迁移至认证的 GET management API，所有操作迁移至认证的 POST management API。
- OMC 使用已登录的 plugin-host 代理；原生页面通过仅保存在内存中的 Management Key 连接。
- 认证失败停止自动轮询，避免重复错误请求。
- 增加五平台原生构建、测试、动态库加载和统一 Release 打包流程。
- 修复商店审核 #345 提出的资源路由与发布平台问题，采用新版本重新提交。
