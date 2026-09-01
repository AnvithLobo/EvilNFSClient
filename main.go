package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/AnvithLobo/EvilNFSClient/pkg/nfs"
	"github.com/AnvithLobo/EvilNFSClient/pkg/ui"
	"github.com/AnvithLobo/EvilNFSClient/pkg/ui/styles"
	nfslib "github.com/AnvithLobo/nfsv3/nfs"
	argparse "github.com/akamensky/argparse"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {

	// preliminary arg validation
	checkValidArgs()

	parser := argparse.NewParser("evilnfscommand", "Connect to an NFS export and optionally run commands or a TUI")

	parser.HelpFunc = func(c *argparse.Command, msg interface{}) string {
		printUsage(false)
		return ""
	}

	listFlag := parser.Flag("l", "list", &argparse.Options{Help: "List NFS exports on the given server and exit"})
	uidFlag := parser.Int("u", "uid", &argparse.Options{Help: "UID to use for the connection (overrides default)"})
	gidFlag := parser.Int("g", "gid", &argparse.Options{Help: "GID to use for the connection (overrides default)"})
	cmdFlag := parser.String("c", "cmd", &argparse.Options{Help: "Non-interactive command to run"})
	helpFlag := parser.Flag("h", "help", &argparse.Options{Help: "Show help"})
	privPort := parser.Flag("p", "privport", &argparse.Options{Help: "Use privileged port for NFS connection (may require root)"})

	// Positional args: server (required), export (optional if --list)
	serverPos := parser.StringPositional(&argparse.Options{Required: true, Help: "NFS server IP/hostname"})
	exportPos := parser.StringPositional(&argparse.Options{Required: false, Help: "Export path on the NFS server"})

	if err := parser.Parse(os.Args); err != nil {
		// Print usage with error context
		printUsage(true)
		fmt.Fprintln(os.Stderr, styles.ErrorStyle.Render(err.Error()))
		os.Exit(2)
	}

	if *helpFlag {
		printUsage(false)
		return
	}

	// Check if the server argument is provided
	if serverPos == nil || *serverPos == "" {
		printUsage(true)
		fmt.Fprintln(os.Stderr, styles.ErrorStyle.Render("Missing server argument."))
		fmt.Fprintln(os.Stderr)
		os.Exit(2)
	}

	server := *serverPos

	// If user asked to list exports, do that and exit
	if *listFlag {
		listNFSExports(server)
		os.Exit(0)
	}

	// For normal mode we need an export argument
	if exportPos == nil || *exportPos == "" {
		printUsage(true)
		fmt.Fprintln(os.Stderr, styles.ErrorStyle.Render("Missing export path."))
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
	export := *exportPos

	var uid uint32 = uint32(os.Getuid())
	var gid uint32 = uint32(os.Getgid())

	if uidFlag != nil {
		uid = uint32(*uidFlag)
	}
	if gidFlag != nil {
		gid = uint32(*gidFlag)
	}

	var command string
	if cmdFlag != nil {
		command = *cmdFlag
	}

	// Initialize NFS client
	client, err := nfs.NewNFSClient(server, export, uid, gid, *privPort)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if strings.Contains(err.Error(), "MNT3ERR_ACCES") && !*privPort {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, styles.HelpArgStyle.Render("Hint:")+" "+styles.HelpDescStyle.Render("The server requires a privileged source port (< 1024)."))
			fmt.Fprintln(os.Stderr, styles.HelpDescStyle.Render("      This is the NFS 'secure' export option — only root can bind to ports below 1024."))
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, styles.HelpDescStyle.Render("  Try running with the -p / --privport flag (requires root or CAP_NET_BIND_SERVICE):"))
			fmt.Fprintln(os.Stderr, styles.ExamplesSmallStyle.Render(fmt.Sprintf("    sudo evilnfsclient %s %s -p", server, export)))
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, styles.HelpDescStyle.Render("  Or grant the capability once to avoid sudo:"))
			fmt.Fprintln(os.Stderr, styles.ExamplesSmallStyle.Render("    sudo setcap 'cap_net_bind_service=+eip' $(which evilnfsclient)"))
		}
		os.Exit(1)
	}

	// Non-interactive mode
	if command != "" {
		// Inject a stderr progress reporter so the user sees activity
		trimmedCmd := strings.TrimSpace(command)
		arrow := "⬇"
		if strings.HasPrefix(trimmedCmd, "put") || strings.HasPrefix(trimmedCmd, "mput") {
			arrow = "⬆"
		}
		labelWidth, barWidth := nfs.ProgressLayout(nfs.TermWidth())
		client.SetProgressFunc(func(u nfs.ProgressUpdate) {
			bar := nfs.RenderFileProgress(u, labelWidth, barWidth)
			fmt.Fprintf(os.Stderr, "\r%s %s", arrow, bar)
		})

		output := client.ExecuteCommand(command)

		// Clear the progress line from stderr before printing results
		client.SetProgressFunc(nil)
		fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))

		for _, line := range output {
			fmt.Println(line)
		}
		return
	}

	// Interactive TUI mode
	m := ui.InitialModel(client)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print buffered output after TUI closes
	if finalModelCast, ok := finalModel.(ui.TUIModel); ok {
		fmt.Println(styles.TitleStyle.Render("🔥 EvilNFSClient"))
		fmt.Println(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf(
			"Connected to %s:%s (UID: %d, GID: %d) | Path: %s",
			finalModelCast.Client.Server,
			finalModelCast.Client.Export,
			finalModelCast.Client.UID,
			finalModelCast.Client.GID,
			finalModelCast.Client.CurrentPath,
		)))
		fmt.Println()

		for _, line := range finalModelCast.Output {
			fmt.Println(line)
		}

		fmt.Println(styles.SuccessStyle.Render("Goodbye!"))
	}
}

