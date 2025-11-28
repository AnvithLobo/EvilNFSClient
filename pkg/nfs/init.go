package nfs

import (
	"fmt"

	"os"

	"github.com/AnvithLobo/EvilNFSClient/pkg/ui/styles"
	nfs "github.com/AnvithLobo/nfsv3/nfs"
	"github.com/AnvithLobo/nfsv3/nfs/rpc"
)

type NFSClient struct {
	mount       *nfs.Target
	auth        *rpc.AuthUnix
	Server      string
	Export      string
	UID         uint32
	GID         uint32
	CurrentPath string
	localPath   string // Current working directory on the local system
}

// NewNFSClient creates and returns a new NFS client connection
func NewNFSClient(server, export string, uid, gid uint32, privPort bool) (*NFSClient, error) {
	auth := rpc.NewAuthUnix("root", uid, gid)

	mount, err := nfs.DialMount(server, privPort)
	if err != nil {
		return nil, fmt.Errorf("failed to dial MOUNT service: %v", err)
	}

	target, err := mount.Mount(export, auth.Auth())
	if err != nil {
		return nil, fmt.Errorf("failed to mount %s: %v", export, err)
	}

	// Get current local working directory
	localPath, err := os.Getwd()
	if err != nil {
		localPath = os.Getenv("PWD")
		if localPath == "" {
			localPath = os.Getenv("HOME")
		}
	}

	return &NFSClient{
		mount:       target,
		auth:        auth,
		Server:      server,
		Export:      export,
		UID:         uid,
		GID:         gid,
		CurrentPath: "/",
		localPath:   localPath,
	}, nil
}

// ExecuteCommand routes commands to their respective handlers
func (c *NFSClient) ExecuteCommand(command string) []string {
	parts := parseCommand(command)
	if len(parts) == 0 {
		return []string{}
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "ls":
		return c.ls(args)
	case "tree":
		return c.tree(args)
	case "cd":
		return c.cd(args)
	case "get":
		return c.get(args)
	case "mget":
		return c.mget(args)
	case "put":
		return c.put(args)
	case "mput":
		return c.mput(args)
	case "rm":
		return c.rm(args)
	case "mkdir":
		return c.mkdir(args)
	case "lls":
		return c.lls(args)
	case "lcd":
		return c.lcd(args)
	case "lmkdir":
		return c.lmkdir(args)
	case "chmod":
		return c.chmod(args)
	case "help":
		return c.help()
	default:
		return []string{styles.ErrorStyle.Render(fmt.Sprintf("Unknown command: %s (try 'help')", cmd))}
	}
}

func (c *NFSClient) help() []string {
	return []string{
		styles.TitleStyle.Render("Available Commands:"),
		"",
		styles.TitleStyle.Render("NFS Commands:"),
		styles.HelpExampleStyle.Render("ls") + " " + styles.HelpOptStyle.Render("[path]") + "                      - List files in directory (current dir if no path)",
		styles.HelpExampleStyle.Render("cd") + " " + styles.HelpArgStyle.Render("<path>") + "                      - Change to directory on NFS share",
		styles.HelpExampleStyle.Render("tree") + " " + styles.HelpOptStyle.Render("[path]") + "                    - Show directory tree (current dir if no path)",
		styles.HelpExampleStyle.Render("mkdir") + " " + styles.HelpOptStyle.Render("[-p]") + " " + styles.HelpArgStyle.Render("<path>") + "              - Create directory on NFS share (-p creates parents)",
		styles.HelpExampleStyle.Render("get") + " " + styles.HelpOptStyle.Render("[-r]") + " " + styles.HelpArgStyle.Render("<remote>") + " " + styles.HelpOptStyle.Render("[<local>]") + "    - Download file/dir (-r for recursive, uses remote name if local omitted)",
		styles.HelpExampleStyle.Render("mget") + " " + styles.HelpArgStyle.Render("<pattern>") + " " + styles.HelpOptStyle.Render("[<dest_dir>]") + "    - Download multiple files matching pattern",
		styles.HelpExampleStyle.Render("put") + " " + styles.HelpOptStyle.Render("[-r]") + " " + styles.HelpArgStyle.Render("<local>") + " " + styles.HelpOptStyle.Render("[<remote>]") + "    - Upload file/dir (-r for recursive, uses local name if remote omitted)",
		styles.HelpExampleStyle.Render("mput") + " " + styles.HelpArgStyle.Render("<pattern>") + " " + styles.HelpOptStyle.Render("[<dest_path>]") + "   - Upload multiple files matching pattern",
		styles.HelpExampleStyle.Render("rm") + " " + styles.HelpOptStyle.Render("[-r]") + " " + styles.HelpArgStyle.Render("<path>") + "                 - Remove file/dir (-r for recursive removal)",
		styles.HelpExampleStyle.Render("chmod") + " " + styles.HelpArgStyle.Render("<mode>") + " " + styles.HelpArgStyle.Render("<file>") + "            - Change file permissions (e.g., 4777)",
		"",
		styles.TitleStyle.Render("Local Commands (prefixed with 'l'):"),
		styles.HelpExampleStyle.Render("lls") + " " + styles.HelpOptStyle.Render("[path]") + "                     - List files in local directory (current dir if no path)",
		styles.HelpExampleStyle.Render("lcd") + " " + styles.HelpArgStyle.Render("<path>") + "                     - Change to directory on local system",
		styles.HelpExampleStyle.Render("lmkdir") + " " + styles.HelpOptStyle.Render("[-p]") + " " + styles.HelpArgStyle.Render("<path>") + "             - Create directory on local system (-p creates parents)",
		"",
		styles.HelpExampleStyle.Render("help") + "                           - Show this help",
		styles.HelpExampleStyle.Render("exit") + "/" + styles.HelpExampleStyle.Render("quit") + "                      - Exit the client",
	}
}
