// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosacli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/openshift/rosa/tests/utils/log"
)

const (
	defaultRunnerFormat = "text"
	jsonRunnerFormat    = "json"
	yamlRunnerFormat    = "yaml"

	redactedValue = "<redacted>"
)

// sensitiveKeys are the config keys and flag names whose value must never reach
// the logs. Matching uses the normalized form, so "client_secret",
// "--client-secret" and "--client-secret=value" all resolve to one entry.
var sensitiveKeys = []string{
	"accesstoken",
	"clientsecret",
	"refreshtoken",
	"token",
}

type runner struct {
	cmds      []string
	cmdArgs   []string
	envs      []string
	runnerCfg *runnerConfig
	dir       string
}

type runnerConfig struct {
	format string
	color  string
	debug  bool
}

func NewRunner() *runner {
	pwd, _ := os.Getwd()
	runner := &runner{
		runnerCfg: &runnerConfig{
			format: "text",
			debug:  false,
			color:  "auto",
		},
		envs: os.Environ(),
		dir:  pwd,
	}
	return runner
}

func (r *runner) Copy() *runner {
	return &runner{
		runnerCfg: r.runnerCfg.Copy(),
	}
}

func (rc *runnerConfig) Copy() *runnerConfig {
	return &runnerConfig{
		format: rc.format,
		color:  rc.color,
		debug:  rc.debug,
	}
}

func (r *runner) format(format string) *runner {
	r.runnerCfg.format = format
	return r
}

func (r *runner) Debug(debug bool) *runner {
	r.runnerCfg.debug = debug
	return r
}

func (r *runner) Color(color string) *runner {
	r.runnerCfg.color = color
	return r
}

func (r *runner) SetDir(dir string) *runner {
	r.dir = dir
	return r
}

func (r *runner) GetDir() string {
	return r.dir
}

func (r *runner) JsonFormat() *runner {
	return r.format(jsonRunnerFormat)
}

func (r *runner) YamlFormat() *runner {
	return r.format(yamlRunnerFormat)
}

func (r *runner) UnsetFormat() *runner {
	return r.format(defaultRunnerFormat)
}

func (r *runner) Cmd(commands ...string) *runner {
	r.cmds = commands
	return r
}

func (r *runner) CmdFlags(cmdFlags ...string) *runner {
	var cmdArgs []string
	cmdArgs = append(cmdArgs, cmdFlags...)
	r.cmdArgs = cmdArgs
	return r
}

func (r *runner) AddCmdFlags(cmdFlags ...string) *runner {
	cmdArgs := append(r.cmdArgs, cmdFlags...)
	r.cmdArgs = cmdArgs
	return r
}

func (r *runner) UnsetArgs() {
	r.cmdArgs = []string{}
}

func (r *runner) UnsetBoolFlag(flag string) *runner {
	var newCmdArgs []string
	cmdArgs := r.cmdArgs
	for _, vv := range cmdArgs {
		if vv == flag {
			continue
		}
		newCmdArgs = append(newCmdArgs, vv)
	}

	r.cmdArgs = newCmdArgs
	return r
}

func (r *runner) UnsetFlag(flag string) *runner {
	cmdArgs := r.cmdArgs
	flagIndex := 0
	for n, key := range cmdArgs {
		if key == flag {
			flagIndex = n
			break
		}
	}

	cmdArgs = append(cmdArgs[:flagIndex], cmdArgs[flagIndex+2:]...)
	r.cmdArgs = cmdArgs
	return r
}

func (r *runner) ReplaceFlag(flag string, value string) *runner {
	cmdArgs := r.cmdArgs
	for n, key := range cmdArgs {
		if key == flag {
			cmdArgs[n+1] = value
			break
		}
	}

	r.cmdArgs = cmdArgs
	return r
}

func (r *runner) AddEnvVar(key string, value string) *runner {
	env := fmt.Sprintf("%s=%s", key, value)
	r.envs = append(r.envs, env)
	return r
}

func (rc *runnerConfig) GenerateCmdFlags() (flags []string) {
	if rc.format == jsonRunnerFormat || rc.format == yamlRunnerFormat {
		flags = append(flags, "--output", rc.format)
	}
	if rc.debug {
		flags = append(flags, "--debug")
	}
	if rc.color != "auto" {
		flags = append(flags, "--color", rc.color)
	}
	return
}

func (r *runner) CmdElements() []string {
	cmdElements := r.cmds
	if len(r.cmdArgs) > 0 {
		cmdElements = append(cmdElements, r.cmdArgs...)
	}
	cmdElements = append(cmdElements, r.runnerCfg.GenerateCmdFlags()...)
	return cmdElements
}
func (r *runner) CMDString() string {
	return fmt.Sprintf("rosa %s", strings.Join(r.CmdElements(), " "))
}

