# 设置环境变量
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"

# 执行编译
go build -x -o vrchat_osc.exe main.go