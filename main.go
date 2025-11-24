package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/AnvithLobo/EvilNFSClient/pkg/nfs"
	"github.com/AnvithLobo/EvilNFSClient/pkg/ui"
	"github.com/AnvithLobo/EvilNFSClient/pkg/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Check for help flag first
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage()
			os.Exit(0)
		}
	}

	if len(os.Args) < 3 {
		printUsage()
		os.Exit(1)
	}

	server := os.Args[1]
	export := os.Args[2]

	uid := uint32(os.Getuid())
	gid := uint32(os.Getgid())
	var command string

	// Parse optional arguments
	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--uid":
			if i+1 < len(os.Args) {
				val, _ := strconv.ParseUint(os.Args[i+1], 10, 32)
				uid = uint32(val)
				i++
			}
		case "--gid":
			if i+1 < len(os.Args) {
				val, _ := strconv.ParseUint(os.Args[i+1], 10, 32)
				gid = uint32(val)
				i++
			}
		case "-c":
			if i+1 < len(os.Args) {
				command = os.Args[i+1]
				i++
			}
		}
	}

	client, err := nfs.NewNFSClient(server, export, uid, gid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Non-interactive mode
	if command != "" {
		output := client.ExecuteCommand(command)
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

	// Print all buffered output to stdout so it persists after TUI closes
	if finalModelCast, ok := finalModel.(ui.TUIModel); ok {
		// Print header
		fmt.Println(styles.TitleStyle.Render("🔥 EvilNFSClient"))
		fmt.Println(lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("Connected to %s:%s (UID: %d, GID: %d) | Path: %s",
			finalModelCast.Client.Server, finalModelCast.Client.Export, finalModelCast.Client.UID, finalModelCast.Client.GID, finalModelCast.Client.CurrentPath)))
		fmt.Println()

		// Print all buffered output
		for _, line := range finalModelCast.Output {
			fmt.Println(line)
		}

		// Print goodbye message
		fmt.Println(styles.SuccessStyle.Render("Goodbye!"))
	}
}

func printUsage() {
	fmt.Print("\n")
	fmt.Println(styles.HelpTitleStyle.Render("🔥 EvilNFSClient"))
	fmt.Println(styles.HelpDescStyle.Render("Modern NFS Client for Hackers"))
	fmt.Print("\n")

	// Usage section
	fmt.Println(styles.HelpSectionStyle.Render("USAGE"))
	usageBox := lipgloss.NewStyle().
		Margin(0, 2).
		Foreground(lipgloss.Color("#A89BFF"))
	fmt.Println(usageBox.Render("evilnfsclient <server> <export> [OPTIONS]"))
	fmt.Print("\n")

	// Arguments section
	fmt.Println(styles.HelpSectionStyle.Render("ARGUMENTS"))
	argBox := lipgloss.NewStyle().Margin(0, 2)
	fmt.Println(argBox.Render(
		styles.HelpArgStyle.Render("  SERVER") + "\n" +
			styles.HelpDescStyle.Render("    NFS server address (IP or hostname)") + "\n" +
			"\n" +
			styles.HelpArgStyle.Render("  EXPORT") + "\n" +
			styles.HelpDescStyle.Render("    NFS export path (e.g., /shared, /mnt/nfs)"),
	))
	fmt.Print("\n")

	// Options section
	fmt.Println(styles.HelpSectionStyle.Render("OPTIONS"))
	optBox := lipgloss.NewStyle().Margin(0, 2)
	fmt.Println(optBox.Render(
		styles.HelpOptStyle.Render("  --uid <UID>") + "\n" +
			styles.HelpDescStyle.Render("    Set user ID for NFS operations (default: current user)") + "\n" +
			"\n" +
			styles.HelpOptStyle.Render("  --gid <GID>") + "\n" +
			styles.HelpDescStyle.Render("    Set group ID for NFS operations (default: current group)") + "\n" +
			"\n" +
			styles.HelpOptStyle.Render("  -c, --command <COMMAND>") + "\n" +
			styles.HelpDescStyle.Render("    Execute single command without interactive TUI mode"),
	))
	fmt.Print("\n")

	// Examples section
	fmt.Println(styles.HelpSectionStyle.Render("EXAMPLES"))

	fmt.Println(styles.HelpDescStyle.Render("  Interactive mode with default user:"))
	fmt.Println(styles.HelpExampleBgStyle.Render("    $ evilnfsclient 192.168.1.100 /shared"))
	fmt.Print("\n")

	fmt.Println(styles.HelpDescStyle.Render("  Connect with specific UID/GID:"))
	fmt.Println(styles.HelpExampleBgStyle.Render("    $ evilnfsclient 192.168.1.100 /shared --uid 1000 --gid 1000"))
	fmt.Print("\n")

	fmt.Println(styles.HelpDescStyle.Render("  Execute single command:"))
	fmt.Println(styles.HelpExampleBgStyle.Render("    $ evilnfsclient 192.168.1.100 /shared -c 'ls /'"))
	fmt.Print("\n")

	fmt.Println(styles.HelpDescStyle.Render("  Download a directory recursively:"))
	fmt.Println(styles.HelpExampleBgStyle.Render("    $ evilnfsclient 192.168.1.100 /shared -c 'get /data -r'"))
	fmt.Print("\n")

	// print help example
	fmt.Println(styles.HelpDescStyle.Render("  Get help information for NFS operations:"))
	fmt.Println(styles.HelpExampleBgStyle.Render("    $ evilnfsclient 192.168.1.100 /shared -c 'help'"))
	fmt.Print("\n")

}
