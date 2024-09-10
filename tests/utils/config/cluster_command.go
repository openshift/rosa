package config

import (
	"fmt"
	"os"
	"strings"
)

type Command interface {
	GetFullCommand() string
	GetFlagValue(flag string, flagWithValue bool) string
	AddFlags(flags ...string)
	ReplaceFlagValue(flags map[string]string)
	DeleteFlag(flag string, flagWithValue bool) error
	CheckFlagExist(flag string) bool
}

type command struct {
	cmd string
}

func GenerateCommand(cmdString string) Command {
	return &command{
		cmd: cmdString,
	}
}

// Get the rosa command for creating cluster from ${SHARED_DIR}/create_cluster.sh
func RetrieveClusterCreationCommand(filePath string) (Command, error) {
	fileContents, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	cmd := &command{
		cmd: strings.Trim(string(fileContents), "\n"),
	}
	return cmd, nil
}

func (c *command) GetFullCommand() string {
	return c.cmd
}

// a function to replace any flag in the command with the key-value map passed to the function
func (c *command) ReplaceFlagValue(flags map[string]string) {
	elements := strings.Split(c.cmd, " ")
	for i, e := range elements {
		if value, ok := flags[e]; ok {
			elements[i+1] = value
		}
	}
	c.cmd = strings.Join(elements, " ")
}

// a function to delete any flag in the command
func (c *command) DeleteFlag(flag string, flagWithValue bool) error {
	elements := strings.Split(c.cmd, " ")
	for i, e := range elements {
		if e == flag {
			if flagWithValue {
				elements = append(elements[:i], elements[i+2:]...)
			} else {
				elements = append(elements[:i], elements[i+1:]...)
			}
			c.cmd = strings.Join(elements, " ")
			return nil
		}
	}
	return fmt.Errorf("cannot find flag %s in command %s", flag, c.cmd)
}

// Get the value of a flag from the command
func (c *command) GetFlagValue(flag string, flagWithValue bool) string {
	elements := strings.Split(c.cmd, " ")
	for i, e := range elements {
		if e == flag {
			if flagWithValue {
				return elements[i+1]
			} else {
				return ""
			}
		}
	}
	return ""
}

// Add flags to the command
func (c *command) AddFlags(flags ...string) {
	for _, flag := range flags {
		c.cmd = strings.TrimSpace(c.cmd)
		// combine the command with space
		c.cmd += " " + flag
	}
}

// Check flag exist
func (c *command) CheckFlagExist(flag string) bool {
	elements := strings.Split(c.cmd, " ")
	for _, e := range elements {
		if e == flag {
			return true
		}
	}
	return false
}
