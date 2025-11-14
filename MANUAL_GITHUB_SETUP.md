## GitHub Repository Oluşturma - Adım Adım

### 1. GitHub'a Git
https://github.com/new

### 2. Formu Doldur
- **Repository name**: `dynamic-monitoring`
- **Description**: `Doğuş Teknoloji API Monitoring - Production-ready synthetic API monitoring`
- **Public** veya **Private** seçin
- ❌ **"Add a README file"** seçmeyin (zaten var)
- ❌ **".gitignore"** eklemeyin (zaten var)
- ❌ **"Choose a license"** seçmeyin

### 3. "Create repository" butonuna tıklayın

### 4. Repo oluştuktan sonra YENİ BİR POWERSHELL PENCERESI açın ve bu komutları çalıştırın:

```powershell
cd C:\dynamic_monitoring

# Remote kontrolü
git remote -v

# Eğer remote yoksa veya yanlışsa:
git remote remove origin
git remote add origin https://github.com/alipolatbt/dynamic-monitoring.git

# Push et (authentication gerekebilir)
git push -u origin main

# Başarılı olduysa, ilk release tag'i oluştur (GitHub Actions tetikler)
git tag v1.0.0
git push origin v1.0.0
```

**NOT**: Eğer authentication hatası alırsanız:
- Git Credential Manager otomatik açılacak
- Browser'da GitHub'a login olun
- Authorize edin
- Tekrar `git push -u origin main` çalıştırın

### 5. GitHub Actions için Secrets Ekle (Docker Hub için)

Repo oluştuktan sonra:

1. Repo sayfasında **Settings** → **Secrets and variables** → **Actions**
2. **New repository secret** butonuna tıkla
3. İki secret ekle:
   - Name: `DOCKERHUB_USERNAME`, Value: Docker Hub kullanıcı adınız
   - Name: `DOCKERHUB_TOKEN`, Value: [Docker Hub'dan token oluşturun](https://hub.docker.com/settings/security)

### 6. Otomatik Build Tetikleme

Tag push'u GitHub Actions'ı tetikleyecek ve Docker Hub'a multi-arch image'lar push edilecek.

---

**Not**: Repo oluşturduktan sonra bana "tamam" yazın, gerisini ben hallederim!
