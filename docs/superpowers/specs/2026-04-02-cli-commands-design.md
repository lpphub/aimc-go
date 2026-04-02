# CLI 命令功能设计

## 目标

为 agent CLI 增加 `/new`、`/resume`、`/quit` 命令，支持会话管理。

## 架构

新增 `assistant/command/` 包，职责分离：
- `command.go` - 命令注册、解析、分发
- `new.go` - `/new` 实现
- `resume.go` - `/resume` 实现
- `quit.go` - `/quit` 实现

`cli.go` 改为：解析输入 → 调用命令或发送给 agent。

`store/store.go` 扩展：新增 `ListSessions()` 和 `GetRecent()` 方法。

## 命令接口

```go
type Command func(ctx context.Context, deps *Dependencies, args string) error

type Dependencies struct {
    Store      store.Store
    Runner     *agent.Runner
    SessionID  *string  // 指向当前 session ID，命令可修改
    Scanner    *bufio.Scanner
}
```

所有命令共享相同签名，`args` 是命令后的额外参数。

## Store 扩展

新增方法：

```go
// ListSessions 返回所有会话（按创建时间倒序）
func (s *JSONLStore) ListSessions() ([]*Session, error)

// GetRecent 返回最近 N 个会话
func (s *JSONLStore) GetRecent(n int) ([]*Session, error)
```

实现：扫描 `data/sessions/*.jsonl`，解析 header 获取 ID 和创建时间，按时间排序。

## 命令实现

### `/new`
- 生成新 UUID
- 更新 `deps.SessionID`
- 输出：`✓ 已切换到新会话: <id>`

### `/resume`
- 无参数：`store.GetRecent(1)`，切换到最近的会话（排除当前）
- 有参数：验证存在，切换到指定会话
- 输出：`✓ 已恢复会话: <id>`

### `/quit`
- 输出：`再见！`
- `os.Exit(0)`

## CLI 流程

```
输入 → 以 '/' 开头？
  是 → 解析命令，执行
  否 → runner.Run(ctx, sessionID, input)
```

## 文件变更

- 新增 `assistant/command/command.go`
- 新增 `assistant/command/new.go`
- 新增 `assistant/command/resume.go`
- 新增 `assistant/command/quit.go`
- 修改 `assistant/cli.go`
- 修改 `assistant/store/store.go`