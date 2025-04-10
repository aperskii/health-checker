# 🔍 Health Checker

A lightweight and configurable application written in **Go**, designed to automate the monitoring of application endpoints. It authenticates via **OAuth2**, checks the health of applications through their APIs, parses JSON responses, and notifies responsible developers via email in case of errors.

---

## 📌 Features

- ✅ Periodic health checks of configured applications
- 🔐 OAuth2 integration for secure API access
- 🧠 Intelligent error detection based on JSON responses
- 📧 Email notifications to assigned developers on failure
- ⚙️ Configurable via JSON/YAML (clients, endpoints, workers, intervals)
- 🚀 Concurrent checks using a worker pool
- 🪵 Optional log inspection and status tracking

---

## ⚙️ How It Works

1. **OAuth2 Authentication**:  
   The app authenticates with an Auth Server using client ID/secret and retrieves an access token.

2. **Endpoint Health Check**:  
   The token is used to call each application endpoint periodically (configurable interval).

3. **Error Detection**:  
   If the JSON response includes specific error codes/fields, it’s flagged.

4. **Email Notification**:  
   If a new error is found, an email is sent to the assigned recipients. Duplicate alerts for the same issue are suppressed for a configured duration.

5. **Concurrency**:  
   A worker pool handles multiple application checks in parallel for performance.

---

## 🧾 Configuration

All settings are defined in a config file (`config.json` or `config.yaml`):

```json
{
  "auth_clients": [
    {
      "client_id": "my-id",
      "client_secret": "my-secret",
      "token_url": "https://auth.example.com/token",
      "applications": [
        {
          "name": "App1",
          "endpoint": "https://app1.example.com/health",
          "check_interval_seconds": 60
        }
      ]
    }
  ],
  "worker_pool_size": 5,
  "log_email": ["dev@example.com"],
  "error_repeat_interval": 3600
}


🚀 Getting Started

# Clone repository
git clone https://github.com/aperskii/health-checker.git
cd health-checker

# Build
go build -o health-checker

# Run
./health-checker --config=config.json


# Build
go build -o health-checker

# Run
./health-checker --config=config.json

📁 Folder Structure

/config       → Configuration files
/internal     → Core business logic
/cmd          → CLI entry point
/utils        → Helper packages (email, logging, etc.)
/tests        → Unit and integration tests


📧 Email Format

Each error email includes:

Application name

Timestamp

Error details from the JSON response

Link to application or log reference (if configured)

🧪 Testing

go test ./...


📄 License
MIT License – see LICENSE for details.

🙌 Contributing
Pull requests welcome! For major changes, please open an issue first to discuss what you would like to change.

🛠 Built With
Go

OAuth2

SMTP

JSON / HTTP / Worker Pools

