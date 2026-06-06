# Buyahref Payment Hub

Central payment system for `buyahref.com` — accepts PhonePe payments for multiple merchant sites.

**GitHub:** https://github.com/sagartiwari-net/payment

---

## VPS Deploy (GitHub se — root password ki zaroorat nahi)

### Step 1: phpMyAdmin — tables banao

1. phpMyAdmin kholo
2. Database **`paymentsystem`** select karo
3. **SQL** tab → file open karo:  
   `payment-hub/migrations/install_all_tables.sql`
4. **Go** click karo

### Step 2: VPS par code clone karo

SSH se login karo (normal user — root nahi chahiye):

```bash
cd ~
git clone https://github.com/sagartiwari-net/payment.git
cd payment/payment-hub
```

**cPanel hai to:** Git Version Control → Clone → URL paste karo.

### Step 3: `.env` file banao (secrets yahan — GitHub par nahi)

```bash
cp env.production.template .env
nano .env
```

Yeh values set karo:

```env
APP_ENV=production
APP_PORT=8090
APP_URL=https://buyahref.com

DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=paymentsystem
DB_USER=paymentsystem
DB_PASSWORD=apna_db_password

PHONEPE_MERCHANT_ID=BUYAONLINE
PHONEPE_SALT_KEY=apna_salt_key
PHONEPE_SALT_INDEX=1
PHONEPE_ENV=PRODUCTION

JWT_SECRET=koi_bhi_random_long_string
LOG_LEVEL=info
```

Save: `Ctrl+O` → Enter → `Ctrl+X`

### Step 4: Build aur start

```bash
chmod +x scripts/vps-deploy.sh
./scripts/vps-deploy.sh
./bin/payment-hub
```

Background mein chalane ke liye:

```bash
nohup ./bin/payment-hub > payment-hub.log 2>&1 &
```

### Step 5: Test

VPS par:

```bash
curl http://127.0.0.1:8090/health
```

Expected:

```json
{"status":"ok","service":"payment-hub","env":"production","database":"ok"}
```

Browser se (Nginx setup ke baad):

```
https://buyahref.com/health
```

---

## Code update karna (Git pull)

```bash
cd ~/payment/payment-hub
git pull
go build -o bin/payment-hub ./cmd/server
# server restart karo
pkill payment-hub
nohup ./bin/payment-hub > payment-hub.log 2>&1 &
```

---

## Project structure

```
payment/
├── IMPLEMENTATION_PLAN.md    # Full technical plan
├── README.md                 # This file
└── payment-hub/              # Go backend
    ├── cmd/server/           # Entry point
    ├── migrations/           # MySQL tables
    ├── env.production.template
    └── scripts/vps-deploy.sh
```

---

## Security note

- **`.env` kabhi GitHub par push mat karo** — passwords wahan nahi jaate
- Repo **Private** rakho recommended
- Salt key / DB password leak ho to turant rotate karo

---

## Local dev (optional — Mac)

Docker Desktop install karke:

```bash
cd payment-hub
make docker-up
make migrate-docker
make run
curl http://localhost:8090/health
```
