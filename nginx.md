To install **NGINX** and configure it to **forward traffic to a local port** (e.g., your Go app on `localhost:9051`), follow these steps. This works on **Ubuntu**, **Debian**, and similar distributions.

---

## ✅ Step 1: Install NGINX

```bash
sudo apt update
sudo apt install nginx -y
```

---

## ✅ Step 2: Configure NGINX to Proxy Requests

Let’s say your app is running on `http://localhost:9051`.

### Option A: Using `/etc/nginx/sites-available` (Ubuntu/Debian Style)

1. **Create a new config file**:

```bash
nano /etc/nginx/sites-available/halmidi
```

2. **Paste this config** (change port if needed):

```nginx
server {
    listen 80;
    server_name _;

    client_max_body_size 10M;

    location / {
        proxy_pass http://127.0.0.1:9051;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

3. **Enable the site**:

```bash
sudo ln -s /etc/nginx/sites-available/halmidi /etc/nginx/sites-enabled/
rm /etc/nginx/sites-enabled/default  # Remove nginx default home page
```

4. **Test the config**:

```bash
sudo nginx -t
```

If you see:

```
nginx: configuration file /etc/nginx/nginx.conf test is successful
```

You're good to go.

5. **Restart NGINX**:

```bash
sudo systemctl restart nginx
```