func listNFSExports(server string) {

	// Fetch exports
	fmt.Printf("Fetching exports from %s...\n\n", server)

	fmt.Println(styles.HelpTitleStyle.Render("🔥 EvilNFSClient"))

	exports, err := nfslib.GetExports(server)
	if err != nil {
		log.Fatalf("Error listing exports: %v", err)
	}

	// Header/title inside the box
	header := styles.HelpSectionStyle.Render("EXPORTS")

	// Build table header (colored)
	leftColTitle := styles.HelpArgStyle.Render(fmt.Sprintf("%-40s", "EXPORT"))
	rightColTitle := styles.HelpOptStyle.Render("ALLOWED CLIENTS")
	tableLines := []string{leftColTitle + " " + rightColTitle}
	tableLines = append(tableLines, strings.Repeat("─", 80-4)) // separator

	// Build rows
	for _, e := range exports {
		clients := "*"
		if len(e.Groups) > 0 {
			clients = fmt.Sprintf("%v", e.Groups)
		}

		// style columns: export path (DirStyle), clients (HelpDescStyle)
		left := styles.DirStyle.Render(fmt.Sprintf("%-40s", e.Dir))
		right := styles.HelpDescStyle.Render(clients)

		tableLines = append(tableLines, left+" "+right)
	}

	tableContent := lipgloss.JoinVertical(lipgloss.Left, tableLines...)

	// Compose box content with the header at top (gives a titled-box appearance)
	boxContent := header + "\n\n" + tableContent

	// Render full-width box (minus 2 for border glyphs)
	box := styles.BoxStyle.
		BorderForeground(lipgloss.Color("#4B9BFF")).
		Width(80).
		Render(boxContent)

	fmt.Println(box)
	fmt.Println() // trailing newline
}

