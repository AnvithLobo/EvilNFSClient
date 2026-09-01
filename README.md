<div align="left" style="position: relative;">
<img src="https://github.com/user-attachments/assets/8ff7308b-c6d4-454a-bba4-77f3c3409601" align="right" width="22%" style="margin: -20px 0 0 20px;">
<h1>EvilNFSClient</h1>

<p align="left">
	<img src="https://img.shields.io/github/license/AnvithLobo/EvilNFSClient?style=for-the-badge&logo=opensourceinitiative&logoColor=white&color=0080ff" alt="license">
	<img src="https://img.shields.io/github/last-commit/AnvithLobo/EvilNFSClient?style=for-the-badge&logo=git&logoColor=white&color=0080ff" alt="last-commit">
	<img src="https://img.shields.io/github/languages/top/AnvithLobo/EvilNFSClient?style=for-the-badge&color=0080ff" alt="repo-top-language">
</p>
<p align="left">Built with the tools and technologies:</p>
<p align="left">
	<img src="https://img.shields.io/badge/Go-00ADD8.svg?style=for-the-badge&logo=Go&logoColor=white" alt="Go">
</p>
</div>
<br clear="right">

A modern, fast, and pentester-friendly NFS client built for red teams, security researchers, and anyone who wants full control over remote NFS exports — **without needing to mount them**.

Repository: `github.com/AnvithLobo/EvilNFSClient`

<img width="1019" height="791" alt="image" src="https://github.com/user-attachments/assets/5794d315-01d7-47e7-b83b-2ffadc4abb4a" />

---

## 📑 Table of Contents

