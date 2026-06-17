package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"webextract/internal/fetcher"
	"webextract/internal/pipeline"
)

var (
	flagRender bool          // --render：是否启用无头浏览器
	flagFormat string        // --format：markdown | json
	flagWait   time.Duration // --wait：动态渲染额外等待时长
	flagOutput string        // --output：输出文件路径（默认 stdout）
)

var rootCmd = &cobra.Command{
	Use:   "webextract [url]",
	Short: "提取任意网页正文，输出 Markdown 或 JSON",
	Long: "webextract 输入一个 URL，抓取网页并用 readability 算法提取正文，输出为 Markdown。\n" +
		"默认使用轻量 HTTP 静态抓取；对 React/Vue 等 JS 动态页面可加 --render 启用无头浏览器渲染。",
	Args: cobra.ExactArgs(1), // 强制恰好一个 URL 参数
	RunE: run,
}

func init() {
	// --render / --wait 用持久化 flag：这样单页（根命令）与 crawl 子命令都能使用，
	// 否则 `crawl --render` 会因子命令不继承根命令的本地 flag 而报「unknown flag」。
	rootCmd.PersistentFlags().BoolVarP(&flagRender, "render", "r", false,
		"启用无头浏览器渲染（用于 React/Vue 等 JS 动态页面）")
	rootCmd.PersistentFlags().DurationVar(&flagWait, "wait", 2*time.Second,
		"动态渲染后的额外等待时间")
	// --format / --output 是单页专属（crawl 自有 -o 输出目录），保持本地 flag 避免冲突。
	rootCmd.Flags().StringVarP(&flagFormat, "format", "f", "markdown",
		"输出格式: markdown | json")
	rootCmd.Flags().StringVarP(&flagOutput, "output", "o", "",
		"输出到文件（默认输出到 stdout）")
}

// run 是策略模式的注入点：根据 --render 选不同 Fetcher，其余流程完全一致
func run(cmd *cobra.Command, args []string) error {
	targetURL := args[0]

	var f fetcher.Fetcher
	if flagRender {
		f = fetcher.NewDynamicFetcher(flagWait)
		// 动态 fetcher 持有 headless Chrome 实例，用完必须关闭，避免进程残留
		if closer, ok := f.(interface{ Close() error }); ok {
			defer closer.Close()
		}
	} else {
		f = fetcher.NewStaticFetcher()
	}

	// 决定输出目标：文件或 stdout
	var out io.Writer = os.Stdout
	if flagOutput != "" {
		file, err := os.Create(flagOutput)
		if err != nil {
			return fmt.Errorf("无法创建输出文件: %w", err)
		}
		defer file.Close()
		out = file
	}

	// 流水线对具体 Fetcher 类型无感知，只依赖接口
	return pipeline.Run(cmd.Context(), f, targetURL, flagFormat, out)
}

// Execute 执行根命令
func Execute() error {
	return rootCmd.ExecuteContext(context.Background())
}