// normalizeKey reduces a command element to a comparable key by dropping any
// inline value, leading dashes, and "-"/"_" separators, so the flag and
// positional spellings of a key match the same sensitiveKeys entry.
func normalizeKey(elem string) string {
	key, _, _ := strings.Cut(elem, "=")
	key = strings.TrimLeft(key, "-")
	return strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(key))
}

// redactedCMDString returns the command with every sensitive value replaced,
// keeping the remaining flags visible so a failed run stays debuggable.
// redactOutput reports whether the command's own output must be withheld, which
// is only true when a sensitive key is read back rather than assigned.
func (r *runner) redactedCMDString() (cmd string, redactOutput bool) {
	elements := slices.Clone(r.CmdElements())
	for i, elem := range elements {
		if !slices.Contains(sensitiveKeys, normalizeKey(elem)) {
			continue
		}
		if key, _, inline := strings.Cut(elem, "="); inline {
			elements[i] = key + "=" + redactedValue
			continue
		}
		// A following flag, or no following element, means the key is read.
		if i+1 >= len(elements) || strings.HasPrefix(elements[i+1], "-") {
			redactOutput = true
			continue
		}
		elements[i+1] = redactedValue
	}
	return fmt.Sprintf("rosa %s", strings.Join(elements, " ")), redactOutput
}

func (r *runner) run(combinedOutput bool) (bytes.Buffer, bytes.Buffer, error) {
	rosacmd := "rosa"
	cmdElements := r.CmdElements()
	cmdString, redactOutput := r.redactedCMDString()
	var output bytes.Buffer
	var stderr bytes.Buffer
	var err error
	retry := 0
	for {
		if retry > 4 {
			err = fmt.Errorf("executing failed: %s", output.String()+"\n"+stderr.String())
			return output, stderr, err
		}

		log.Logger.Infof("Running command: %s", cmdString)

		output.Reset()
		stderr.Reset()
		cmd := exec.Command(rosacmd, cmdElements...)
		cmd.Env = append(cmd.Env, r.envs...)
		cmd.Stdout = &output
		if combinedOutput {
			cmd.Stderr = cmd.Stdout
		} else {
			cmd.Stderr = &stderr
		}
		cmd.Dir = r.dir

		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("%s: %s", err.Error(), output.String()+"\n"+stderr.String())
		}
		if redactOutput {
			log.Logger.Warnf("Command reads a sensitive value. Hiding the output.")
		} else if combinedOutput {
			log.Logger.Infof("Get Combining Stdout and Stderr is :\n%s", output.String())
		} else {
			log.Logger.Infof("Get stdout is:\n%s", output.String())
			log.Logger.Infof("Get stderr is:\n%s", stderr.String())
		}

		tokenErr := "Not able to get authentication token"
		if strings.Contains(output.String(), tokenErr) || strings.Contains(stderr.String(), tokenErr) {
			retry = retry + 1
			log.Logger.Warnf("[Retry] Not able to get authentication token!! Wait and sleep 5s to do the %d retry", retry)
			time.Sleep(5 * time.Second)
			continue
		}
		return output, stderr, err
	}
}

func (r *runner) Run() (bytes.Buffer, error) {
	output, _, err := r.run(true)
	return output, err
}

func (r *runner) RunSeparateOutput() (bytes.Buffer, bytes.Buffer, error) {
	return r.run(false)
}

func (r *runner) RunCMD(command []string) (bytes.Buffer, error) {
	var output bytes.Buffer
	var err error
	log.Logger.Infof("%s command is running", command[0])
	output.Reset()
	cmd := exec.Command(command[0], command[1:]...) // #nosec G204
	cmd.Stdout = &output
	cmd.Stderr = cmd.Stdout
	cmd.Dir = r.dir

	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("%s: %s", err.Error(), output.String())
	}
	log.Logger.Debugf("Get Combining Stdout and Stderr is : %s", output.String())

	return output, err
}

// RunPipeline runs a pipeline of commands, each specified as a slice of strings.
func (r *runner) RunPipeline(commands ...[]string) (bytes.Buffer, error) {
	var output bytes.Buffer
	var err error

	cmds := make([]*exec.Cmd, len(commands))

	for i, command := range commands {
		cmds[i] = exec.Command(command[0], command[1:]...) // #nosec G204
		if i > 0 {
			cmds[i].Stdin, _ = cmds[i-1].StdoutPipe()
		}
		cmds[i].Stderr = &output

	}

	cmds[len(cmds)-1].Stdout = &output

	for _, cmd := range cmds {
		log.Logger.Infof("Running commands: %s", cmd.String())
		if err = cmd.Start(); err != nil {
			return output, fmt.Errorf("starting %v: %v", cmd.Args, err)
		}
	}

	for _, cmd := range cmds {
		if err = cmd.Wait(); err != nil {
			return output, fmt.Errorf("waiting for %v: %v", cmd.Args, err)
		}
	}
	return output, nil
}
