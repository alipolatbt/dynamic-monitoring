# GitHub'a Nasıl Push Edilir

## Adım 1: GitHub'da Yeni Repo Oluştur

1. https://github.com/new adresine git
2. Repository name: `dynamic-monitoring` (veya istediğiniz isim)
3. Description: `Doğuş Teknoloji API Monitoring - Production-ready synthetic API monitoring with dynamic token acquisition and email alerts`
4. **Public** veya **Private** seçin
5. **Create repository** butonuna tıklayın

## Adım 2: Local Repo'yu GitHub'a Bağla ve Push Et

GitHub'da repo oluşturduktan sonra aşağıdaki komutları çalıştırın:

```powershell
# GitHub repo URL'inizi buraya yazın (örnek aşağıda)
# Eğer HTTPS kullanıyorsanız:
git remote add origin https://github.com/<username>/dynamic-monitoring.git

# Veya SSH kullanıyorsanız:
# git remote add origin git@github.com:<username>/dynamic-monitoring.git

# Main branch olarak ayarla (modern GitHub convention)
git branch -M main

# Push et
git push -u origin main
```

## Adım 3: GitHub Actions için Secrets Ekle (Docker Hub için)

Otomatik build/deploy için:

1. GitHub repo sayfasında **Settings** > **Secrets and variables** > **Actions**
2. **New repository secret** butonuna tıkla
3. İki secret ekle:
   - Name: `DOCKERHUB_USERNAME`, Value: Docker Hub kullanıcı adınız
   - Name: `DOCKERHUB_TOKEN`, Value: Docker Hub access token ([buradan](https://hub.docker.com/settings/security) oluşturun)

## Adım 4: Tag Oluştur ve Otomatik Build Tetikle

```powershell
# İlk release tag'i oluştur
git tag v1.0.0
git push origin v1.0.0
```

Bu tag push'u GitHub Actions'ı tetikler ve otomatik olarak Docker Hub'a multi-arch image'lar push edilir.

## Alternatif: Manuel Docker Hub Push

Eğer GitHub Actions kullanmak istemiyorsanız:

```powershell
# Docker Hub'a giriş
docker login

# Local image'ları build et
docker-compose build

# Publish script ile push et
.\scripts\publish-images.ps1 -Username <dockerhub-username>
```

## Repo Linki

Repo oluşturduktan sonra README.md dosyasındaki `<your-repo-url>` kısmını güncelleyin:

```powershell
# README.md'de değişiklik yap
# Commit ve push et
git add README.md
git commit -m "Update repo URL in README"
git push
```
