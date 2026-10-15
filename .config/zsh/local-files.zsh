# Per-machine files (never in git) <-> Bitwarden, see ~/.config/dotfiles/local-files.sh
#   local-backup              save this machine's files
#   local-restore [machine]   bring them back (keeps existing files; --force overwrites)
alias local-backup='~/.config/dotfiles/local-files.sh backup'
alias local-restore='~/.config/dotfiles/local-files.sh restore'
