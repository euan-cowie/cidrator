# Fish completion for cidrator

# Main commands
complete -c cidrator -f -n '__fish_use_subcommand' -a 'cidr' -d 'CIDR network analysis and manipulation'
complete -c cidrator -f -n '__fish_use_subcommand' -a 'dns' -d 'DNS analysis and lookup tools'
complete -c cidrator -f -n '__fish_use_subcommand' -a 'scan' -d 'Network scanning and discovery'
complete -c cidrator -f -n '__fish_use_subcommand' -a 'fw' -d 'Firewall rule generation and analysis'
complete -c cidrator -f -n '__fish_use_subcommand' -a 'version' -d 'Print version information'
complete -c cidrator -f -n '__fish_use_subcommand' -a 'help' -d 'Show help for commands'

# Global flags
complete -c cidrator -l help -d 'Show help'
complete -c cidrator -l config -d 'Configuration file'

# CIDR subcommands
complete -c cidrator -f -n '__fish_seen_subcommand_from cidr' -a 'explain' -d 'Show detailed information about a CIDR range'
complete -c cidrator -f -n '__fish_seen_subcommand_from cidr' -a 'expand' -d 'List all IP addresses in a CIDR range'
complete -c cidrator -f -n '__fish_seen_subcommand_from cidr' -a 'contains' -d 'Check if an IP address is in a CIDR range'
complete -c cidrator -f -n '__fish_seen_subcommand_from cidr' -a 'count' -d 'Count total addresses in a CIDR range'
complete -c cidrator -f -n '__fish_seen_subcommand_from cidr' -a 'overlaps' -d 'Check if two CIDR ranges overlap'
complete -c cidrator -f -n '__fish_seen_subcommand_from cidr' -a 'divide' -d 'Divide a CIDR range into smaller subnets'

# CIDR flags
complete -c cidrator -n '__fish_seen_subcommand_from cidr; and __fish_seen_subcommand_from explain expand count' -l format -a 'table json yaml' -d 'Output format'
complete -c cidrator -n '__fish_seen_subcommand_from cidr; and __fish_seen_subcommand_from expand' -l limit -d 'Maximum number of IPs'
complete -c cidrator -n '__fish_seen_subcommand_from cidr; and __fish_seen_subcommand_from expand' -l one-line -d 'Output on one line'

# DNS subcommands
complete -c cidrator -f -n '__fish_seen_subcommand_from dns' -a 'lookup' -d 'Perform DNS lookups'
complete -c cidrator -f -n '__fish_seen_subcommand_from dns' -a 'reverse' -d 'Perform reverse DNS lookups'

# DNS flags
complete -c cidrator -n '__fish_seen_subcommand_from dns; and __fish_seen_subcommand_from lookup' -l type -a 'A AAAA MX TXT CNAME NS PTR' -d 'DNS record type'
complete -c cidrator -n '__fish_seen_subcommand_from dns; and __fish_seen_subcommand_from lookup' -l server -d 'DNS server to query'

# Scan subcommands
complete -c cidrator -f -n '__fish_seen_subcommand_from scan' -a 'ping' -d 'Ping sweep across network ranges'
complete -c cidrator -f -n '__fish_seen_subcommand_from scan' -a 'port' -d 'Scan ports on target hosts'

# Scan flags
complete -c cidrator -n '__fish_seen_subcommand_from scan; and __fish_seen_subcommand_from port' -l ports -d 'Port range to scan'
complete -c cidrator -n '__fish_seen_subcommand_from scan; and __fish_seen_subcommand_from port' -l threads -d 'Number of concurrent threads'
complete -c cidrator -n '__fish_seen_subcommand_from scan; and __fish_seen_subcommand_from port' -l udp -d 'Scan UDP ports'

# Firewall subcommands
complete -c cidrator -f -n '__fish_seen_subcommand_from fw' -a 'analyze' -d 'Analyze firewall configurations'
complete -c cidrator -f -n '__fish_seen_subcommand_from fw' -a 'generate' -d 'Generate firewall rules for CIDR ranges'

# Firewall flags
complete -c cidrator -n '__fish_seen_subcommand_from fw; and __fish_seen_subcommand_from generate' -l format -a 'iptables pf cisco juniper' -d 'Firewall format'
complete -c cidrator -n '__fish_seen_subcommand_from fw; and __fish_seen_subcommand_from generate' -l action -a 'allow deny' -d 'Default action'
complete -c cidrator -n '__fish_seen_subcommand_from fw; and __fish_seen_subcommand_from generate' -l protocol -a 'tcp udp icmp all' -d 'Protocol' 