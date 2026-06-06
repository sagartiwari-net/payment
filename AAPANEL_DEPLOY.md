# aaPanel Deploy — buyahref.com/payment

Path on server: `/www/wwwroot/buyahref.com/payment`

Public URL: `https://buyahref.com/payment/health`

> **Port 8090** — Payment Hub alag port use karta hai taaki `8080` par jo doosra Go project chal raha hai usse conflict na ho.

---

## Part 1 — phpMyAdmin (SQL Tables)

1. aaPanel login → **Database** → **phpMyAdmin**
2. Left side se database **`paymentsystem`** click karo
3. Top par **SQL** tab
4. Neeche wala poora SQL copy-paste karo → **Go**

Ya **Import** tab se file upload karo:
`/www/wwwroot/buyahref.com/payment/payment-hub/migrations/install_all_tables.sql`
(Git clone ke baad)

---

## Part 2 — aaPanel Terminal (Commands)

Terminal kholo (aaPanel → Terminal). Commands **order mein** chalao:

### 2.1 Folder + Git Clone

```bash
cd /www/wwwroot/buyahref.com

# Agar purana khali folder hai to hata do (optional)
# rm -rf payment

git clone https://github.com/sagartiwari-net/payment.git payment

cd /www/wwwroot/buyahref.com/payment/payment-hub
```

### 2.2 `.env` file banao

```bash
cat > /www/wwwroot/buyahref.com/payment/payment-hub/.env << 'EOF'
APP_ENV=production
APP_PORT=8090
APP_URL=https://buyahref.com/payment

DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=paymentsystem
DB_USER=paymentsystem
DB_PASSWORD=erF7nCaY4bkwxwEj

REDIS_URL=redis://127.0.0.1:6379

PHONEPE_MERCHANT_ID=BUYAONLINE
PHONEPE_SALT_KEY=26223eb5-ecec-47df-973b-0bc8dc7ed187
PHONEPE_SALT_INDEX=1
PHONEPE_ENV=PRODUCTION

JWT_SECRET=buyahref_payment_hub_jwt_change_later
LOG_LEVEL=info
EOF
```

### 2.3 Go install (pehli baar — skip if `go version` works)

```bash
go version
```

Agar "command not found" aaye:

```bash
cd /tmp
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
source /root/.bashrc
go version
```

### 2.4 Build

```bash
cd /www/wwwroot/buyahref.com/payment/payment-hub
go mod tidy
mkdir -p bin
go build -buildvcs=false -o bin/payment-hub ./cmd/server
chmod +x bin/payment-hub
```

### 2.5 Test run (temporary)

```bash
cd /www/wwwroot/buyahref.com/payment/payment-hub
./bin/payment-hub
```

Dusre terminal mein test:

```bash
curl http://127.0.0.1:8090/health
```

`"database":"ok"` aana chahiye. Stop: `Ctrl+C`

### 2.6 Background mein chalao (Supervisor — recommended)

aaPanel → **App Store** → **Supervisor** install karo (agar nahi hai)

Phir **Supervisor** → **Add Daemon**:

| Field | Value |
|-------|-------|
| Name | payment-hub |
| Run User | www |
| Run Dir | /www/wwwroot/buyahref.com/payment/payment-hub |
| Start Command | /www/wwwroot/buyahref.com/payment/payment-hub/bin/payment-hub |
| Processes | 1 |

Save → Start

**Ya terminal se (simple):**

```bash
cd /www/wwwroot/buyahref.com/payment/payment-hub
nohup ./bin/payment-hub > payment-hub.log 2>&1 &
```

---

## Part 3 — Nginx (buyahref.com/payment URL)

aaPanel → **Website** → **buyahref.com** → **Config** (Nginx config)

`server { ... }` block ke andar **location /** se pehle yeh add karo:

```nginx
    location /payment/ {
        proxy_pass http://127.0.0.1:8090/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Prefix /payment;
    }
```

Save → Nginx **Reload**

### Test URLs

```bash
curl https://buyahref.com/payment/health
```

Browser:

```
https://buyahref.com/payment/health
```

Expected:

```json
{"status":"ok","service":"payment-hub","env":"production","database":"ok"}
```

---

## Part 4 — SQL (phpMyAdmin copy-paste)

Git clone se pehle bhi chala sakte ho — yeh SQL phpMyAdmin mein paste karo:

File: `payment-hub/migrations/install_all_tables.sql`  
(GitHub: https://github.com/sagartiwari-net/payment/blob/main/payment-hub/migrations/install_all_tables.sql)

---

## Update (code change ke baad)

```bash
cd /www/wwwroot/buyahref.com/payment
git pull
cd payment-hub
go build -buildvcs=false -o bin/payment-hub ./cmd/server
```

Supervisor use kar rahe ho to **Restart** daemon.  
`nohup` use kiya ho to:

```bash
pkill -f payment-hub
cd /www/wwwroot/buyahref.com/payment/payment-hub
nohup ./bin/payment-hub > payment-hub.log 2>&1 &
```

---

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `Access denied` DB | `.env` password check; aaPanel → Database user `paymentsystem` |
| `connection refused` curl | `./bin/payment-hub` chal raha hai? `ps aux \| grep payment-hub` |
| 404 on /payment/health | Nginx config add kiya + reload? |
| 502 Bad Gateway | Go app port 8090 par run ho raha hai? |
| Permission denied | `chown -R www:www /www/wwwroot/buyahref.com/payment` |

---

## Folder structure (final)

```
/www/wwwroot/buyahref.com/payment/
├── README.md
├── IMPLEMENTATION_PLAN.md
└── payment-hub/
    ├── .env              ← secrets (manual)
    ├── bin/payment-hub   ← compiled binary
    ├── cmd/server/
    ├── migrations/
    └── payment-hub.log
```
