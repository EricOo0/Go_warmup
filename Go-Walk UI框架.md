编译命令

```go
mac 下编译linux和windows命令
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build
```

github.com/lxn/walk/declarative





```
go build -ldflags="-H windowsgui"
```