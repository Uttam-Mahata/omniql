package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Uttam-Mahata/omniql/pkg/core"
	"github.com/peterh/liner"
)

const historyFile = ".omniql_history"

// runREPL starts an interactive REPL session backed by the given engine.
func runREPL(engine *core.Engine) {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	// Load history from ~/.omniql_history
	histPath := historyPath()
	if f, err := os.Open(histPath); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	defer func() {
		// Persist history on exit.
		if f, err := os.Create(histPath); err == nil {
			line.WriteHistory(f)
			f.Close()
		}
	}()

	fmt.Fprintln(os.Stdout, "OmniQL interactive shell. Type .help for commands, .exit to quit.")

	var accumulated strings.Builder

	for {
		prompt := "omniql> "
		if accumulated.Len() > 0 {
			prompt = "...> "
		}

		input, err := line.Prompt(prompt)
		if err != nil {
			// EOF or Ctrl-C
			fmt.Fprintln(os.Stdout)
			break
		}

		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			continue
		}

		line.AppendHistory(trimmed)

		// Handle dot-commands
		if accumulated.Len() == 0 && strings.HasPrefix(trimmed, ".") {
			if handleDotCommand(trimmed, engine) {
				break // .exit
			}
			continue
		}

		// Accumulate multi-line JSON
		accumulated.WriteString(trimmed)

		// Execute when line ends with } (best-effort heuristic)
		if strings.HasSuffix(trimmed, "}") {
			queryStr := accumulated.String()
			accumulated.Reset()
			executeREPLQuery(engine, queryStr)
		}
	}
}

// handleDotCommand processes a dot-command. Returns true if the REPL should exit.
func handleDotCommand(cmd string, engine *core.Engine) bool {
	switch strings.ToLower(cmd) {
	case ".exit", ".quit":
		fmt.Fprintln(os.Stdout, "Bye!")
		return true

	case ".help":
		fmt.Fprintln(os.Stdout, "Dot commands:")
		fmt.Fprintln(os.Stdout, "  .help     Show this help message")
		fmt.Fprintln(os.Stdout, "  .drivers  List registered drivers")
		fmt.Fprintln(os.Stdout, "  .routes   List target→driver route mappings")
		fmt.Fprintln(os.Stdout, "  .exit     Exit the shell")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Enter OQL JSON to execute (multi-line until closing '}')")
		fmt.Fprintln(os.Stdout, `Example: {"target":"users","action":"FIND","filter":{}}`)

	case ".drivers":
		drivers := engine.Drivers()
		sort.Strings(drivers)
		if len(drivers) == 0 {
			fmt.Fprintln(os.Stdout, "(no drivers registered)")
		} else {
			for _, d := range drivers {
				fmt.Fprintln(os.Stdout, " -", d)
			}
		}

	case ".routes":
		routes := engine.Routes()
		if len(routes) == 0 {
			fmt.Fprintln(os.Stdout, "(no explicit routes)")
		} else {
			// Sort targets for determinism
			targets := make([]string, 0, len(routes))
			for t := range routes {
				targets = append(targets, t)
			}
			sort.Strings(targets)
			for _, t := range targets {
				fmt.Fprintf(os.Stdout, "  %s → %s\n", t, routes[t])
			}
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q. Type .help for available commands.\n", cmd)
	}
	return false
}

// executeREPLQuery parses and executes an OQL JSON string, printing the result.
func executeREPLQuery(engine *core.Engine, queryStr string) {
	var query core.OQLQuery
	if err := json.Unmarshal([]byte(queryStr), &query); err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		return
	}

	result, err := engine.Execute(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
		return
	}

	if result.Error != nil {
		fmt.Fprintf(os.Stderr, "Error [%s]: %s\n", result.Error.Code, result.Error.Message)
		return
	}

	out, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encode error: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stdout, string(out))
}

// historyPath returns the path to the history file.
func historyPath() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, historyFile)
	}
	return historyFile
}
