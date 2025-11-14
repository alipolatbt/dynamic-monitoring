# Doğuş Teknoloji API Monitoring

Modern, lightweight synthetic API monitoring tool with dynamic token acquisition, email alerts, and NewRelic-like dashboard.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8.svg)
![React](https://img.shields.io/badge/React-18.2-61DAFB.svg)

## ✨ Features

- 🔄 **Dynamic Token Acquisition**: Automatically fetch auth tokens and use them in subsequent requests
- 📊 **Real-time Dashboard**: NewRelic-inspired UI with Material-UI components
- 📧 **Email Alerts**: SMTP-based email notifications when checks fail
- 🎯 **Multi-step API Monitoring**: Support for complex API flows (auth → API calls)
- 🔍 **Manual Triggers**: Instantly run any check on-demand
- 💾 **Persistent Configuration**: SMTP settings survive container restarts
- 🚀 **Production Ready**: Docker Compose & Kubernetes manifests included
- 🌍 **Multi-arch Support**: amd64 and arm64 images via GitHub Actions

## 🚀 Quick Start

### Development

```bash
# Clone the repository
git clone <your-repo-url>
cd dynamic_monitoring

# Build and start services
docker-compose up -d

# Access the UI
open http://localhost:3000
```

### Production

See [DEPLOYMENT.md](./DEPLOYMENT.md) for detailed deployment instructions.

## 📸 Screenshots

The dashboard provides:
- Active checks management with play/delete controls
- Real-time results table with error highlighting
- Multi-step form for creating complex API monitoring flows
- SMTP configuration dialog for email alerts

## 🏗️ Architecture

```
┌─────────────┐         ┌──────────────┐
│  Frontend   │────────▶│   Backend    │
│  (React)    │   API   │   (Go/Gin)   │
│  Nginx:80   │         │   :8080      │
└─────────────┘         └──────────────┘
                              │
                              ▼
                        ┌──────────────┐
                        │  SMTP Server │
                        │  (Alerts)    │
                        └──────────────┘
```

**Backend**: Go 1.25 + Gin framework
- Token-based API monitoring with goroutines
- SMTP alerting with TLS support
- Thread-safe state management
- JSON file persistence for settings

**Frontend**: React 18 + TypeScript + Material-UI
- Light theme with compact design
- Multi-step forms for complex scenarios
- Real-time polling (5s intervals)
- Responsive layout

## 📋 API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/checks` | Create a new monitoring check |
| GET | `/api/checks` | List all active checks |
| GET | `/api/results` | Get recent check results (max 200) |
| POST | `/api/checks/:name/trigger` | Manually trigger a check |
| DELETE | `/api/checks/:name` | Delete a check and its results |
| GET | `/api/smtp` | Get SMTP settings (password masked) |
| POST | `/api/smtp` | Update SMTP settings |

## 🔧 Configuration

### Environment Variables

**Backend:**
- `GIN_MODE`: `debug` or `release` (default: debug)
- `SMTP_SETTINGS_PATH`: Path to SMTP settings file (default: `/data/smtp_settings.json`)

**Frontend:**
- `REACT_APP_API_URL`: Backend API URL (default: `http://backend:8080`)

## 📦 Docker Images

Images are available on Docker Hub:
- `<username>/dynamic-monitoring-backend`
- `<username>/dynamic-monitoring-frontend`

To build and publish your own images:
```powershell
docker login
.\scripts\publish-images.ps1 -Username <your-dockerhub-username>
```

## 🔐 Security Considerations

⚠️ **Important**: This is a monitoring tool intended for internal use. Consider these security aspects:

1. **SMTP Credentials**: Stored in plain text in `/data/smtp_settings.json`. Use Docker secrets or Kubernetes Secrets in production.
2. **No Authentication**: The UI has no built-in auth. Use reverse proxy authentication or network isolation.
3. **CORS**: Backend allows all origins. Restrict in production.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License.

## 🆘 Support

For issues and questions:
- Create an issue in the GitHub repository
- See [DEPLOYMENT.md](./DEPLOYMENT.md) for troubleshooting

---

Built with ❤️ for Doğuş Teknoloji
