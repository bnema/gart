# Gart - Dotfile Manager

Gart is a command-line tool written in Go that helps you manage and sync your dotfiles across different systems. With Gart, you can easily keep your configuration files up to date and maintain a consistent setup across multiple machines.

## Features
- **Easy Addition**: Add a dotfile directory to Gart with a single command (e.g., `gart add ~/.config/nvim`)
- **Automatic Updates**: Use the update command to detect changes in your dotfiles and backup them automatically (e.g., `gart update`)
- **Quick Overview**: List select and remove the dotfiles currently being managed with `gart list`
- **Flexible Naming**: Optionally assign custom names to your dotfiles for easier management

![Demo Deploy](assets/demo.gif?raw=true)

## Installation

### Prerequisites

- Linux
- Go >= 1.22

### Option 1: One-liner Makefile Installation
You can install Gart using this one-liner, which clones the repository, builds the binary, and installs it:

```bash
git clone https://github.com/bnema/gart.git && cd gart && make && sudo make install
```
   Note: This method requires sudo privileges to move the binary to the /usr/bin directory.

### Option 2: Installing with Go Install
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

To update a specific dotfile, use the `update` command followed by the name of the dotfile:
```
gart update nvim
```
This will detect changes in the specified dotfile and save the updated version to your designated store directory.

To update all the dotfiles specified in the `config.toml` file, simply run:
```
gart update
```

To list all the dotfiles currently being managed by Gart, use the `list` command:
```
gart list
```
This will display a list of all the dotfiles specified in the `config.toml` file.
## Configuration

The `config.toml` file is automatically created by Gart if it doesn't exist. It allows you to specify the dotfiles you want to manage. Each entry in the file represents a dotfile, with the key being the name of the dotfile and the value being the path to the dotfile on your local system.

Example:

```toml
[dotfiles]
nvim = "~/.config/nvim"
zsh = "~/.zshrc"
```

## Roadmap
- [x] Allow adding a single file
- [ ] Create a state with git after each detected change
- [ ] Custom store set in the config file
- [x] Remove a dotfile from the list view
- [ ] Status command to display the status of all the dotfiles
- [x] Version command

## License

This project is licensed under the [MIT License](LICENSE).
