# Production Deployment Checklist

## 📋 Hazırlık Aşaması

### 1. API İzinleri ve Bilgileri
- [ ] Related Digital hesap bilgileri alındı (username/password)
- [ ] API endpoint'leri doğrulandı
- [ ] Rate limiting kuralları öğrenildi
- [ ] IP whitelist varsa sunucu IP'si eklendi
- [ ] Test environment ile deneme yapıldı
- [ ] Production credentials hazır

### 2. SMTP Email Ayarları
- [ ] SMTP server adresi belirlendi
- [ ] SMTP port numarası (587/465/25) belirlendi
- [ ] Email authentication bilgileri hazır
- [ ] Test email gönderimi başarılı
- [ ] Firewall'da SMTP portları açık
- [ ] Gönderici email adresi belirlendi
- [ ] Alarm alacak email adresleri belirlendi

### 3. Sunucu/Hosting Ortamı
- [ ] Docker ve Docker Compose kurulu
- [ ] Minimum 2GB RAM, 2 CPU core
- [ ] Disk space: en az 5GB boş alan
- [ ] Port 8080 (backend) ve 3000 (frontend) açık
- [ ] SSL/TLS sertifikası varsa nginx config güncellendi
- [ ] Firewall kuralları ayarlandı

### 4. Güvenlik
- [ ] SMTP credentials environment variable veya secret olarak saklanıyor
- [ ] Frontend için basic auth veya IP restriction eklendi (opsiyonel)
- [ ] Backend CORS ayarları production için kısıtlandı
- [ ] SSL/TLS sertifikası kuruldu (HTTPS için)
- [ ] Log dosyaları için retention policy belirlendi

### 5. Monitoring ve Backup
- [ ] Docker volume backup stratejisi belirlendi (/data için)
- [ ] Container restart policy: unless-stopped
- [ ] Log monitoring ayarlandı (opsiyonel: ELK, Grafana)
- [ ] Health check endpoint'leri test edildi
- [ ] Alert email'leri test edildi

## 🚀 Deployment Adımları

### 1. Sunucuda Kurulum
```bash
# Docker ve Docker Compose kurulu mu kontrol et
docker --version
docker compose version

# Projeyi sunucuya kopyala (git clone veya scp)
git clone https://github.com/alipolatbt/dynamic-monitoring.git
cd dynamic-monitoring
```

### 2. Environment Ayarları
```bash
# .env dosyası oluştur
cp .env.example .env

# .env dosyasını düzenle
nano .env

# İçerik:
BACKEND_IMAGE=alpolatdcoker/dynamic-monitoring-backend:v1.0.1
FRONTEND_IMAGE=alpolatdcoker/dynamic-monitoring-frontend:v1.0.1
```

### 3. Production Compose ile Başlat
```bash
# Image'ları pull et
docker compose -f docker-compose.prod.yml pull

# Başlat
docker compose -f docker-compose.prod.yml up -d

# Log'ları kontrol et
docker compose -f docker-compose.prod.yml logs -f
```

### 4. SMTP Ayarlarını Yap
1. http://<sunucu-ip>:3000 adresine git
2. **SMTP Settings** butonuna tıkla
3. SMTP bilgilerini gir:
   - Host: smtp.gmail.com (veya kurumsal SMTP)
   - Port: 587
   - Username: email@example.com
   - Password: app-password-veya-email-sifresi
   - From: gonderici@example.com
   - To: alici@example.com
   - Enable: ✅ işaretle
4. **Save Settings** butonuna tıkla

### 5. İlk Check'i Oluştur
1. **Add Check** butonuna tıkla
2. Related Digital senaryosunu gir:
   - **Step 1 - Authentication:**
     - URL: https://rdportalservice.relateddigital.com/auth/login
     - Method: POST
     - Token Path: `ServiceTicket`
     - Body: `{"username": "...", "password": "..."}`
   - **Step 2 - PostHtml API:**
     - URL: https://rdportalservice.relateddigital.com/notification/api/PostHtml
     - Method: POST
     - Token Header: `ServiceTicket`
   - **Step 3 - Monitor Settings:**
     - Check Name: Related Digital Notification
     - Interval: 60 seconds
     - Timeout: 10 seconds
