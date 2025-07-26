#compdef cidrator

_cidrator() {
    local context state line
    typeset -A opt_args

    _arguments -C \
        '1: :_cidrator_commands' \
        '*:: :->args' \
        && return 0

    case $state in
        args)
            case $words[1] in
                cidr)
                    _arguments -C \
                        '1: :_cidrator_cidr_commands' \
                        '*:: :->cidr_args'
                    case $state in
                        cidr_args)
                            case $words[1] in
                                explain|expand|count)
                                    _arguments \
                                        '--format[Output format]:format:(table json yaml)' \
                                        '--limit[Maximum number of items]:limit:' \
                                        '--one-line[Output on one line]'
                                    ;;
                                contains|overlaps)
                                    _arguments \
                                        '*:CIDR or IP:'
                                    ;;
                                divide)
                                    _arguments \
                                        '*:CIDR and number:'
                                    ;;
                            esac
                            ;;
                    esac
                    ;;
                dns)
                    _arguments -C \
                        '1: :_cidrator_dns_commands' \
                        '*:: :->dns_args'
                    ;;
                scan)
                    _arguments -C \
                        '1: :_cidrator_scan_commands' \
                        '*:: :->scan_args'
                    ;;
                fw)
                    _arguments -C \
                        '1: :_cidrator_fw_commands' \
                        '*:: :->fw_args'
                    ;;
                version)
                    # No additional arguments for version
                    ;;
                help)
                    _arguments \
                        '1: :_cidrator_commands'
                    ;;
            esac
            ;;
    esac
}

_cidrator_commands() {
    local commands
    commands=(
        'cidr:CIDR network analysis and manipulation'
        'dns:DNS analysis and lookup tools'
        'scan:Network scanning and discovery'
        'fw:Firewall rule generation and analysis'
        'version:Print version information'
        'help:Show help for commands'
    )
    _describe 'commands' commands
}

_cidrator_cidr_commands() {
    local commands
    commands=(
        'explain:Show detailed information about a CIDR range'
        'expand:List all IP addresses in a CIDR range'
        'contains:Check if an IP address is in a CIDR range'
        'count:Count total addresses in a CIDR range'
        'overlaps:Check if two CIDR ranges overlap'
        'divide:Divide a CIDR range into smaller subnets'
    )
    _describe 'cidr commands' commands
}

_cidrator_dns_commands() {
    local commands
    commands=(
        'lookup:Perform DNS lookups'
        'reverse:Perform reverse DNS lookups'
    )
    _describe 'dns commands' commands
}

_cidrator_scan_commands() {
    local commands
    commands=(
        'ping:Ping sweep across network ranges'
        'port:Scan ports on target hosts'
    )
    _describe 'scan commands' commands
}

_cidrator_fw_commands() {
    local commands
    commands=(
        'analyze:Analyze firewall configurations'
        'generate:Generate firewall rules for CIDR ranges'
    )
    _describe 'fw commands' commands
}

_cidrator "$@" 