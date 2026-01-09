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
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Define command line parameters
	envPath := flag.String("e", ".env", "Path to the .env file")
	workDir := flag.String("d", "", "Working directory for the program execution (chdir)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: env-run [-e .env] [-d ./dir] -- <command> [args...]\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Get the remaining arguments after -- as the command to execute
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No command given")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	// 2. Load environment variables (non-intrusive read)
	if _, err := os.Stat(*envPath); err == nil {
		err := godotenv.Load(*envPath)
		if err != nil {
			log.Fatalf("Error: Unable to parse env file %s: %v", *envPath, err)
		}
	} else {
		log.Printf("Info: Env file %s not found, skipping loading", *envPath)
	}

	// 3. Prepare to execute command
	cmdName := args[0]
	cmdArgs := args[1:]
	cmd := exec.Command(cmdName, cmdArgs...)

	// Set working directory
	if *workDir != "" {
		cmd.Dir = *workDir
	}

	// Inherit and inject environment variables
	cmd.Env = os.Environ()

	// Bind standard I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 4. Signal Forwarding
	// Ensure signals like Ctrl+C are passed to the child process
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for sig := range sigs {
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		}
	}()

	// 5. Start the program
	if err := cmd.Start(); err != nil {
		log.Fatalf("Error: Unable to start program: %v", err)
	}

	// Wait for program to finish and return correct exit code
	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
