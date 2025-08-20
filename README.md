# Gart - Dotfile Manager

Gart is a command-line tool written in Go that helps you manage and sync your dotfiles across different Linux systems.

![Demo Deploy](assets/demo.gif?raw=true)

## Navigation

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Roadmap](#roadmap)
- [License](#license)

## Features
- **Quick Addition**: Add a dotfile directory or a single file to Gart with a single command (e.g., `gart add ~/.config/zsh` or `gart add ~/.config/nvim/init.lua`)
- **Security Scanning**: Detects sensitive information like API keys, passwords, and tokens in your dotfiles before syncing
- **Ignore Patterns**: Exclude specific files or directories using the `--ignore` flag (e.g., `gart add ~/.config/nvim --ignore "init.bak" --ignore "doc/"`)
- **Easy sync**: Use the sync command to detect changes in all your managed dotfiles and backup them automatically (e.g., `gart sync` or for a single dotfile `gart sync nvim`)
- **Simple Overview**: List, select and remove the dotfiles currently being managed with `gart list`
- **Flexible Naming**: (Optional) assign custom names to your dotfiles for easier management (e.g., `gart add ~/.config/nvim nvim-backup`)
- **Git Versioning:** (Optional) Git-based version control with templated, configurable commits and customizable branch names (default: hostname).
- **Auto-Push:** (Optional) Push changes to the remote repository automatically.

## Installation

### Prerequisites

- Linux
- Go >= 1.22

### Downloading the Binary

**New**: Pre-built binaries for Linux (amd64 and arm64) are now available for each release. You can download them from the [Releases page](https://github.com/bnema/gart/releases).
1. Extract the archive:
   ```bash
   tar -xzf gart_*_linux_*.tar.gz
   ```
2. Make the binary executable:
   ```bash
   chmod +x gart
   ```
3. Move the binary to a directory in your PATH:
   ```bash
   sudo mv gart /usr/local/bin/
   or
   sudo mv gart /usr/bin/
   or
   mv gart /home/user/.local/bin/
   ```

###  Building from Source

You can install Gart using this one-liner, which clones the repository, builds the binary, and installs it:
```bash
git clone https://github.com/bnema/gart.git && cd gart && make && sudo make install
```
   Note: This method requires sudo privileges to move the binary to the /usr/bin directory.

### Installing with Go Install
Alternatively, you can install Gart directly using Go's install command:
```
go install github.com/bnema/gart@latest
```
This will install the latest version of Gart to your `$GOPATH/bin` directory. Make sure this directory is in your system's PATH to run Gart from anywhere.

## Usage

To add a new dotfile to the configuration, use the `add` command followed by the path to the dotfile and the name (optional)
```
gart add ~/.config/nvim
# or with a custom name
gart add ~/.config/hypr Hyprland
# or with ignore patterns
gart add ~/.config/fish --ignore "*.log" --ignore "cache/"
```

Note: The `--ignore` flag allows you to specify patterns for files or directories that should be excluded when adding or syncing dotfiles. You can specify multiple patterns by using the flag multiple times or by editing your `config.toml` file under the `[dotfiles.ignores]` section.

To update/synchronize a specific dotfile, use the `sync` command followed by the name of the dotfile:
```
gart sync nvim
```
This will detect changes in the specified dotfile and save the updated version to your designated store directory.

To sync all the dotfiles specified in the `config.toml` file, simply run:
```
gart sync
```

To skip security scanning (useful when you're sure your files are clean or for private repos):
```
gart sync --no-security
# or for a specific dotfile
gart sync nvim --no-security
```

To list all the dotfiles currently being managed by Gart, use the `list` command:
```
gart list
```
This will display a list of all the dotfiles specified in the `config.toml` file.

## Configuration

Gart uses a `config.toml` file for configuration, which is automatically created if it doesn't exist. The configuration and data storage locations follow platform-specific conventions:

**Linux/Unix**: 
- Config: `$XDG_CONFIG_HOME/gart/config.toml` (defaults to `~/.config/gart/config.toml`)
- Data: `$XDG_DATA_HOME/gart/store/` (defaults to `~/.local/share/gart/store/`)

**Windows**:
- Config: `%APPDATA%\gart\config.toml`
- Data: `%LOCALAPPDATA%\gart\store\`

**macOS**:
- Config: `~/Library/Preferences/gart/config.toml`
- Data: `~/Library/Application Support/gart/store/`

The configuration file is divided into two main sections: `[dotfiles]` and `[settings]`.

### Dotfiles Section

The `[dotfiles]` section lists the dotfiles you want to manage, and the optional `[dotfiles.ignores]` section specifies patterns to ignore for each dotfile.

Example:

```toml
[dotfiles]
alacritty = "/home/user/.config/alacritty"
nvim = "/home/user/.config/nvim"
starship = "/home/user/.config/starship.toml"
fish = "/home/user/.config/fish"

[dotfiles.ignores]
alacritty = ["config.bak"]
fish = ["*.json", "cache/", "temp*/", "**/*.log"]
nvim = ["*.swap", "backup/"]
```

Common ignore pattern examples:
```toml
[dotfiles.ignores]
dotfile = [
    "cache/",            # Ignores cache directory
    "*/temp/",           # Ignores temp directories one level deep
    "**/node_modules/",  # Ignores node_modules directories at any depth
    "*.log",             # Ignores all log files
    "test*/",            # Ignores directories starting with test
    "*_modules/",        # Ignores directories ending with _modules
    "*.{jpg,png,gif}",   # Ignores common image files
]
```
Note: All the `.git/` directories are ignored by default.

### Settings Section

The `[settings]` section contains global configuration options for Gart:

```toml
[settings]
git_versioning = true
storage_path = "/home/user/.local/share/gart/store"
reverse_sync = false

[settings.git]
auto_push = false
branch = "custom-branch-name"
commit_message_format = "{{ .Action }} {{ .Dotfile }}"
```

- `git_versioning`: Enables or disables Git versioning for your dotfiles.
- `storage_path`: Sets the directory where Gart stores managed dotfiles.
- `reverse_sync`: Determines the direction of synchronization:
  - `false` (default): Push mode - syncs from local config files (~/.config) to store directory
  - `true`: Pull mode - syncs from store directory to local config files.
- `[settings.git]`: Subsection for Git-specific settings.
  - `auto_push`: Enables or disables auto-pushing to the remote repository. (You must have a remote repository set up)
  - `branch`: Specifies the Git branch to use for versioning. If not set, the default branch name will be the hostname of your machine.
  - `commit_message_format`: Specifies the format of the commit message when updating a dotfile. The message is templated using Go's text/template package and has access to the following fields (for now):
    - `.Action`: The action performed (e.g., "Add", "Update", "Remove").
    - `.Dotfile`: The name of the dotfile being handled.

### Security Configuration

Gart includes comprehensive security scanning to detect sensitive information in dotfiles before adding them. The security features are enabled by default but can be customized or disabled entirely.

```toml
[settings.security]
enabled = true              # Enable/disable security scanning entirely
scan_content = true         # Enable/disable content scanning for secrets
exclude_patterns = true     # Enable/disable pattern-based exclusions
sensitivity = "medium"      # Sensitivity level: "low", "medium", "high", "paranoid"
fail_on_secrets = true      # Fail when secrets are found (vs warning only)
interactive = true          # Show interactive prompts for security findings

[settings.security.content_scan]
entropy_threshold = 4.5     # Minimum entropy for secret detection
min_secret_length = 20      # Minimum length for potential secrets
max_file_size = 5242880     # Maximum file size to scan (5MB)
scan_binary_files = false   # Whether to scan binary files
context_window = 50         # Lines of context around findings

[settings.security.allowlist]
patterns = ["TEST_*", "DEMO_*"]  # Allowed secret patterns
files = ["test.env", "demo.config"]  # Files to skip scanning
```

**What the security scanner does:**
- Finds API keys, tokens, passwords, and other secrets in your files
- Recognizes common patterns like AWS keys, GitHub tokens, JWT tokens, etc.
- Avoids flagging normal config text, UI descriptions, and documentation
- Shows you what it found and lets you decide what to do
- Groups findings by how serious they are (Critical/High/Medium/Low)

**To completely disable security scanning:**
```toml
[settings.security]
enabled = false
```

**To skip security scanning for specific syncs:**
```bash
gart sync --no-security        # Skip for all dotfiles
gart sync nvim --no-security   # Skip for one dotfile
```

## Roadmap
- [x] Allow adding a single file
- [x] Create a state with git after each detected change
- [x] Custom store set in the config file
- [x] Remove a dotfile from the list view
- [x] Version command
- [x] Auto-push feature
- [x] Ignore flag for the add command
- [x] Reverse sync mode
- [x] Security scanning for sensitive information
- [x] Option to skip security scanning (--no-security flag)
- [ ] Status command to display the status of all the dotfiles (last commit, changes, etc.)
- [ ] Remove command to remove a dotfile from the store
- [ ] Update command to update Gart to the latest version
- [ ] Add more templated fields for the commit message (e.g., date, time, etc.)


## License

This project is licensed under the [MIT License](LICENSE).
