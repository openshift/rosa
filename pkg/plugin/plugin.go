package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var SupportedBinaries = []string{"aws", "ocm"}
var SupportedPrefixes = []string{"rosa"}

type Plugin struct {
	Name string
	Path string
}

type Handler interface {
	HandlePluginCommand(cmdArgs []string) (found bool, err error)
	FindPlugins() (result []Plugin, err error)
}

type DefaultPluginSpec struct {
	ValidPrefixes []string
	ValidBinaries []string
}

func NewDefaultPluginHandler() Handler {
	return &DefaultHandler{
		validPrefixes: SupportedPrefixes,
		validBinaries: SupportedBinaries,
	}
}

var _ Handler = DefaultHandler{}

type DefaultHandler struct {
	validPrefixes []string
	validBinaries []string
}

func (d DefaultHandler) HandlePluginCommand(cmdArgs []string) (found bool, err error) {
	if len(cmdArgs) == 0 {
		return false, nil
	}

	foundBinaryPath, found := d.lookup(cmdArgs[0])
	if !found || len(foundBinaryPath) == 0 {
		return false, nil
	}

	if err := d.execute(foundBinaryPath, cmdArgs[1:], os.Environ()); err != nil {
		return true, err
	}

	return true, nil
}

func (d DefaultHandler) lookup(filename string) (string, bool) {
	// check for valid binary matches first
	for _, binary := range d.validBinaries {
		if filename == binary {
			path, err := exec.LookPath(filename)
			if err != nil || len(path) == 0 {
				continue
			}
			return path, true
		}
	}
	// if not valid binary matches check for valid prefixes
	for _, prefix := range d.validPrefixes {
		path, err := exec.LookPath(fmt.Sprintf("%s-%s", prefix, filename))
		if err != nil || len(path) == 0 {
			continue
		}
		return path, true
	}
	return "", false
}

func (d DefaultHandler) execute(executablePath string, cmdArgs []string, env []string) error {
	cmd := exec.Command(executablePath, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = env
	return cmd.Run()
}

func (d DefaultHandler) FindPlugins() (result []Plugin, err error) {
	defaultPath := filepath.SplitList(os.Getenv("PATH"))
	newPath := uniquePath(defaultPath)

	for _, dir := range newPath {
		_, err = os.Stat(dir)
		if os.IsNotExist(err) {
			err = nil
			continue
		}
		if err != nil {
			return
		}
		var list []Plugin
		list, err = d.listPlugins(dir)
		if err != nil {
			return
		}
		result = append(result, list...)
	}
	return
}

func (d DefaultHandler) listPlugins(dir string) (result []Plugin, err error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, item := range items {
		if item.IsDir() {
			continue
		}
		name := item.Name()
		if !d.matchBinaryName(name) {
			continue
		}
		path := filepath.Join(dir, name)
		var executable bool
		executable, err = isExecutable(path)
		if err != nil {
			return
		}
		if !executable {
			fmt.Printf("Warning: %s identified as an ROSA plugin, but it is not executable.\n", path)
		}
		if runtime.GOOS == "windows" {
			name = strings.TrimSuffix(name, ".exe")
		}
		plugin := Plugin{
			Name: name,
			Path: dir,
		}
		result = append(result, plugin)
	}
	return
}

func (d DefaultHandler) matchBinaryName(name string) bool {
	for _, binary := range d.validBinaries {
		if binary == name {
			return true
		}
	}
	for _, prefix := range d.validPrefixes {
		if strings.HasPrefix(name, fmt.Sprintf("%s-", prefix)) {
			return true
		}
	}
	return false
}