3. **Create Check** butonuna tıkla

### 6. Test Et
- [ ] Manuel trigger ile check'i test et (Play butonu)
- [ ] Results tablosunda sonuçları gör
- [ ] Bir hata oluşturup email alındığını doğrula
- [ ] Container restart sonrası SMTP ayarlarının kaldığını kontrol et
- [ ] Delete butonunun çalıştığını test et

## 🔧 Troubleshooting

### SMTP Email Gönderilmiyor
```bash
# Backend log'larını kontrol et
docker compose -f docker-compose.prod.yml logs backend | grep SMTP

# Yaygın hatalar:
# - "connection refused": SMTP port firewall'da kapalı
# - "authentication failed": Yanlış username/password
# - "TLS handshake failed": Port 587 yerine 465 deneyin veya vice versa
```

### API Check Başarısız
```bash
# Backend log'larını kontrol et
docker compose -f docker-compose.prod.yml logs backend | grep ERROR

# Related Digital API'ye erişim var mı test et
curl -X POST https://rdportalservice.relateddigital.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"test"}'
```

### Container Restart Sonrası SMTP Ayarları Kayboluyor
```bash
# Volume'un mount edildiğini kontrol et
docker volume ls | grep backend_data

# Volume içeriğini kontrol et
docker exec -it dynamic-monitoring-backend sh
ls -la /data/smtp_settings.json
cat /data/smtp_settings.json
```

## 📊 Production Best Practices

### 1. Nginx Reverse Proxy (Opsiyonel ama Önerilen)
```nginx
# /etc/nginx/sites-available/monitoring
server {
    listen 80;
    server_name monitoring.dogustech.com;

    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### 2. SSL/TLS Sertifikası (Let's Encrypt)
```bash
# Certbot kurulumu
sudo apt install certbot python3-certbot-nginx

# Sertifika oluştur
sudo certbot --nginx -d monitoring.dogustech.com
```

### 3. Otomatik Backup
```bash
# Backup script oluştur: /opt/backup-monitoring.sh
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
docker run --rm -v dynamic_monitoring_backend_data:/data \
  -v /opt/backups:/backup alpine \
  tar czf /backup/smtp_settings_$DATE.tar.gz -C /data .

# Crontab ekle
crontab -e
# Her gün saat 02:00'de backup
0 2 * * * /opt/backup-monitoring.sh
```

### 4. Auto-restart on Boot
```bash
# Systemd service oluştur: /etc/systemd/system/monitoring.service
[Unit]
Description=Dogus Monitoring
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/dynamic-monitoring
ExecStart=/usr/bin/docker compose -f docker-compose.prod.yml up -d
ExecStop=/usr/bin/docker compose -f docker-compose.prod.yml down

[Install]
WantedBy=multi-user.target

# Enable
sudo systemctl enable monitoring
```

## ✅ Final Checklist

Deployment tamamlandıktan sonra:
- [ ] UI açılıyor ve erişilebilir
- [ ] Check oluşturuluyor ve çalışıyor
- [ ] Manuel trigger çalışıyor
- [ ] Results görünüyor
- [ ] Email alerts geliyor
- [ ] Delete butonu çalışıyor
- [ ] Container restart sonrası ayarlar korunuyor
- [ ] Backup stratejisi çalışıyor
- [ ] SSL/TLS sertifikası kurulu (opsiyonel)
- [ ] Dokümantasyon ekiple paylaşıldı

## 📞 İletişim

Sorun olursa:
- GitHub Issues: https://github.com/alipolatbt/dynamic-monitoring/issues
- README: https://github.com/alipolatbt/dynamic-monitoring
- DEPLOYMENT.md: Detaylı deployment rehberi
