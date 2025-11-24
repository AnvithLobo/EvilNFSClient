# EvilNFSClient

A focused NFS client for security researchers and pentesters that makes direct manipulation of NFS-mounted filesystems easy — including operations that help test post-exploitation and NFS-based privilege escalation scenarios.

Repository: `github.com/AnvithLobo/EvilNFSClient`

<img width="1062" height="551" alt="image" src="https://github.com/user-attachments/assets/87e7f22e-8c79-439f-a703-b0732f56ac5d" />

---

## Table of contents

- [EvilNFSClient](#evilnfsclient)
  - [Table of contents](#table-of-contents)
  - [What it does](#what-it-does)
  - [Quick start](#quick-start)
  - [Features](#features)
    - [Remote NFS operations](#remote-nfs-operations)
    - [Local operations (prefix `l`)](#local-operations-prefix-l)
    - [Session control](#session-control)
  - [Commands (high level)](#commands-high-level)
  - [Examples](#examples)
  - [Permissions \& SUID/SGID notes](#permissions--suidsgid-notes)
  - [Installation](#installation)
    - [Prerequisites](#prerequisites)
    - [Build](#build)
  - [Usage \& options](#usage--options)
  - [Tips \& behavior](#tips--behavior)
  - [Disclaimer \& legal](#disclaimer--legal)
  - [Author](#author)
  - [License](#license)

---

## What it does

EvilNFSClient is a terminal-based NFS client that exposes filesystem operations on an NFS export with convenience features useful for red team / pentest workflows or just as a quick CLI based NFS file manager without the overhead of mounting.:

* read/write/delete files and directories
* recursive upload/download (preserve tree structures)
* directly set SUID, SGID and sticky bits
* colorized, aligned directory listings for quick scanning
* interactive shell with command history and paging

---

## Quick start

- Build it yourself:

```bash
git clone github.com/AnvithLobo/EvilNFSClient
cd EvilNFSClient
go build -o evilnfsclient
./evilnfsclient <server-ip> <export-path>
```


- Or download pre-built binaries from the [releases page](https://github.com/AnvithLobo/EvilNFSClient/packages).



---

## Features

<img width="1048" height="1002" alt="image" src="https://github.com/user-attachments/assets/54032c3b-05e8-4e7c-9caf-fc8641fa2cb3" />


### Remote NFS operations

* `ls [path]` — list remote directory (colorized)
* `cd <path>` — change remote working dir
* `tree [path]` — recursive tree view
* `get [-r] <remote> [<local>]` — download file/dir
* `mget <pattern> [<dest_dir>]` — download by glob
* `put [-r] <local> [<remote>]` — upload file/dir
* `mput <pattern> [<dest_path>]` — upload multiple local files
* `rm [-r] <path>` — remove file/dir
* `mkdir [-p] <path>` — create directory
* `chmod <mode> <file>` — set permission bits including SUID/SGID/sticky

### Local operations (prefix `l`)

* `lls [path]`, `lcd <path>`, `lmkdir [-p] <path>`



### Session control

* `help`, `exit`/`quit`
* arrow keys for history, PgUp/PgDn for scroll
* Ctrl+C to quit

---

## Commands (high level)

Show common usage patterns inside the interactive shell.

Navigation & exploration:

```bash
nfs> ls
nfs> cd public
nfs> tree
```

File transfer:

```bash
# upload
nfs> put /tmp/shell.sh payload.sh

# download
nfs> get payload.sh ./downloaded.sh

# recursive upload/download
nfs> put -r ./tools /shared/tools
nfs> get -r /shared/sensitive /tmp/data
```

Batch transfers (globs):

```bash
nfs> mget *.log ./logs/
nfs> mput /tmp/*.elf /shared/payloads/
```

Directory management:

```bash
nfs> mkdir -p /shared/a/b/c
nfs> rm -r /shared/old_stuff
```

Local commands:

```bash
nfs> lcd /tmp
nfs> lmkdir -p workspace/subdir
nfs> lls
```

---

## Examples

Interactive (default UID/GID):

```bash
./evilnfsclient 192.168.1.100 /shared
```

Interactive (custom UID/GID):

```bash
./evilnfsclient 192.168.1.100 /shared --uid 0 --gid 0
```

Non-interactive (single command):

```bash
./evilnfsclient 192.168.1.100 /shared -c "ls /"
```

---

## Permissions & SUID/SGID notes

`chmod` supports standard octal modes and honor SUID/SGID/sticky bits:

* `chmod 4755 /shared/binary` → SUID set (`-rwsr-xr-x`)
* `chmod 2755 /shared/binary` → SGID set (`-rwxr-sr-x`)
* `chmod 6777 /shared/binary` → combined special bits (`-rwsrwsrwx`)

**Security note:** setting SUID/SGID on binaries can enable privilege escalation when executed on a system that honors those bits. The client manipulates the NFS export entries; whether the target system executes with escalated privileges depends on the target host and its mount/OS behavior.

Example (create SUID binary):

```bash
nfs> put ./shell /shared/shell
nfs> chmod 6755 /shared/shell
# On the target host: /shared/shell  -> may run with owner privileges
```

---

## Installation

### Prerequisites

* Go 1.24.x or later
* Network access to the NFS server/export

### Build

```bash
git clone https://github.com/AnvithLobo/EvilNFSClient
cd EvilNFSClient
go build -o evilnfsclient
```

---

## Usage & options

```bash
./evilnfsclient <server> <export> [options]
```

Options:

* `--uid <uid>` — run NFS ops as this UID (default: your current UID)
* `--gid <gid>` — run NFS ops as this GID (default: your current GID)
* `-c <command>` — execute command and exit (non-interactive)

---

## Tips & behavior

* Relative remote paths resolve against the current remote directory.
* Relative local paths resolve against the current local working directory.
* Home expansion (`~`) is supported for local paths.
* Long outputs can be paged with PgUp/PgDn; use command history to re-run previous commands quickly.
* Use `-r` carefully on `rm` / `put` / `get` — recursive operations are powerful and destructive.

---

## Disclaimer & legal

This tool is intended strictly for authorized security testing and penetration testing. Unauthorized access to computer systems, networks, or data is illegal. Use EvilNFSClient **only** on systems you own or for which you have explicit permission.

By using this software you accept responsibility for your actions.

---

## Author

Anvith Lobo — [@AnvithLobo](https://github.com/AnvithLobo)

## License

See `LICENSE` in this repository for license details.


---