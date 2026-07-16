//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("Desktop Helper " + appVersion + " Go UI 当前只支持 Windows。请在 Windows 上运行：go run ./cmd/desktop-helper")
}
