package main

import (
	"fmt"
	"os"
	"path"
)

func fatalf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
	os.Exit(1)
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	var config_file string
	var err error

	checkRequirements()

	parsed, err := parseCommands(os.Args[1:])
	if err != nil {
		fatalf("%s\n", err)
	}

	if parsed.cmd == "create" {
		config_file = path.Join("config", parsed.options["config"].(string)+".yaml")
	} else {
		if file_exists(path.Join("instances", parsed.id)) {
			config_file = path.Join("instances", parsed.id, "config.yaml")
		} else {
			fatalf("No instance named %s", parsed.id)
		}
	}

	inst, err := buildInstance(parsed.id, config_file)
	if err != nil {
		fatalf("Error : %v", err)
	}

	switch parsed.cmd {
	case "create":
		err = inst.Create(parsed.options["force"].(bool))
	case "start":
		err = inst.Start(parsed.options["verbose"].(bool))
	case "stop":
		err = inst.Stop()
	case "status":
		inst.Status()
	case "destroy":
		err = inst.Destroy()
	case "protect":
		err = inst.Protect()
	case "shell":
		err = inst.Shell()
	case "resize":
		err = inst.Resize()
	}
	if err != nil {
		fmt.Println(err)
	}
}
