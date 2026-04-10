# trusted_proxy module for bunny.net

This module retrieves bunny.net CDN IP addresses from their [/mc/nodes/plain](https://api.bunny.net/mc/nodes/plain) API endpoint.

Note: IPv6 is not currently supported by bunny.net as of August 2026. This module is supported from Caddy v2.6.3 onwards.

# Example config

Put following config in global options under corresponding server options:

```
trusted_proxies bunnynet {
    interval 12h
    timeout 15s
}
```

# Defaults

| Name     | Description                                 | Type     | Default    |
|----------|---------------------------------------------|----------|------------|
| interval | How often lists are retrieved               | duration | 1h         |
| timeout  | Maximum time to wait to get a response      | duration | no timeout |
