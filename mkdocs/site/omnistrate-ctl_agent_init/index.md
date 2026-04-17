## omnistrate-ctl agent init

Initialize Claude Code skills and agent instructions for Omnistrate

### Synopsis

Initializes Claude Code skills and agent instructions for Omnistrate. This command will:

1. Clone the agent-instructions repository (or use local directory)
1. Copy skills to .claude/skills/ directory
1. Merge AGENTS.md and CLAUDE.md into your project

```text
omnistrate-ctl agent init [flags]
```

### Options

```text
  -h, --help                        help for init
      --instruction-source string   Path to local agent-instructions directory (default: clones from GitHub)
```

### Options inherited from parent commands

```text
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

- [omnistrate-ctl agent](../omnistrate-ctl_agent/) - Manage AI agent configurations and skills
