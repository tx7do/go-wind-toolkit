package pkg

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

var unExpandVarPath = []string{"~", ".", ".."}

// Repo is git repository manager.
type Repo struct {
	url    string
	home   string
	branch string
}

func repoDir(url string) string {
	vcsURL, err := ParseVCSUrl(url)
	if err != nil {
		return url
	}
	// check host contains port
	host, _, err := net.SplitHostPort(vcsURL.Host)
	if err != nil {
		host = vcsURL.Host
	}
	// Blank path-traversal-looking host segments ("~", ".", "..") by exact
	// match so they never contribute to the cache directory name. A TrimLeft
	// here would strip arbitrary leading characters instead of whole segments.
	for _, seg := range unExpandVarPath {
		if host == seg {
			host = ""
			break
		}
	}
	dir := path.Base(path.Dir(vcsURL.Path))
	url = fmt.Sprintf("%s/%s", host, dir)
	return url
}

// NewRepo new a repository manager.
func NewRepo(url string, branch string) *Repo {
	return &Repo{
		url:    url,
		home:   kratosHomeWithDir("repo/" + repoDir(url)),
		branch: branch,
	}
}

// Path returns the repository cache path.
func (r *Repo) Path() string {
	// Repo name = last path segment of the URL with a ".git" suffix removed.
	// Derived via path.Base so the extraction can never go out of slice
	// bounds, regardless of where ".git" or "/" appear in the URL.
	repo := strings.TrimSuffix(path.Base(strings.TrimSuffix(r.url, "/")), ".git")
	var branch string
	if r.branch == "" {
		branch = "@main"
	} else {
		branch = "@" + r.branch
	}
	return path.Join(r.home, repo+branch)
}

// Pull fetch the repository from remote url.
func (r *Repo) Pull(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "symbolic-ref", "HEAD")
	cmd.Dir = r.Path()
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	cmd = exec.CommandContext(ctx, "git", "pull")
	cmd.Dir = r.Path()
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		return err
	}
	return err
}

// Clone clones the repository to cache path.
func (r *Repo) Clone(ctx context.Context) error {
	if _, err := os.Stat(r.Path()); !os.IsNotExist(err) {
		return r.Pull(ctx)
	}
	var cmd *exec.Cmd
	if r.branch == "" {
		cmd = exec.CommandContext(ctx, "git", "clone", r.url, r.Path())
	} else {
		cmd = exec.CommandContext(ctx, "git", "clone", "-b", r.branch, r.url, r.Path())
	}
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))
	if err != nil {
		return err
	}
	return nil
}

// CopyTo copies the repository to project path.
func (r *Repo) CopyTo(ctx context.Context, to string, modPath string, ignores []string) error {
	if err := r.Clone(ctx); err != nil {
		return err
	}
	mod, err := ModulePath(filepath.Join(r.Path(), "go.mod"))
	if err != nil {
		return err
	}
	return copyDir(r.Path(), to, []string{mod, modPath}, ignores)
}

// CopyToV2 copies the repository to project path
func (r *Repo) CopyToV2(ctx context.Context, to string, modPath string, ignores, replaces []string) error {
	if err := r.Clone(ctx); err != nil {
		return err
	}
	mod, err := ModulePath(filepath.Join(r.Path(), "go.mod"))
	if err != nil {
		return err
	}
	replaces = append([]string{mod, modPath}, replaces...)
	return copyDir(r.Path(), to, replaces, ignores)
}

// ModulePath returns go module path.
func ModulePath(filename string) (string, error) {
	modBytes, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return modfile.ModulePath(modBytes), nil
}

// CopyLocalTemplate 把本地模板目录拷到项目路径——免 clone 的 Repo.CopyTo 等价物。
// 替换机制一致:读模板 go.mod 的 module 名,以 [模板module, modPath] 对做字节替换。
// 适合团队自持的脚手架模板目录(版本管理完全交给模板目录自身)。
func CopyLocalTemplate(srcDir, to, modPath string, ignores []string) error {
	abs, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("模板目录不存在: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("模板路径不是目录: %s", abs)
	}
	mod, err := ModulePath(filepath.Join(abs, "go.mod"))
	if err != nil {
		return fmt.Errorf("模板缺少可解析的 go.mod: %w", err)
	}
	return copyDir(abs, to, []string{mod, modPath}, ignores)
}

// CopyLocalTemplateV2 是 CopyLocalTemplate 的多替换对版本(等价 Repo.CopyToV2),
// 供 `gow add service` 的本地模板路径使用。
func CopyLocalTemplateV2(srcDir, to, modPath string, ignores, replaces []string) error {
	abs, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("模板目录不存在: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("模板路径不是目录: %s", abs)
	}
	mod, err := ModulePath(filepath.Join(abs, "go.mod"))
	if err != nil {
		return fmt.Errorf("模板缺少可解析的 go.mod: %w", err)
	}
	replaces = append([]string{mod, modPath}, replaces...)
	return copyDir(abs, to, replaces, ignores)
}
