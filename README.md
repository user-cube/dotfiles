# Dotfiles

This repo contains configuration files and scripts for various tools and environments.

## Directory Structure

- `.config/`: Contains configuration files for Neovim and other tools.
- `.warp/`: Contains theme files.
- `iTerm2/`: Contains configuration for iTerm2.
- `.p10k.zsh`: Configuration file for Powerlevel10k, a theme for Zsh.

## Configuration Details

### Neovim

Configuration files for Neovim are located in `.config/nvim/`. The main configuration file is `init.lua`.

### iTerm2

Configuration for iTerm2 is located in `iTerm2/iTerm2Profile.json`.

### Powerlevel10k

Configuration for Powerlevel10k is located in `.p10k.zsh`.

## Themes

Theme files for various tools are located in `.warp/themes/`.

## Other Files

- `.gitignore`: Specifies which files and directories to ignore in Git.
- `.zshrc`: Configuration file for Zsh.

## Automated Configuration

This workspace uses an automated method to place these files and configurations using Ansible. The Ansible playbooks for this automation can be found at [this GitHub repository](https://github.com/user-cube/ansible-playbooks/tree/main/macos).

These playbooks handle the setup and configuration of the tools and environments mentioned above, ensuring a consistent and reproducible configuration across different machines.