func printUsage(short bool) {
	fmt.Print("\n")
	fmt.Println(styles.HelpTitleStyle.Render("🔥 EvilNFSClient"))
	fmt.Println(styles.HelpDescStyle.Render("Modern NFS mount browser and file manager"))
	fmt.Print("\n")

	termWidth := styles.TerminalWidth()

	if termWidth > 80 {
		termWidth = 80
	}

	// ----------------
	// USAGE (boxed)
	// ----------------

	usageHeader := styles.HelpSectionStyle.Render("USAGE")
	connectLine := styles.HelpArgStyle.Render(" Connect") + "\n" +
		styles.UsageStyle.Render("    evilnfsclient <server> <export> [OPTIONS]")
	listLine := styles.HelpArgStyle.Render(" List") + "\n" +
		styles.UsageStyle.Render("    evilnfsclient --list <server>")
	usageContent := lipgloss.JoinVertical(lipgloss.Left, connectLine, "", listLine)

	fmt.Println(styles.BoxStyle.Width(termWidth - 2).
		BorderForeground(lipgloss.Color("#4B9BFF")).
		Foreground(lipgloss.Color("#E0E0E0")).
		Render(usageHeader + "\n" + usageContent))

	// ----------------
	// ARGUMENTS (boxed)
	// ----------------
	argHeader := styles.HelpSectionStyle.Render("ARGUMENTS")
	argContent := styles.HelpArgStyle.Render("  SERVER") + "\n" +
		styles.HelpDescStyle.Render("    NFS server address (IP or hostname)") + "\n\n" +
		styles.HelpArgStyle.Render("  EXPORT") + "\n" +
		styles.HelpDescStyle.Render("    NFS export path (e.g., /shared, /mnt/nfs)")

	fmt.Println(styles.BoxStyle.Width(termWidth - 2).
		BorderForeground(lipgloss.Color("#04B575")).
		Foreground(lipgloss.Color("#E0E0E0")).
		Render(argHeader + "\n" + argContent))

	// ----------------
	// OPTIONS (boxed)
	// ----------------
	optHeader := styles.HelpSectionStyle.Render("OPTIONS")
	optContent := styles.HelpOptStyle.Render("  -l, --list ") + "\n" +
		styles.HelpDescStyle.Render("    List NFS exports on the given server and exit") + "\n\n" +
		styles.HelpOptStyle.Render("  -u, --uid <UID>") + "\n" +
		styles.HelpDescStyle.Render("    Set user ID for NFS operations (default: current user)") + "\n\n" +
		styles.HelpOptStyle.Render("  -g, --gid <GID>") + "\n" +
		styles.HelpDescStyle.Render("    Set group ID for NFS operations (default: current group)") + "\n\n" +
		styles.HelpOptStyle.Render("  -c, --command <COMMAND>") + "\n" +
		styles.HelpDescStyle.Render("    Execute single command without interactive TUI mode") + "\n\n" +
		styles.HelpOptStyle.Render("  -p, --privport ") + "\n" +
		styles.HelpDescStyle.Render("    Use privileged port for NFS connection (may require root)")
	fmt.Println(styles.BoxStyle.Width(termWidth - 2).
		BorderForeground(lipgloss.Color("#4B9BFF")).
		Foreground(lipgloss.Color("#E0E0E0")).
		Render(optHeader + "\n" + optContent))

	// ----------------
	// EXAMPLES (boxed and titled)
	// ----------------

	// show examples only if short is set to false
	if short {
		return
	}

	exHeader := styles.HelpSectionStyle.Render("EXAMPLES")
	examples := lipgloss.JoinVertical(lipgloss.Left,
		styles.HelpDescStyle.Render("Interactive mode with default user:"),
		styles.ExamplesSmallStyle.Render("  $ evilnfsclient 192.168.1.100 /shared"),
		"",
		styles.HelpDescStyle.Render("List NFS exports on a server:"),
		styles.ExamplesSmallStyle.Render("  $ evilnfsclient --list 192.168.1.100"),
		"",
		styles.HelpDescStyle.Render("Connect with specific UID/GID:"),
		styles.ExamplesSmallStyle.Render("  $ evilnfsclient 192.168.1.100 /shared --uid 1000 --gid 1000"),
		"",
		styles.HelpDescStyle.Render("Execute a single command:"),
		styles.ExamplesSmallStyle.Render("  $ evilnfsclient 192.168.1.100 /shared -c 'ls /'"),
		"",
		styles.HelpDescStyle.Render("Download a directory recursively:"),
		styles.ExamplesSmallStyle.Render("  $ evilnfsclient 192.168.1.100 /shared -c 'get /data -r'"),
		"",
		styles.HelpDescStyle.Render("Get help for commands:"),
		styles.ExamplesSmallStyle.Render("  $ evilnfsclient 192.168.1.100 /shared -c 'help'"),
	)

	// Put the header inside the box for a titled-box look
	fmt.Println(styles.BoxStyle.Width(termWidth - 2).
		BorderForeground(lipgloss.Color("#AAAAFF")).
		Foreground(lipgloss.Color("#D0D0D0")).
		Render(exHeader + "\n" + examples))

}

func checkValidArgs() {
	allowedFlags := map[string]struct{}{
		"-l": {}, "--list": {},
		"-u": {}, "--uid": {},
		"-g": {}, "--gid": {},
		"-c": {}, "--cmd": {},
		"-h": {}, "--help": {},
		"-p": {}, "--privport": {},
	}

	// parse only the arguments (not argv[0])
	rawArgs := os.Args[1:]

	// quick pass to detect unknown flags (helpful UX)
	// we allow flag values after a flag (simple heuristic: values not starting with '-')
	for i := 0; i < len(rawArgs); i++ {
		a := rawArgs[i]
		if strings.HasPrefix(a, "-") {
			// handle combined short flags like -abc? (we don't support), treat whole token
			// if flag is in allowed set, skip potential value afterwards for flags that expect one
			if _, ok := allowedFlags[a]; !ok {
				printUsage(true)
				fmt.Fprintln(os.Stderr, styles.ErrorStyle.Render(fmt.Sprintf("Unknown flag: %s", a)))
				fmt.Fprintln(os.Stderr)
				os.Exit(2)
			}
			// if it's a flag that takes a value, skip the next token (value) so we don't treat it as a flag
			if a == "-u" || a == "--uid" || a == "-g" || a == "--gid" || a == "-c" || a == "--cmd" {
				i++ // skip next token (if missing, parser.Parse will catch it)
			}
		}
	}
}
