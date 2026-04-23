package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type menuItem struct {
	label  string
	inputs []menuInput
	run    func(inputs []string)
}

type menuInput struct {
	prompt       string
	defaultValue string
}

const menuWidth = 48

// ANSI color codes
const (
	colorReset      = "\033[0m"
	colorSelectedBg = "\033[44m" // blue background
	colorSelectedFg = "\033[97m" // bright white text
	colorBold       = "\033[1m"
)

func menuRow(text string, highlight bool) string {
	padded := fmt.Sprintf("%-*s", menuWidth, text)
	if highlight {
		return "│" + colorSelectedBg + colorSelectedFg + colorBold + " " + padded + " " + colorReset + "│"
	}
	return "│ " + padded + " │"
}

func menuDivider(left, right, line string) string {
	bar := strings.Repeat(line, menuWidth+2)
	return left + bar + right
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func printMenu(items []menuItem, selected int) {
	title := "ShellRelay"
	padding := (menuWidth - len(title)) / 2
	centeredTitle := fmt.Sprintf("%*s%s", padding, "", title)
	fmt.Println(menuDivider("╔", "╗", "═"))
	fmt.Println(menuRow(centeredTitle, false))
	fmt.Println(menuDivider("╠", "╣", "═"))
	for i, item := range items {
		isSelected := i == selected
		prefix := "     "
		if isSelected {
			prefix = "  ▶  "
		}
		fmt.Println(menuRow(fmt.Sprintf("%s%2d. %s", prefix, i+1, item.label), isSelected))
	}
	fmt.Println(menuDivider("╠", "╣", "═"))
	isSelected := selected == len(items)
	prefix := "     "
	if isSelected {
		prefix = "  ▶  "
	}
	fmt.Println(menuRow(fmt.Sprintf("%s %2d. %s", prefix, 0, "Exit"), isSelected))
	fmt.Println(menuDivider("╚", "╝", "═"))
	fmt.Println("\n  ↑↓ to move   Enter to select   q to quit")
}

// readKey reads a single keypress.
// If stdin is a real TTY, uses raw mode for arrow key support.
// Falls back to line-based input otherwise.
func readKey() string {
	fd := int(os.Stdin.Fd())

	// Not a TTY — fall back to plain line input
	if !term.IsTerminal(fd) {
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "quit"
		}
		line = strings.TrimSpace(line)
		if line == "0" || line == "q" || line == "Q" {
			return "quit"
		}
		return line
	}

	// Raw mode for real TTY
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "quit"
	}
	defer term.Restore(fd, oldState)

	buf := make([]byte, 3)
	n, err := os.Stdin.Read(buf)
	if err != nil || n == 0 {
		return "quit"
	}

	// Escape sequence: arrow keys are ESC [ A/B
	if n >= 3 && buf[0] == 0x1b && buf[1] == '[' {
		switch buf[2] {
		case 'A':
			return "up"
		case 'B':
			return "down"
		}
		return ""
	}

	switch buf[0] {
	case 13, 10: // Enter
		return "enter"
	case 'q', 'Q', 3, 4: // q, Ctrl+C, Ctrl+D
		return "quit"
	case '0':
		return "quit"
	}

	if buf[0] >= '1' && buf[0] <= '9' {
		return string(buf[:1])
	}

	return ""
}

