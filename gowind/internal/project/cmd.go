package project

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/tx7do/go-wind-toolkit/gowind/internal/buf"
	"github.com/tx7do/go-wind-toolkit/gowind/internal/pkg"
)

// CmdProject represents the project command.
var CmdProject = &cobra.Command{
	Use:     "project [name]",
	Aliases: []string{"proj"},
	Short:   "create a new project scaffold",
	Long:    "Create a project using the repository template. Example: gow new project helloworld",
	Args:    cobra.ExactArgs(1),
	Run:     run,
}

var (
	repoURL    string
	branch     string
	timeout    string
	moduleName string
	nomod      bool
)

const (
	GithubRepoURL = "https://github.com/tx7do/go-wind-admin-template.git"
	GiteeRepoURL  = "https://gitee.com/tx7do/go-wind-admin-template.git"
)

func canReach(addr string, d time.Duration) bool {
	conn, err := net.DialTimeout("tcp", addr, d)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func init() {
	timeout = "60s"

	// 优先使用 GitHub，若不可达则回退到 Gitee
	if canReach("github.com:443", 3*time.Second) {
		repoURL = GithubRepoURL
	} else {
		repoURL = GiteeRepoURL
	}

	CmdProject.Flags().StringVarP(&repoURL, "repo-url", "r", repoURL, "layout repo")
	CmdProject.Flags().StringVarP(&branch, "branch", "b", branch, "repo branch")
	CmdProject.Flags().StringVarP(&timeout, "timeout", "t", timeout, "time out")
	CmdProject.Flags().StringVarP(&moduleName, "module", "m", moduleName, "set go module name, if not set, use project name")
	CmdProject.Flags().BoolVarP(&nomod, "nomod", "", nomod, "retain go mod")
}

func run(cmd *cobra.Command, args []string) {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	t, err := time.ParseDuration(timeout)
	if err != nil {
		panic(err)
	}

	parentCtx := cmd.Context()
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, t)
	defer cancel()

	name := ""
	if len(args) == 0 {
		prompt := &survey.Input{
			Message: "What is project name ?",
			Help:    "Created project name.",
		}
		err = survey.AskOne(prompt, &name)
		if err != nil || name == "" {
			return
		}
	} else {
		name = args[0]
	}

	projectName, workingDir := processProjectParams(name, wd)

	if isDirExists(workingDir, projectName) {
		fmt.Printf("🚫 %s already exists\n", projectName)
		prompt := &survey.Confirm{
			Message: "📂 Do you want to override the folder ?",
			Help:    "Delete the existing folder and create the project.",
		}
		var override bool
		e := survey.AskOne(prompt, &override)
		if e != nil {
			return
		}
		if !override {
			return
		}
		_ = os.RemoveAll(filepath.Join(workingDir, projectName))
	}

	fmt.Printf("🚀 Creating service %s, layout repo is %s, please wait a moment.\n\n", projectName, repoURL)

	p := &Project{
		Name:   projectName,
		Module: projectName,
	}
	if moduleName != "" {
		p.Module = moduleName
	}

	done := make(chan error, 1)
	go func() {
		if !nomod {
			done <- p.New(ctx, workingDir, repoURL, branch)
			return
		}
		projectRoot := getGoModProjectRoot(workingDir)
		if goModIsNotExistIn(projectRoot) {
			done <- fmt.Errorf("🚫 go.mod don't exists in %s", projectRoot)
			return
		}

		packagePath, e := filepath.Rel(projectRoot, filepath.Join(workingDir, projectName))
		if e != nil {
			done <- fmt.Errorf("🚫 failed to get relative path: %v", err)
			return
		}
		packagePath = strings.ReplaceAll(packagePath, "\\", "/")

		mod, e := pkg.ModulePath(filepath.Join(projectRoot, "go.mod"))
		if e != nil {
			done <- fmt.Errorf("🚫 failed to parse `go.mod`: %v", e)
			return
		}
		// Get the relative path for adding a project based on Go modules
		p.Path = filepath.Join(strings.TrimPrefix(workingDir, projectRoot+"/"), p.Name)
		done <- p.Add(ctx, workingDir, repoURL, branch, mod, packagePath)
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			_, _ = fmt.Fprint(os.Stderr, "\033[31mERROR: project creation timed out\033[m\n")
			return
		}
		_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: failed to create project(%s)\033[m\n", ctx.Err().Error())

	case err = <-done:
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: Failed to create project(%s)\033[m\n", err.Error())
		}

		if err = pkg.GoModTidy(ctx, filepath.Join(workingDir, projectName)); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "\033[31mERROR: failed to run `go mod tidy`: %s\033[m\n", err.Error())
			return
		}

		if err = buf.GenerateFromPath(ctx, filepath.Join(workingDir, projectName, "api")); err != nil {
			return
		}

		fmt.Printf("✅ Project %s created successfully at %s\n", projectName, filepath.Join(workingDir, projectName))
	}
}
