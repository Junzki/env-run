/*
 *  Copyright (C) 2026 Andrija Junzki <andrew.junzki AT gmail.com>

 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
)

type stringArray []string

func (i *stringArray) String() string {
	return fmt.Sprint(*i)
}

func (i *stringArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	// 1. Define command line parameters
	var envPaths stringArray
	flag.Var(&envPaths, "e", "Path to the .env file (can be repeated, later overrides earlier)")
	workDir := flag.String("d", "", "Working directory for the program execution (chdir)")

	flag.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "Usage: env-run [-e .env] [-d ./dir] -- <command> [args...]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Get the remaining arguments after -- as the command to execute
	args := flag.Args()
	if len(args) == 0 {
		_, _ = fmt.Fprintln(os.Stderr, "Error: No command given")
		_, _ = fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	if len(envPaths) == 0 {
		envPaths = append(envPaths, ".env")
	}

	// 2. Load environment variables (non-intrusive read)

	// Snapshot original environment so we can restore it later.
	// This ensures shell variables take precedence over .env files,
	// while allowing later .env files to override earlier ones.
	originalEnv := os.Environ()

	for _, envPath := range envPaths {
		if _, err := os.Stat(envPath); err == nil {
			err := godotenv.Overload(envPath)
			if err != nil {
				log.Fatalf("Error: Unable to parse env file %s: %v", envPath, err)
			}
		} else {
			log.Printf("Info: Env file %s not found, skipping loading", envPath)
		}
	}

	// Restore original environment variables
	for _, envLine := range originalEnv {
		parts := strings.SplitN(envLine, "=", 2)
		if len(parts) == 2 {
			_ = os.Setenv(parts[0], parts[1])
		}
	}

	// 3. Execute command (Process Replacement)
	cmdName := args[0]
	cmdArgs := args

	// Find the absolute path of the command
	binary, err := exec.LookPath(cmdName)
	if err != nil {
		log.Fatalf("Error: Command not found: %s", cmdName)
	}
	// Make sure the binary path is absolute before changing directory
	binary, err = filepath.Abs(binary)
	if err != nil {
		log.Fatalf("Error: Unable to resolve absolute path of executable: %v", err)
	}

	// Set working directory if requested
	if *workDir != "" {
		if err := os.Chdir(*workDir); err != nil {
			log.Fatalf("Error: Unable to change directory to %s: %v", *workDir, err)
		}
	}

	// EXEC syscall replaces the current process with the new one.
	// We pass:
	// 1. The path to the binary
	// 2. The arguments (must include the command name as the first argument)
	// 3. The environment variables
	env := os.Environ()
	if err := syscall.Exec(binary, cmdArgs, env); err != nil {
		log.Fatalf("Error: Failed to execute command: %v", err)
	}
}
