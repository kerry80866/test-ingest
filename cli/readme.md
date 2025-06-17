# Solana 数据查询工具

本目录包含针对 **Lindorm 宽表引擎** 存储的 Solana 链上数据的各类查询脚本，适用于开发者排查事件、验证数据与调试链上状态。

所有脚本均使用 Go 编写，依赖 `consts.go` 中的 DSN 配置。执行示例：

```bash
go run consts.go <脚本文件名>.go [参数...]
```

---

## 🛠 索引管理

### 重新构建所有索引

```bash
go run consts.go build_indexes.go
```

---

## 🧮 event_id 构造工具

根据 `tx_hash` 和指令位置构造 `event_id`：

```bash
go run build_event_id.go <tx_hash> <ix_index> <inner_index>
```

---

## 📦 chain_event 查询脚本

### 检查所有表索引

```bash
go run consts.go check_indexer.go
```

### 按 event_id 查询事件（含解析）

```bash
go run consts.go query_event_by_id.go <event_id>
```

---

## 👤 user_wallet + token 查询事件

```bash
go run consts.go query_by_user_token.go <user_wallet> <token>
```

支持以下组合：

- user_wallet + token
- user_wallet + token + event_type
- user_wallet + token + event_type + event_id

---

## 🏊 pool_address 查询事件

```bash
go run consts.go query_by_pool.go <pool_address>
```

支持以下组合：

- pool_address
- pool_address + event_type
- pool_address + event_type + event_id

---

## 💰 balance 表查询

### 按 token 查询 holders（使用 idx_balance_token）

```bash
go run consts.go query_balance.go <token_address>
```

### token + owner 精确匹配

```bash
go run consts.go query_balance.go <token_address> <owner_address>
```

### 按 owner 查询（使用 idx_balance_owner_token）

```bash
go run consts.go query_balance.go -o <owner_address>
```

### owner + token 精确匹配

```bash
go run consts.go query_balance.go -o <owner_address> <token_address>
```

---

## 📄 精确 account_address 查询 balance 记录

```bash
go run consts.go query_balance_by_account.go <account_address>
```

---

## 🔍 查询池子信息（pool 表）

### 查询指定池子地址

```bash
go run consts.go query_pool.go <pool_address>
```

### 查询某 token 相关的所有池子

```bash
go run consts.go query_pool.go -t <token_address>
```

### 查询 token + quote 的池子（使用 idx_pool_token_quote）

```bash
go run consts.go query_pool.go -t <token_address> <quote_address>
```

---

