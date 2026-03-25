## omnistrate-ctl alarms notification-channel event-history

Show event history for a notification channel with interactive TUI

### Synopsis

Display event history for a notification channel in an interactive table interface that allows expanding rows to see event details.

```
omnistrate-ctl alarms notification-channel event-history [channel-id] [flags]
```

### Options

```
  -e, --end-time string     End time for event history (RFC3339 format)
  -h, --help                help for event-history
  -s, --start-time string   Start time for event history (RFC3339 format)
```

### Options inherited from parent commands

```
  -o, --output string   Output format (text|table|json) (default "table")
  -v, --version         Print the version number of omnistrate-ctl
```

### SEE ALSO

* [omnistrate-ctl alarms notification-channel](omnistrate-ctl_alarms_notification-channel.md)	 - Manage notification channels

