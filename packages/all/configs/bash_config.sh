# source generated configs if present
if [ -d ~/rc/configs-generated ]; then
    # bash_env first, so PATH entries in bash_paths can use the variables it exports.
    source ~/rc/configs-generated/bash_env
    source ~/rc/configs-generated/bash_paths
    source ~/rc/configs-generated/bash_aliases
    # bash_init is for interactive shells only (prompt, completions, key bindings,
    # shell hooks). Debian/Ubuntu's bash also reads ~/.bashrc for non-interactive
    # commands run over SSH (`ssh host cmd`, and the `bash -c` they start), which
    # must not run it. What those commands need goes in bash_env (exports) and
    # bash_paths (PATH entries).
    case $- in
        *i*) source ~/rc/configs-generated/bash_init ;;
    esac
fi
