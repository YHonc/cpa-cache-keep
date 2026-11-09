# 验证记录

版本：0.11.0。实际部署：Linux amd64，CPA 8.0.59，Oh My CPA 0.3.23。

验证日期：2026-11-09（Asia/Hong_Kong）。

| 检查 | 结果 |
| --- | --- |
| `go test ./...` | 通过 |
| `go test -race ./...` | 通过 |
| C shared 动态库在独立进程加载 | 通过 |
| 插件请求收录、勾选与文件持久化 | 通过 |
| JSON 状态不包含重放正文、认证头或 API Key | 通过 |
| OMC 登录后的静态 HTML、management 状态代理 | 200；实际版本 0.11.0 |
| OMC 登录后的 management POST | 缺少会话 ID 返回 400，确认进入插件操作路由；未改变用户会话 |
| CPA 原生 management 状态和操作未认证请求 | 均返回 401；带 Management Key 的状态请求返回 200 |
| resource 携带旧状态、开关、删除、设置、重放参数 | 均返回相同静态 HTML；会话不变 |
| resource POST、management actions GET | 404 / 405，不能修改状态 |
| OMC 未登录状态请求 | 被拒绝 |
| 升级后的会话、CPA 配置 | 会话保留，配置文件哈希不变 |
| 原生资源公网隔离 | 404，保留已认证代理入口 |
| CPA、OMC 服务 | active |

## 五平台发布验证

[原生 CI](https://github.com/YHonc/cpa-cache-keep/actions/runs/42555320351) 对 Darwin amd64、Darwin arm64、Linux amd64、Linux arm64、Windows amd64 分别执行单元测试、race 测试、C shared 构建和 ctypes 加载及路由检查，全部通过。发布源码为 `23c388ce596529aa9db2719eeb6c769bcca768ee`。

[v0.11.0 Release](https://github.com/YHonc/cpa-cache-keep/releases/tag/v0.11.0) 包含五个平台 ZIP 和 checksums.txt。五个包均通过匿名公开下载、SHA-256、ZIP 完整性和根目录动态库文件名检查。跨平台结果覆盖原生构建、测试和加载；实际 CPA 部署验证在 Linux amd64 完成。

已安装 Linux 动态库 SHA-256：`e9b92e50a3cc9c92e2a2f87613e27fbb9cdc09f4dbd44b06b1b8340cf5f12992`。

## 真实上游验证

以下为 2026-11-08 在 0.10.0 上完成的上游验证；0.11.0 修改路由和发布流程，未重跑模型调用。使用受控合成请求调用 `claude-haiku-5-5`，每次输出上限 1 token，显式携带独立会话 ID 与 1h cache_control。

- 首次请求写入 22,558 token；响应的 `ephemeral_1h_input_tokens=22558`、`ephemeral_5m_input_tokens=0`。
- 第二个会话使用不同 ID，插件中形成独立条目。
- 开启第一个会话并执行立即续期，实际保活读取 22,558 token，写入 0 token。
- 在第一个会话继续发送普通请求，原会话 ID 与选择保留；最近请求时间更新，下次续期更新为最近请求后 50 分钟。
- 验证结束后移除合成会话，未留下测试保活计划。

这组结果验证了 Messages 入口的真实缓存读取和显式会话 ID 归组。50 分钟到期发送使用构造时间测试，未实等 50 分钟。真实 Magpie / Codex 链路的会话 ID 保留情况，以及 Responses / Chat Completions 重放的上游命中，需要分别验证。

## 界面

原型已检查宽屏与 375px 窄屏、明暗切换、主题记忆、问号说明与模拟请求顺延。整合页通过 JavaScript 语法检查、静态资源隔离测试及真实 OMC HTML / management 接口检查；整合后的浏览器复查遇到连接超时，未将其记为通过。
