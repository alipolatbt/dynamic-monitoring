# Doğuş Teknoloji API Monitoring - Deployment Guide

Bu rehber, monitoring uygulamasını farklı ortamlarda nasıl çalıştıracağınızı açıklar.

## 🚀 Hızlı Başlangıç (Development)

```powershell
# Tüm servisleri build et ve çalıştır
docker-compose up -d

# UI'yi aç
http://localhost:3000
```

## 📦 Docker Hub'a Yayınlama

### Manuel Yayınlama

```powershell
# 1. Docker Hub'a giriş yap
docker login

# 2. Build et (eğer henüz etmediysen)
docker-compose build

# 3. Publish script'ini çalıştır
.\scripts\publish-images.ps1 -Username "<docker-hub-username>" -Version "1.0.0"
```

### GitHub Actions ile Otomatik Yayınlama

1. GitHub repo ayarlarından Secrets ekle:
   - `DOCKERHUB_USERNAME`: Docker Hub kullanıcı adınız
   - `DOCKERHUB_TOKEN`: Docker Hub access token ([buradan oluştur](https://hub.docker.com/settings/security))

2. Tag oluştur ve push et:
```powershell
git tag v1.0.0
git push origin v1.0.0
```

3. GitHub Actions otomatik olarak multi-arch (amd64/arm64) build yapıp Docker Hub'a push eder.

## 🌐 Production Deployment

### Docker Compose ile

1. `.env` dosyası oluştur:
```powershell
Copy-Item .env.example .env
```

2. `.env` dosyasını düzenle:
```env
BACKEND_IMAGE=<docker-hub-username>/dynamic-monitoring-backend:1.0.0
FRONTEND_IMAGE=<docker-hub-username>/dynamic-monitoring-frontend:1.0.0
```

3. Production compose ile başlat:
```powershell
docker compose -f docker-compose.prod.yml up -d
```

### Kubernetes Deployment

1. Manifest dosyalarını düzenle:
   - `k8s/backend-deployment.yaml` içindeki `<username>` kısmını değiştir
   - `k8s/frontend-deployment.yaml` içindeki `<username>` kısmını değiştir

2. Deploy et:
```powershell
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/backend-deployment.yaml
kubectl apply -f k8s/backend-service.yaml
kubectl apply -f k8s/frontend-deployment.yaml
kubectl apply -f k8s/frontend-service.yaml
kubectl apply -f k8s/ingress.yaml
```

3. Servis durumunu kontrol et:
```powershell
kubectl get pods -n monitoring
kubectl get svc -n monitoring
```

## 🔧 Özellikler

### SMTP Ayarlarının Kalıcılığı

- SMTP ayarları backend container içinde `/data/smtp_settings.json` dosyasına kaydedilir
- Docker Compose: `backend_data` named volume ile kalıcıdır
- Kubernetes: Varsayılan olarak `emptyDir` kullanır (pod restart'ta silinir)
  - Kalıcılık için PersistentVolumeClaim kullanabilirsiniz

### Environment Variables

Backend:
- `GIN_MODE`: `debug` (development) veya `release` (production)
- `SMTP_SETTINGS_PATH`: SMTP ayarlar dosyasının konumu (varsayılan: `/data/smtp_settings.json`)

## 📊 Monitoring & Logs

```powershell
# Docker Compose logs
docker-compose logs -f backend
docker-compose logs -f frontend

# Kubernetes logs
kubectl logs -n monitoring -f deployment/backend
kubectl logs -n monitoring -f deployment/frontend
```

## 🔒 Güvenlik Notları

1. **SMTP Parolası**: `/data/smtp_settings.json` dosyasında düz metin olarak saklanır. Production'da bir secret store (Vault, K8s Secret) kullanmanız önerilir.

2. **CORS**: Backend tüm origin'lere açık. Production'da `Access-Control-Allow-Origin` değerini kısıtlayın.

3. **Frontend Auth**: Şu an authentication yok. Nginx basic auth veya reverse proxy ile koruma ekleyebilirsiniz.

## 🆘 Troubleshooting

### Frontend backend'e bağlanamıyor

- Docker Compose: Backend'in `backend` host adıyla erişilebilir olduğundan emin olun
- Kubernetes: Backend Service adının `backend` olduğunu doğrulayın
- Nginx config: `frontend/nginx.conf` içinde `proxy_pass http://backend:8080/api/` kontrolü yapın

### SMTP ayarları kayboldu

- Docker Compose: `docker volume ls` ile `backend_data` volume'ünün var olduğunu kontrol edin
- Volume silindiyse SMTP ayarlarını UI'den tekrar girin

### Image pull hatası

- Docker Hub'da image'ların public olduğundan emin olun
- Veya Kubernetes için `imagePullSecrets` ekleyin
