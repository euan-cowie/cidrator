#!/bin/bash

_cidrator_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    local commands="cidr dns scan fw version help"
    
    # CIDR subcommands
    local cidr_commands="explain expand contains count overlaps divide"
    
    # DNS subcommands
    local dns_commands="lookup reverse"
    
    # Scan subcommands
    local scan_commands="ping port"
    
    # Firewall subcommands
    local fw_commands="analyze generate"
    
    # Global flags
    local global_flags="--help --config"
    
    # CIDR flags
    local cidr_flags="--format --limit --one-line"

    case ${COMP_CWORD} in
        1)
            # Complete main commands
            COMPREPLY=($(compgen -W "${commands}" -- ${cur}))
            return 0
            ;;
        2)
            # Complete subcommands based on main command
            case ${prev} in
                cidr)
                    COMPREPLY=($(compgen -W "${cidr_commands}" -- ${cur}))
                    return 0
                    ;;
                dns)
                    COMPREPLY=($(compgen -W "${dns_commands}" -- ${cur}))
                    return 0
                    ;;
                scan)
                    COMPREPLY=($(compgen -W "${scan_commands}" -- ${cur}))
                    return 0
                    ;;
                fw)
                    COMPREPLY=($(compgen -W "${fw_commands}" -- ${cur}))
                    return 0
                    ;;
            esac
            ;;
        *)
            # Complete flags for specific commands
            case ${prev} in
                --format)
                    COMPREPLY=($(compgen -W "table json yaml" -- ${cur}))
                    return 0
                    ;;
                --limit)
                    # No completion for numeric values
                    return 0
                    ;;
                explain|expand|contains|count|overlaps|divide)
                    # Complete CIDR flags
                    COMPREPLY=($(compgen -W "${cidr_flags}" -- ${cur}))
                    return 0
                    ;;
            esac
            ;;
    esac

    # Default completion
    COMPREPLY=($(compgen -W "${global_flags}" -- ${cur}))
}

complete -F _cidrator_completion cidrator 