package main

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"
)

const ( // can be ORed
	CommandAsRoot = 1
	CommandAsUser = 2
)

type CommandOption struct {
	mandatory bool
	value     any
	dfault    any
}

type Command struct {
	run_as  int                      // privileges to run this command
	options map[string]CommandOption // option flags and defaults
}

type ParsedCommand struct { // result to the caller
	cmd     string
	id      string
	options map[string]any
}

// Parse args and returns a ParsedCommand with options (if any) or nil if something went wrong
func parseCommands(args []string) (*ParsedCommand, error) {
	cmds := map[string]Command{
		"create": {
			run_as: CommandAsUser,
			options: map[string]CommandOption{
				"--config": {
					mandatory: true,
					value:     nil,
					dfault:    "",
				},
				"--force": {
					mandatory: false,
					value:     nil,
					dfault:    false,
				},
			},
		},
		"start": {
			run_as: CommandAsRoot,
			options: map[string]CommandOption{
				"--verbose": {
					mandatory: false,
					value:     nil,
					dfault:    false,
				},
			},
		},
		"shell": {
			run_as:  CommandAsUser,
			options: map[string]CommandOption{},
		},
		"stop": {
			run_as:  CommandAsRoot,
			options: map[string]CommandOption{},
		},
		"destroy": {
			run_as:  CommandAsUser,
			options: map[string]CommandOption{},
		},
		"protect": {
			run_as:  CommandAsUser,
			options: map[string]CommandOption{},
		},
		"status": {
			run_as:  CommandAsRoot | CommandAsUser,
			options: map[string]CommandOption{},
		},
		"resize": {
			run_as: CommandAsUser,
			options: map[string]CommandOption{
				"--size": {
					mandatory: false,
					value:     nil,
					dfault:    80,
				},
			},
		},
	}

	if len(args) < 2 {
		return nil, fmt.Errorf("Not enough arguments")
	}

	argCmd := args[0] // command as a string, to be used later
	var cmd Command
	var ok bool
	var err error

	if cmd, ok = cmds[argCmd]; !ok {
		return nil, fmt.Errorf("Unknown command [%s]", argCmd)
	}
	cmd = cmds[argCmd] // command as a Command to get the related options

	currentUser, _ := user.Current()
	if currentUser.Uid == "0" && cmd.run_as&CommandAsRoot != 1 {
		return nil, fmt.Errorf("sudo is prohibited for command %s", argCmd)
	}
	if currentUser.Uid != "0" && cmd.run_as&CommandAsUser != 2 {
		return nil, fmt.Errorf("sudo is mandatory for command %s", argCmd)
	}

	id := args[1]                  // there's always an ID
	for _, arg := range args[2:] { // Parse above cmd and id
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("option need to start with --")
		}

		splitted := strings.Split(arg, "=") // an option might be foo=bar
		option_name := splitted[0]

		var val CommandOption
		if val, ok = cmd.options[option_name]; !ok { // is this option allowed for this command ?
			return nil, fmt.Errorf("Unexpected option [%s] for command [%s]", option_name, argCmd)
		}
		option := cmd.options[option_name]

		switch val.dfault.(type) {
		case bool:
			option.value = true

		case int:
			if len(splitted) > 1 { // foo=42 is mandatory for ints
				option.value, err = strconv.Atoi(splitted[1])
				if err != nil {
					return nil, fmt.Errorf("Can't parse value [%s] for option [%s]", splitted[1], option_name)
				}
			} else {
				return nil, fmt.Errorf("Missing value for option [%s]", option_name)
			}

		case string:
			if len(splitted) > 1 { // foo=bar is mandatory for strings
				option.value = splitted[1]
			} else {
				return nil, fmt.Errorf("Missing value for option [%s]", option_name)
			}

		default:
			return nil, fmt.Errorf("Unhandled type [%t] for option [%s]", val.dfault, option_name)
		}
		cmd.options[option_name] = option
	}

	options := make(map[string]any)
	for option_name, opt := range cmd.options {
		if opt.value == nil {
			if opt.mandatory { // option been left uninitialized but is mandatory
				return nil, fmt.Errorf("Mandatory option [%s]", option_name)
			}
			opt.value = opt.dfault // not a mandatory option, left uninitialized so let's fallback to default value
		}
		option_name = strings.TrimPrefix(option_name, "--")
		options[option_name] = opt.value
	}

	return &ParsedCommand{
		cmd:     argCmd,
		id:      id,
		options: options,
	}, nil
}
