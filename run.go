package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

func runCmd(logOutput io.Writer, printCmd bool, workDir string, command string, args ...string) error {
	fmt.Fprintln(logOutput, "--- Running Command ---")

	var cmd *exec.Cmd
	var shell string
	var shellArgs []string

	if runtime.GOOS == "windows" {
		shell = "powershell"                                              // or "cmd"
		shellArgs = []string{"-NoProfile", "-NonInteractive", "-Command"} // PowerShell args
		fullCmdStr := fmt.Sprintf("%s %s", command, strings.Join(args, " "))
		cmd = exec.Command(shell, append(shellArgs, fullCmdStr)...)

		if printCmd {
			fmt.Fprintf(logOutput, "Shell: %s\nArgs: %s \"%s\"\nDirectory: %s\n", shell, strings.Join(shellArgs, " "), fullCmdStr, workDir)
		}

	} else { // Unix-like
		shell = os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/sh" // or "/bin/bash"
			fmt.Fprintf(logOutput, "SHELL env var not set, defaulting to: %s\n", shell)
		}
		shellArgs = []string{"-l", "-c"}
		fullCmdStr := command
		if len(args) > 0 {
			quotedArgs := make([]string, len(args))
			for i, arg := range args {
				if strings.Contains(arg, " ") {
					quotedArgs[i] = fmt.Sprintf("'%s'", arg)
				} else {
					quotedArgs[i] = arg
				}
			}
			fullCmdStr += " " + strings.Join(quotedArgs, " ")
		}

		cmd = exec.Command(shell, append(shellArgs, fullCmdStr)...)

		if printCmd {
			fmt.Fprintf(logOutput, "Shell: %s\nArgs: %s \"%s\"\nDirectory: %s\n", shell, strings.Join(shellArgs, " "), fullCmdStr, workDir)
		}
	}

	if workDir != "" {
		cmd.Dir = workDir
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(logOutput, "Error creating stdout pipe: %v\n", err)
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(logOutput, "Error creating stderr pipe: %v\n", err)
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			fmt.Fprintln(logOutput, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(logOutput, "Error reading stdout: %v\n", err)
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			fmt.Fprintln(logOutput, "ERR: "+scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(logOutput, "Error reading stderr: %v\n", err)
		}
	}()

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(logOutput, "Error starting command: %v\n", err)
		return err
	}

	wg.Wait()

	err = cmd.Wait()
	fmt.Fprintf(logOutput, "--- Command Finished (Exit Code: %d) ---\n", cmd.ProcessState.ExitCode())
	return err
}
