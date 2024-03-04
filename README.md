
# Gart - Dotfile Manager

Gart is a command-line tool written in Go that helps you manage and sync your dotfiles across different systems. With Gart, you can easily keep your configuration files up to date and maintain a consistent setup across multiple machines.

## Features
- Automatically detects changes in your dotfiles
- Syncs dotfiles between your local system and a designated store directory
- Supports copying entire directories and preserving file modes
- Provides a simple and intuitive command-line interface
- Auto-creates the configuration file if it doesn't exist
- Allows adding new dotfiles to the configuration
- Lists all the dotfiles currently being managed in a sexy table.
- TODO: Delete from the table
- TODO: Custom storage location and/or default OS configuration directory.

## Installation

### Prerequisites

- Go >= 1.22

### Installing via Makefile

1. Clone this repository:
   git clone https://github.com/bnema/gart.git
2. Navigate to the project directory:
```bash
 cd gart
```

3. Build and install the binary using the provided Makefile:
```bash
   make && sudo make install
```
   Note: This step requires sudo privileges to move the binary to the `/usr/bin` directory.

## Usage

2. To add a new dotfile to the configuration, use the `add` command to display the text inputs for the path and the name.
   ```
   gart add 
   ```

3. To update a specific dotfile, use the `update` command followed by the name of the dotfile:
   ```
   gart update neovim
   ```
   This will detect changes in the specified dotfile and save the updated version to your designated store directory.

4. To update all the dotfiles specified in the `config.toml` file, simply run:
   ```
   gart update
   ```

5. To list all the dotfiles currently being managed by Gart, use the `list` command:
   ```
   gart list
   ```
   This will display a list of all the dotfiles specified in the `config.toml` file.
## Configuration

The `config.toml` file is automatically created by Gart if it doesn't exist. It allows you to specify the dotfiles you want to manage. Each entry in the file represents a dotfile, with the key being the name of the dotfile and the value being the path to the dotfile on your local system.

Example:

```toml
[dotfiles]
neovim = "~/.config/nvim"
zsh = "~/.zshrc"
```

## License

This project is licensed under the [MIT License](LICENSE).
```

This README includes the full markdown formatting, including headings, code blocks, and links. It provides instructions for installing Gart using the provided Makefile or manually, along with usage examples and configuration details.

Feel free to customize the README further based on your specific project requirements and preferences.