- [📑 Table of Contents](#-table-of-contents)
- [🚀 What it does](#-what-it-does)
- [⚡ Quick start](#-quick-start)
  - [🔧 Build from source](#-build-from-source)
  - [📥 Download binaries](#-download-binaries)
- [🧰 Features](#-features)
  - [🌐 Remote NFS operations](#-remote-nfs-operations)
  - [💻 Local operations (prefix `l`)](#-local-operations-prefix-l)
  - [🎛️ Session control](#️-session-control)
- [📘 Commands (high level)](#-commands-high-level)
  - [🔎 Navigation](#-navigation)
  - [📤 Upload](#-upload)
  - [📥 Download](#-download)
  - [📦 Multi-file](#-multi-file)
  - [🗂 Directory mgmt](#-directory-mgmt)
  - [💻 Local commands](#-local-commands)
- [🧪 Examples](#-examples)
- [🔐 Permissions \& SUID/SGID Notes](#-permissions--suidsgid-notes)
  - [Priv-Esc scenario](#priv-esc-scenario)
- [📦 Installation](#-installation)
  - [🛠 Prerequisites](#-prerequisites)
  - [🔨 Build](#-build)
- [⚙️ Usage \& options](#️-usage--options)
- [💡 Tips \& behavior](#-tips--behavior)
- [🛠️ Troubleshooting](#️-troubleshooting)
  - [MNT3ERR\_ACCES on mount](#mnt3err_acces-on-mount)
- [📌 Project Roadmap](#-project-roadmap)
- [⚠️ Disclaimer](#️-disclaimer)
- [👤 Author](#-author)
- [📄 License](#-license)

---

## 🚀 What it does

EvilNFSClient is a **TUI-powered**, and **powerful** NFS client designed for offensive security workflows and regular use.

Use it as:

* A **post-exploitation helper**
* A **privilege escalation tool**
* A **standalone NFS file manager** (no mount needed!)
* A **fast recursive uploader/downloader**

✨ Features at a glance:

* Full file manipulation (read, write, delete)
* Upload/download directories with `-r`
* Set **SUID/SGID/Sticky bit** permissions
* Interactive shell with history + scrolling

---

## ⚡ Quick start

### 🔧 Build from source

```bash
git clone https://github.com/AnvithLobo/EvilNFSClient
cd EvilNFSClient
go build -o evilnfsclient
./evilnfsclient <server-ip> <export-path>
```

### 📥 Download binaries

➡️ Pre-built releases: **[https://github.com/AnvithLobo/EvilNFSClient/releases](https://github.com/AnvithLobo/EvilNFSClient/releases)**

---

## 🧰 Features

<img width="1062" height="551" alt="image" src="https://github.com/user-attachments/assets/87e7f22e-8c79-439f-a703-b0732f56ac5d" />


### 🌐 Remote NFS operations

* `ls [path]` — colorized directory listing
* `cd <path>` — switch directories
* `tree [path]` — recursive view
* `get [-r] <remote> [<local>]` — download
* `mget <pattern> [<dest_dir>]` — multi-download
* `put [-r] <local> [<remote>]` — upload
* `mput <pattern> [<dest_path>]` — multi-upload
* `rm [-r] <path>` — delete files/folders
* `mkdir [-p] <path>` — create directories
* `chmod <mode> <file>` — permission editing w/ SUID/SGID

### 💻 Local operations (prefix `l`)

* `lls [path]`, `lcd <path>`, `lmkdir [-p] <path>`

### 🎛️ Session control

* `help`  → show commands
* Arrow keys → history
* PgUp/PgDn → scroll
* `Ctrl + C`, `exit`, `quit` → exit

---

## 📘 Commands (high level)

### 🔎 Navigation

```bash
nfs> ls
nfs> cd public
nfs> tree
```

### 📤 Upload

```bash
nfs> put /tmp/shell.sh payload.sh
nfs> put -r ./tools /shared/tools
```

### 📥 Download

```bash
nfs> get payload.sh ./downloaded.sh
nfs> get -r /shared/sensitive /tmp/data
```

### 📦 Multi-file

```bash
nfs> mget *.log ./logs/
nfs> mput /tmp/*.elf /shared/payloads/
```

### 🗂 Directory mgmt

```bash
nfs> mkdir -p /shared/a/b/c
nfs> rm -r /shared/old_stuff
```

### 💻 Local commands

```bash
nfs> lcd /tmp
nfs> lmkdir -p workspace/subdir
nfs> lls
```

---

## 🧪 Examples

```bash
# Connect to NFS server with the export /shared
./evilnfsclient 192.168.1.100 /shared
# List available NFS exports on the server
./evilnfsclient --list 192.168.1.100
# Connect with overridden UID and GID
./evilnfsclient 192.168.1.100 /shared --uid 0 --gid 0
# Run a single command non-interactively
./evilnfsclient 192.168.1.100 /shared -c "ls /"
# Connect using a privileged source port (needed for exports with the 'secure' option)
sudo ./evilnfsclient 192.168.1.100 /shared -p
# Privileged port with spoofed UID/GID
sudo ./evilnfsclient 192.168.1.100 /shared -p --uid 0 --gid 0
```

---

## 🔐 Permissions & SUID/SGID Notes

Supported modes include full SUID, SGID, and sticky bit manipulation.

Examples:

* `chmod 4755 file` → **SUID**
* `chmod 2755 file` → **SGID**
* `chmod 6777 file` → **SUID + SGID**

### Priv-Esc scenario

```bash
nfs> put ./shell /shared/shell
nfs> chmod 6755 /shared/shell
```

Then on the target:

```bash
/shared/shell
```

⚠️ Whether the target honors SUID/SGID over NFS depends on mount + OS settings.

---

## 📦 Installation

### 🛠 Prerequisites

* Go 1.24.x+
* Access to NFS server/export

### 🔨 Build

```bash
git clone https://github.com/AnvithLobo/EvilNFSClient
cd EvilNFSClient
go build -o evilnfsclient
```

---

## ⚙️ Usage & options

```bash
./evilnfsclient <server> <export> [options]
```

Options:

* `--uid <uid>` / `-u` — override UID
* `--gid <gid>` / `-g` — override GID
* `-c <cmd>` — run command (non-interactive)
* `-p` / `--privport` — bind to a privileged source port (< 1024); required when the NFS export uses the `secure` option

---

## 💡 Tips & behavior

* Remote paths → resolved against remote CWD
* Local paths → resolved against local CWD
* `~` expansion supported
* PgUp/PgDn scrolls output
* Use `-r` with caution (recursive delete!)

---

## 🛠️ Troubleshooting

### MNT3ERR_ACCES on mount

```
Error: failed to mount /export/path: MNT3ERR_ACCES
```

This means the NFS server has the export configured with the **`secure`** option (the default on most Linux NFS servers). It requires the client to connect from a **source port below 1024**, which only root (or a process with `CAP_NET_BIND_SERVICE`) can bind to.

**Fix — Use `sudo` with the `-p` flag:**

```bash
sudo evilnfsclient <server> <export> -p
```

When `-p` is not used and the error occurs, the tool will automatically print a hint with the exact commands to run.

---

## 📌 Project Roadmap

- [X] **`Task 1`**: <strike>General NFS Commands.</strike>
- [X] **`Task 2`**: <strike>List NFS Exports.</strike>
- [ ] **`Task 3`**: Check for `Root File System Escape`.

---

## ⚠️ Disclaimer

This tool is intended strictly for authorized security testing and penetration testing. Unauthorized access to computer systems, networks, or data is illegal. Use EvilNFSClient **only** on systems you own or for which you have explicit permission.

By using this software you accept responsibility for your actions.

---

## 👤 Author

**Anvith Lobo** — [https://github.com/AnvithLobo](https://github.com/AnvithLobo)

---

## 📄 License

See `LICENSE` for details.

---

✨ *Built for red teamers and power users who need full control over NFS exports.*
