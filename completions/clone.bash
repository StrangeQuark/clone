_clone() {
  local current previous
  current="${COMP_WORDS[COMP_CWORD]}"
  previous="${COMP_WORDS[COMP_CWORD-1]}"

  case "$previous" in
    --into) COMPREPLY=( $(compgen -d -- "$current") ); return 0 ;;
    --domain|--username) COMPREPLY=(); return 0 ;;
    update) COMPREPLY=( $(compgen -W 'domain username' -- "$current") ); return 0 ;;
  esac
  COMPREPLY=( $(compgen -W 'init update config --help --version --ssh --into --background' -- "$current") )
}
complete -F _clone clone
