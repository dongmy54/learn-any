package crawler

import "testing"

// TestSeedNotOverwrittenByRootPage：种子页固定 index.md，而爬到的站点根页（空路径）
// 绝不能也映射成 index.md，否则会覆盖种子页（曾导致种子内容被首页替换、内容丢失）。
func TestSeedNotOverwrittenByRootPage(t *testing.T) {
	seed := urlToPath("https://openrouter.ai/docs/guides/overview/models", "out", true)
	root := urlToPath("https://openrouter.ai/", "out", false)
	if seed == root {
		t.Fatalf("种子页与站点根页路径冲突，会互相覆盖: %s", seed)
	}
	if root != "out/openrouter.ai/home.md" {
		t.Fatalf("站点根页应落到 home.md，实际 %s", root)
	}
}

// TestSeedFixedIndex：种子页无论路径如何都落到 index.md。
func TestSeedFixedIndex(t *testing.T) {
	got := urlToPath("https://e.com/a/b/c", "out", true)
	if got != "out/e.com/index.md" {
		t.Fatalf("种子页应固定 index.md，实际 %s", got)
	}
}

// TestNonSeedNeverClaimsIndex：非种子页即使路径恰为 /index 也不得占用 index.md。
func TestNonSeedNeverClaimsIndex(t *testing.T) {
	got := urlToPath("https://e.com/index", "out", false)
	if got == "out/e.com/index.md" {
		t.Fatalf("非种子的 /index 页不得占用种子专属的 index.md: %s", got)
	}
}

// TestDistinctPathsDistinctFiles：不同路径映射到不同文件，避免互相覆盖。
func TestDistinctPathsDistinctFiles(t *testing.T) {
	a := urlToPath("https://e.com/docs/intro", "out", false)
	b := urlToPath("https://e.com/docs/guide", "out", false)
	if a == b {
		t.Fatalf("不同路径不应映射到同一文件: %s", a)
	}
}

// TestSameQueryDistinctFiles：同路径不同 query 应落到不同文件。
func TestSameQueryDistinctFiles(t *testing.T) {
	a := urlToPath("https://e.com/list?page=1", "out", false)
	b := urlToPath("https://e.com/list?page=2", "out", false)
	if a == b {
		t.Fatalf("同路径不同 query 不应映射到同一文件: %s", a)
	}
}
