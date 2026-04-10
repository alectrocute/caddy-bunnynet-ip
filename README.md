# trusted_proxy module for bunny.net

This module retrieves cloudflare ips from their offical website, [ipv4]([https://www.cloudflare.com/ips-v4](https://api.bunny.net/mc/nodes/plain)). ipv6 is not currently supported by bunny.net as of August 2026. This module is supported from Caddy v2.6.3 onwards.

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
| interval | How often cloudflare ip lists are retrieved | duration | 1h         |
| timeout  | Maximum time to wait to get a response      | duration | no timeout |
