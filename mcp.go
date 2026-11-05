package main

import (
	"context"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/crepidoma/lumetio/internal/app"
	"github.com/crepidoma/lumetio/internal/localconf"
	"github.com/crepidoma/lumetio/internal/mcpserver"
	"github.com/crepidoma/lumetio/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runMCP は、画面を出さずに MCP サーバ (stdio) として動く。
// 標準出力は MCP のやり取りに使うので、知らせは標準エラーに出す。
func runMCP() error {
	path, err := localconf.Path()
	if err != nil {
		return err
	}
	local, err := localconf.Load(path)
	if err != nil {
		return err
	}
	a := app.New(store.New(local.DataDir), local.OutputDir, nil)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return mcpserver.New(a, version()).Run(ctx, &mcp.StdioTransport{})
}

// version は、ビルドに埋め込まれたモジュールの版を返す。
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}
	return "(devel)"
}
