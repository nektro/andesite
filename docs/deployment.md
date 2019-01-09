# Deploying Andesite

## Nginx
1. Setup a "development" environment using the installation instructions in [`README.md`](../README.md)
2. Make sure Andesite is visible from http://localhost:8000
3. Download Nginx (https://nginx.org/en/download.html)
4. Configure a `location` context for Andesite such as:
```
location / {
    proxy_pass http://localhost:8000/;
}
```
> Note: the leading slash at the end of `proxy_pass` is critical, particularly if you are serving Andesite from a `location` that isn't `/`
