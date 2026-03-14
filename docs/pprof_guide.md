# AIGateway pprof + 压测操作指南（可直接复制执行）

## 终端 A：启动网关

```powershell
cd D:\ygrttdx\golangPrpjects\AIcoding\ClaudeCoding\AIGateway
go run main.go -endpoint server -config D:\ygrttdx\golangPrpjects\AIcoding\ClaudeCoding\AIGateway\conf\dev
```

## 终端 B：压测（可直接复制执行）

```powershell
go run .\scripts\loadtest\loadtest.go `
  -url http://127.0.0.1:8080/v1/chat/completions `
  -method POST `
  -H "Authorization: Bearer xxx" `
  -H "Content-Type: application/json" `
  -body '{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}' `
  -c 100 -d 120s -timeout 10s
```

## 终端 C：抓取各类指标（可直接复制执行）

### 1) CPU Profile（60 秒）

```powershell
go tool pprof "-http=127.0.0.1:6063" http://127.0.0.1:6062/debug/pprof/profile?seconds=60
```

### 2) Heap（当前占用）

```powershell
go tool pprof "-http=127.0.0.1:6064" http://127.0.0.1:6062/debug/pprof/heap
```

### 3) Allocs（累计分配）

```powershell
go tool pprof "-http=127.0.0.1:6065" http://127.0.0.1:6062/debug/pprof/allocs
```

### 4) Goroutine（协程状态）

```powershell
go tool pprof "-http=127.0.0.1:6066" http://127.0.0.1:6062/debug/pprof/goroutine
```

### 5) Block（阻塞）

```powershell
go tool pprof "-http=127.0.0.1:6067" http://127.0.0.1:6062/debug/pprof/block
```

### 6) Mutex（锁竞争）

```powershell
go tool pprof "-http=127.0.0.1:6068" http://127.0.0.1:6062/debug/pprof/mutex
```

## 补充：如果你只想导出文件

### CPU

```powershell
go tool pprof -output C:\Users\53039\pprof\pprof.main.exe.samples.cpu.003.pb.gz http://127.0.0.1:6062/debug/pprof/profile?seconds=60
```

### Heap

```powershell
go tool pprof -output C:\Users\53039\pprof\pprof.main.exe.samples.heap.003.pb.gz http://127.0.0.1:6062/debug/pprof/heap
```
