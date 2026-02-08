# Documentation

Here you can find configuration file references for:

- [repos.yaml](repos.yaml)
- [stacks.yaml](stacks.yaml)
- [config.yaml](config.yaml)
- [webhook.md](webhook.md)

## Web UI

The SwarmCD web UI provides a dashboard to monitor your stacks.

### Manual Refresh

The UI includes a manual refresh button (sync icon) in the header bar. Unlike automatic polling, this gives you control over when to fetch and display updates:

1. **Check for updates**: Click the refresh button to fetch the latest stack statuses from the server
2. **Update indicator**: If there are changes, a green pulsing badge appears on the button and the icon turns green
3. **Apply update**: Click the button again to apply the update and refresh the displayed data

This approach lets you review the current state before applying updates, avoiding unexpected UI changes while you're viewing the dashboard.
