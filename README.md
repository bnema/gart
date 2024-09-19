# Gart - Dotfile Manager

Gart is a command-line tool written in Go that helps you manage and sync your dotfiles across different systems. With Gart, you can easily keep your configuration files up to date and maintain a consistent setup across multiple machines.

## Features
- **Quick Addition**: Add a dotfile directory or a single file to Gart with a single command (e.g., `gart add ~/.config/zsh` or `gart add ~/.config/nvim/init.lua`)
- **Easy sync**: Use the sync command to detect changes in all your managed dotfiles and backup them automatically (e.g., `gart sync` or for a single dotfile `gart sync nvim`)
- **Simple Overview**: List, select and remove the dotfiles currently being managed with `gart list`
- **Flexible Naming**: (Optional) assign custom names to your dotfiles for easier management (e.g., `gart add ~/.config/nvim nvim-backup`)
- **Git Versioning:** (Optional) Git-based version control with templated, configurable commits and customizable branch names (default: hostname).
- **Auto-Push:** (Optional) Push changes to the remote repository automatically.

![Demo Deploy](assets/demo.gif?raw=true)

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
or
gart add ~/.config/hypr Hyprland
```

To update/synchronize a specific dotfile, use the `sync` command followed by the name of the dotfile:
```
gart sync nvim
```
This will detect changes in the specified dotfile and save the updated version to your designated store directory.

To sync all the dotfiles specified in the `config.toml` file, simply run:
```
gart sync
```

To list all the dotfiles currently being managed by Gart, use the `list` command:
```
gart list
```
This will display a list of all the dotfiles specified in the `config.toml` file.

## Configuration

Gart uses a `config.toml` file for configuration, which is automatically created in the default location (`$XDG_CONFIG_HOME/gart/config.toml`) if it doesn't exist. This file allows you to specify the dotfiles you want to manage and configure various settings.

The configuration file is divided into two main sections: `[dotfiles]` and `[settings]`.

### Dotfiles Section

The `[dotfiles]` section lists the dotfiles you want to manage. Each entry represents a dotfile, with the key being the name of the dotfile and the value being the path to the dotfile on your local system.

Example:

```toml
[dotfiles]
alacritty = "/home/user/.config/alacritty"
nvim = "/home/user/.config/nvim"
starship = "/home/user/.config/starship.toml"
```

### Settings Section

The `[settings]` section contains global configuration options for Gart:

```toml
[settings]
git_versioning = true
storage_path = "/home/user/.config/gart/.store"

[settings.git]
branch = "custom-branch-name"
commit_message_format = "{{ .Action }} {{ .Dotfile }}"
```

- `git_versioning`: Enables or disables Git versioning for your dotfiles.
- `storage_path`: Sets the directory where Gart stores managed dotfiles.
- `[settings.git]`: Subsection for Git-specific settings.
  - `branch`: Specifies the Git branch to use for versioning. If not set, the default branch name will be the hostname of your machine.
  - `commit_message_format`: Specifies the format of the commit message when updating a dotfile. The message is templated using Go's text/template package and has access to the following fields (for now):
    - `.Action`: The action performed (e.g., "Add", "Update", "Remove").
    - `.Dotfile`: The name of the dotfile being handled.

## Roadmap
- [x] Allow adding a single file
- [x] Create a state with git after each detected change
- [x] Custom store set in the config file
- [x] Remove a dotfile from the list view
- [x] Version command
- [ ] Status command to display the status of all the dotfiles (last commit, changes, etc.)
- [ ] Remove command to remove a dotfile from the store
- [ ] Update command to update Gart to the latest version
- [ ] Add more templated fields for the commit message (e.g., date, time, etc.)


## License

This project is licensed under the [MIT License](LICENSE).
