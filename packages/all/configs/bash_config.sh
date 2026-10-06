# source generated configs if present
if [ -d ~/rc/configs-generated ]; then
    source ~/rc/configs-generated/bash_paths
    source ~/rc/configs-generated/bash_aliases
    # bash_init is for interactive shells only (prompt, completions, key bindings,
    # shell hooks). Debian/Ubuntu's bash also reads ~/.bashrc for non-interactive
    # commands run over SSH (`ssh host cmd`, and the `bash -c` they start), which
    # must not run it. What those commands need (PATH, exports) goes in bash_paths.
    case $- in
        *i*) source ~/rc/configs-generated/bash_init ;;
    esac
fi
