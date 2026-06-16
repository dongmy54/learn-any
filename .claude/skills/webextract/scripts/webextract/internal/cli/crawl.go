package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"webextract/internal/crawler"
	"webextract/internal/fetcher"
)

// crawl 子命令专有 flag。--render / --wait 复用根命令已有的同名 flag，
// 由 flagRender / flagWait 控制 Fetcher 实现，无需在此重复声明。
var (
	cFlagMax     int
	cFlagDepth   int
	cFlagConc    int
	cFlagSameDom bool
	cFlagOutDir  string
	cFlagDelay   time.Duration
)

// crawlCmd 作为子命令挂载到 rootCmd；根命令（单页模式）保持原样，零回归。
var crawlCmd = &cobra.Command{
	Use:   "crawl [url]",
	Short: "批量抓取：从种子 URL 出发，抓取整个站点相关页面到目录",
	Long: "crawl 以广度优先方式从种子 URL 抓取，主链接页面优先且必抓，\n" +
		"随后并发抓取其相关链接，每页正文（Markdown）写入目录。\n\n" +
		"默认仅抓同域链接，并通过 --max 控制规模、--depth 控制深度。",
	Args: cobra.ExactArgs(1),
	RunE: runCrawl,
}

func init() {
	rootCmd.AddCommand(crawlCmd)
	crawlCmd.Flags().IntVar(&cFlagMax, "max", 10, "最大抓取页数（含种子）")
	crawlCmd.Flags().IntVar(&cFlagDepth, "depth", 1, "抓取深度（0=仅种子页）")
	crawlCmd.Flags().IntVar(&cFlagConc, "concurrency", 5, "并发 worker 数")
	crawlCmd.Flags().BoolVar(&cFlagSameDom, "same-domain", true, "仅抓同域链接（防爬虫外溢）")
	crawlCmd.Flags().StringVarP(&cFlagOutDir, "output", "o", "docs", "输出目录")
	crawlCmd.Flags().DurationVar(&cFlagDelay, "delay", 0, "每请求间隔（限流，如 500ms）")
}

func runCrawl(cmd *cobra.Command, args []string) error {
	seed := args[0]

	// 复用根命令 --render 决定 Fetcher 实现，与单页模式行为一致
	var f fetcher.Fetcher
	if flagRender {
		f = fetcher.NewDynamicFetcher(flagWait)
		if closer, ok := f.(interface{ Close() error }); ok {
			defer closer.Close()
		}
	} else {
		f = fetcher.NewStaticFetcher()
	}
	// TODO: cFlagDelay > 0 时用限流装饰器包裹 f（当前 MVP 暂未实现限流）

	report, err := crawler.Crawl(context.Background(), seed, f, crawler.Options{
		Max:         cFlagMax,
		MaxDepth:    cFlagDepth,
		Concurrency: cFlagConc,
		SameDomain:  cFlagSameDom,
		OutputDir:   cFlagOutDir,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(),
		"批量抓取完成：%d 页成功（含主链接），%d 页失败 → %s\n",
		report.Total, report.Failed, cFlagOutDir)
	return nil
}
