package pkg

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRepo(t *testing.T) {
	urls := []string{
		// ssh://[user@]host.xz[:port]/path/to/repo.git/
		"ssh://git@github.com:7875/go-kratos/kratos.git",
		// git://host.xz[:port]/path/to/repo.git/
		"git://github.com:7875/go-kratos/kratos.git",
		// http[s]://host.xz[:port]/path/to/repo.git/
		"https://github.com:7875/go-kratos/kratos.git",
		// ftp[s]://host.xz[:port]/path/to/repo.git/
		"ftps://github.com:7875/go-kratos/kratos.git",
		//[user@]host.xz:path/to/repo.git/
		"git@github.com:go-kratos/kratos.git",
		// ssh://[user@]host.xz[:port]/~[user]/path/to/repo.git/
		"ssh://git@github.com:7875/go-kratos/kratos.git",
		// git://host.xz[:port]/~[user]/path/to/repo.git/
		"git://github.com:7875/go-kratos/kratos.git",
		//[user@]host.xz:/~[user]/path/to/repo.git/
		"git@github.com:go-kratos/kratos.git",
		///path/to/repo.git/
		"//github.com/go-kratos/kratos.git",
		// file:///path/to/repo.git/
		"file://./github.com/go-kratos/kratos.git",
	}
	for _, url := range urls {
		dir := repoDir(url)
		if dir != "github.com/go-kratos" && dir != "/go-kratos" {
			t.Fatal(url, "repoDir test failed", dir)
		}
	}
}

func TestRepoClone(t *testing.T) {
	// 把 HOME 指到临时目录:NewRepo→kratosHome() 走 os.UserHomeDir(),Clone 会把
	// service-layout 缓存在 $HOME/.kratos/repo 下。不重定向就直接写进开发者真实的
	// ~/.kratos(本机这份缓存就是上一轮跑测试留下的),而且缓存一旦存在,Clone 会
	// 改走 git pull —— 换 HOME 让每次都是冷缓存,跑的是同一条 clone 路径。
	t.Setenv("HOME", t.TempDir())

	r := NewRepo("https://github.com/go-kratos/service-layout.git", "")
	if err := r.Clone(context.Background()); err != nil {
		t.Fatal(err)
	}
	// 落地目录也用临时路径:原先写死 /tmp/test_repo,同机并发跑会互相踩。
	to := filepath.Join(t.TempDir(), "test_repo")
	if err := r.CopyTo(context.Background(), to, "github.com/go-kratos/kratos-layout", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(to, "go.mod")); err != nil {
		t.Fatalf("CopyTo 没有把模板落到 %s: %v", to, err)
	}
}
