package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/meimolihan/fan-files/version"
)

// cliVersionString 统一版本格式：v 前缀 + build。
func cliVersionString() string {
	v := version.Version
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	build := version.Build
	if build == "" {
		build = "dev"
	}
	return fmt.Sprintf("%s (%s)", v, build)
}

// cliVersionBlock 构建 2panel 风格彩色 KV 版本块，作为 cobra 的版本模板输出。
func cliVersionBlock() string {
	lines := []string{
		strings.TrimSuffix(cliBannerStr("版本信息"), "\n"),
		cliSepStr(),
		cliKVStr("版本", cliVersionString()),
		cliKVStr("Go 版本", runtime.Version()),
		cliKVStr("平台", runtime.GOOS+"/"+runtime.GOARCH),
		cliKVStr("编译时间", version.BuildTime),
		cliSepStr(),
	}
	return strings.Join(lines, "\n")
}