# Deploying Andesite via Caddy

1. Setup a "development" environment using the installation instructions in [`README.md`](../README.md)
2. Make sure Andesite is visible from http://localhost:8000
3. Download Caddy (https://caddyserver.com/download)
4. Configure a new site context for Andesite such as:
```caddy
andesite.example.com {
    proxy / http://localhost:8000/ {
        transparent
    }
}
```

### Using HTTPS
Add the following option to your ``proxy`` block:
```caddy
header_upstream X-TLS-Enabled true
```

### Serving from an HTTP base that is not `/`
```caddy
example.com {
    proxy /andesite http://localhost:8000/ {
        transparent
    }
}
```
Notes:
- The leading slash at the end of the `proxy` directive is critical, particularly if you are serving Andesite from a URL that isn't `/`.
- The `--base` option must be sent with the exact text of the Caddy location. Ie: `./andesite --root ROOT --base /andesite/`.
- If the exposed port is not `80` or `443`, then the `header_upstream` directive value must be `Host $host:$server_port`.
- Your OAuth2 callback URL must the full accessible location of `ANDESITE/callback`.