func cmdMenu(_ []string) {
	items := []menuItem{
		{
			label: "Register server with Gmail account",
			inputs: []menuInput{
				{prompt: "Server ID", defaultValue: ""},
				{prompt: "Gmail address", defaultValue: ""},
				{prompt: "Display name (optional)", defaultValue: ""},
			},
			run: func(inputs []string) {
				if inputs[0] == "" {
					fmt.Println("Error: server ID is required")
					return
				}
				if inputs[1] == "" {
					fmt.Println("Error: Gmail address is required")
					return
				}
				// flags must come before positional arg for Go's flag package
				args := []string{"--email", inputs[1]}
				if inputs[2] != "" {
					args = append(args, "--name", inputs[2])
				}
				args = append(args, inputs[0])
				cmdAnnounce(args)
			},
		},
		{
			label: "Start daemon",
			inputs: []menuInput{
				{prompt: "Server ID", defaultValue: ""},
				{prompt: "Token (sr_...)", defaultValue: ""},
			},
			run: func(inputs []string) {
				args := []string{}
				if inputs[0] != "" {
					args = append(args, inputs[0])
				}
				if inputs[1] != "" {
					args = append(args, inputs[1])
				}
				cmdStart(args)
			},
		},
		{
			label:  "Stop daemon",
			inputs: nil,
			run:    func(_ []string) { cmdStop(nil) },
		},
		{
			label:  "Restart daemon",
			inputs: nil,
			run:    func(_ []string) { cmdRestart(nil) },
		},
		{
			label:  "Status",
			inputs: nil,
			run:    func(_ []string) { cmdStatus(nil) },
		},
		{
			label: "Logs",
			inputs: []menuInput{
				{prompt: "Number of lines", defaultValue: "50"},
			},
			run: func(inputs []string) {
				cmdLogs([]string{"-n", inputs[0]})
			},
		},
		{
			label:  "Sessions",
			inputs: nil,
			run:    func(_ []string) { cmdSessions(nil) },
		},
		{
			label: "Rotate token",
			inputs: []menuInput{
				{prompt: "New token (sr_...)", defaultValue: ""},
			},
			run: func(inputs []string) {
				if inputs[0] == "" {
					fmt.Println("Error: token is required")
					return
				}
				cmdRotate([]string{"--token", inputs[0]})
			},
		},
		{
			label: "Set relay URL",
			inputs: []menuInput{
				{prompt: "Relay URL (hostname or wss://...)", defaultValue: ""},
			},
			run: func(inputs []string) {
				if inputs[0] == "" {
					fmt.Println("Error: URL is required")
					return
				}
				cmdRelay([]string{"--url", inputs[0]})
			},
		},
		{
			label:  "Upgrade to latest release",
			inputs: nil,
			run:    func(_ []string) { cmdUpgrade(nil) },
		},
		{
			label:  "Daemon install",
			inputs: nil,
			run:    func(_ []string) { cmdDaemon([]string{"install"}) },
		},
		{
			label:  "Daemon uninstall",
			inputs: nil,
			run:    func(_ []string) { cmdDaemon([]string{"uninstall"}) },
		},
		{
			label:  "Version",
			inputs: nil,
			run:    func(_ []string) { fmt.Printf("shellrelay v%s\n", Version) },
		},
	}

	selected := 0
	total := len(items) + 1 // items + Exit

	for {
		clearScreen()
		printMenu(items, selected)

		key := readKey()

		switch key {
		case "up":
			selected = (selected - 1 + total) % total
		case "down":
			selected = (selected + 1) % total
		case "quit":
			clearScreen()
			fmt.Println("Bye.")
			return
		case "enter":
			if selected == len(items) {
				clearScreen()
				fmt.Println("Bye.")
				return
			}
			runMenuItem(items[selected])
		default:
			// Direct number key 1-9
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				idx := int(key[0]-'0') - 1
				if idx < len(items) {
					selected = idx
					runMenuItem(items[selected])
				}
			}
		}
	}
}

func runMenuItem(item menuItem) {
	clearScreen()
	fmt.Printf("→ %s\n\n", item.label)

	reader := bufio.NewReader(os.Stdin)
	collected := make([]string, len(item.inputs))
	for i, inp := range item.inputs {
		if inp.defaultValue != "" {
			fmt.Printf("  %s [%s]: ", inp.prompt, inp.defaultValue)
		} else {
			fmt.Printf("  %s: ", inp.prompt)
		}
		val, _ := reader.ReadString('\n')
		val = strings.TrimSpace(val)
		if val == "" {
			val = inp.defaultValue
		}
		collected[i] = val
	}

	if len(item.inputs) > 0 {
		fmt.Println()
	}

	item.run(collected)

	fmt.Println("\n--- press Enter to continue ---")
	// Drain any leftover bytes then wait for Enter
	buf := make([]byte, 64)
	for {
		n, _ := os.Stdin.Read(buf)
		if n > 0 {
			for _, b := range buf[:n] {
				if b == '\n' || b == '\r' {
					return
				}
			}
		}
	}
}
