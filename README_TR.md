# niye

[For English README](README.md)

## Genel Bakış

Go ile yazılmış küçük bir CLI aracı; terminal hata çıktısını okuyup iki satırlık açıklama yazdırır:

- **Neden**: tek cümle
- **Çözüm**: kopyalayıp çalıştırılabilir komut(lar)

Komut adı: `niye`. Go 1.22+ gerektirir.

---

## Örnek

```bash
$ ./niye "git: 'pushh' is not a git command."
● Neden: 'pushh' geçerli bir git komutu değil, muhtemelen 'push' yazılmak istenmiş.
● Çözüm:
    git push
```

Çıktı formatı (birebir):

```text
● Neden: <tek cümle>
● Çözüm:
    <kopyalayıp çalıştırabileceğiniz komut(lar)>
```

---

## Kurulum

```bash
git clone https://github.com/kuscadev/niye.git
cd niye
make build          # `niye` binary'sini derler
./niye --help       # test

make install        # ~/.local/bin'e kurar
```

Klonlamadan kurulum:

```bash
go install github.com/kuscadev/niye/cmd/niye@latest
```

---

## Hızlı Kullanım

```bash
# Derleme hatalarını borula
go build ./... 2>&1 | niye

# Hata mesajını argüman olarak ver
niye "git: 'pushh' is not a git command."

# İngilizce çıktı
niye --lang en "error: command not found"

# Etkileşimli yapılandırma
niye init
```

```bash
niye [seçenekler] [hata_mesajı]

Seçenekler:
  --lang tr|en              Çıktı dili (varsayılan: tr)
  --provider openai|gemini  Sağlayıcı (varsayılan: openai)
  --base-url URL            API temel URL
  --model MODEL             Model adı
  --api-key KEY             API anahtarı
  --timeout SÜRE            İstek zaman aşımı (varsayılan: 30s)
  --config DOSYA            Yapılandırma dosyası yolu
  -h, --help                Bu yardımı gösterir
  --version                 Sürümü gösterir
```

Argüman yok ve stdin bir terminalse: beklemeden kullanım mesajı yazıp 1 ile çıkar.

Modele giden girdi 32 KB ile sınırlıdır: daha uzun girdilerde baş ve son tutulur, orta atılır ve stderr'e tek satır bilgi yazılır.

### Çıkış Kodları

| Kod | Anlam |
|-----|-------|
| 0 | Başarılı |
| 1 | Kullanım/yapılandırma hatası (eksik anahtar, bozuk config, boş giriş, geçersiz bayrak) |
| 2 | Sağlayıcı hatası (401/403, 404, 429, ağ, zaman aşımı) |

---

## Yapılandırma

Yapılandırma dosyası: `~/.config/niye/config.toml` (mod `0600`). `XDG_CONFIG_HOME`'a saygı duyar.

```toml
provider = "openai"          # openai | gemini
base_url = "https://api.openai.com/v1"
model    = "gpt-4o-mini"
api_key  = "sk-..."          # opsiyonel; yerine NIYE_API_KEY ortam değişkeni kullanılabilir
lang     = "tr"              # tr | en
timeout  = "30s"
```

### Öncelik Sırası

| Öncelik | Kaynak |
|---------|--------|
| 1 (en yüksek) | CLI bayrakları (`--provider`, `--base-url`, `--model`, `--api-key`, `--lang`, `--timeout`, `--config`) |
| 2 | Ortam değişkenleri (`NIYE_*`, sonra `OPENAI_API_KEY` / `GEMINI_API_KEY`) |
| 3 | `~/.config/niye/config.toml` |
| 4 (en düşük) | Yerleşik varsayılanlar |

Ortam değişkenleri:

- `NIYE_PROVIDER`, `NIYE_BASE_URL`, `NIYE_MODEL`, `NIYE_API_KEY`, `NIYE_LANG`, `NIYE_TIMEOUT`
- `OPENAI_API_KEY` (openai sağlayıcısı için)
- `GEMINI_API_KEY` (gemini sağlayıcısı için)

### Etkileşimli Kurulum

```bash
niye init
```

Sorular: sağlayıcı, base URL, model, **API anahtarı (terminalde gizli giriş)**, dil, zaman aşımı.
Mevcut config varsa üzerine yazmadan önce sorar.
Anahtarı diske yazmamak için seçenek sunar (`NIYE_API_KEY` kullan).

---

## Sağlayıcılar

| Sağlayıcı | Base URL Örneği | Kimlik Doğrulama |
|-----------|-----------------|-------------------|
| **OpenAI** | `https://api.openai.com/v1` | `Bearer` başlığı |
| **OpenRouter** | `https://openrouter.ai/api/v1` | `Bearer` başlığı |
| **Ollama** | `http://localhost:11434/v1` | **Kimlik doğrulama yok** (boş anahtar) |
| **Gemini (OpenAI-uyumlu)** | `https://generativelanguage.googleapis.com/v1beta/openai/` | `Bearer` başlığı |
| **Gemini (native)** | `https://generativelanguage.googleapis.com/v1beta/models/<model-adı>` | `x-goog-api-key` başlığı |

Güncel model adı için sağlayıcı belgesine bakın.

Yanıt işleme: önce JSON `{"reason","fix"}` bekler; olmazsa etiketli bölümlere (`Neden:`/`Çözüm:`), kod bloğuna, sonra ilk-satır-neden geri dönüşüne bakar.

---

## Gizlilik

LLM'e göndermeden önce hata metni maskelenir. Yalnızca maskelenmiş hata metni ve kısa sistem istemi gönderilir.

| Desen | Değişim |
|-------|---------|
| Ev dizini (`/home/kullanıcı/...`) | `~/...` |
| Kullanıcı adı (kelime sınırı, ≥3 karakter) | `<user>` |
| `key=`, `token=`, `password=`, `secret=`, `passwd=` değerleri (≥4 karakter) | `***` |
| `sk-...` anahtarları, `ghp_`/`gho_`/`github_pat_` token'ları, `AKIA...` AWS kimlikleri, JWT | `***` |
| `https://kullanıcı...@host` kullanıcı bilgisi | `https://***@host` |
| `Bearer <token>` | `Bearer ***` |
| JSON `"key": "value"` | `"key": "***"` |

Maskeleme en iyi çabadır: her sırrı yakalamayı garanti etmez. Hassas çıktıları paylaşmadan önce gözden geçirin.

Not: ücretsiz Gemini kotasında içeriğiniz Google tarafından ürün geliştirmede kullanılabilir. Hassas içerik için yerel bir sağlayıcı (örn. Ollama) ya da ücretli plan kullanın.

Modele giden girdi 32 KB ile sınırlıdır (baş + son tutulur, orta atılır).

---

## Güvenlik

- Önerilen düzeltme **asla çalıştırılmaz.** Sadece yazdırılır.
- Tehlikeli komutlarda komutun **üzerinde** uyarı satırı yazdırılır:
  - Hedefi `/`, `~`, `$HOME`, `*` ya da `.` olan `rm -rf`
  - `sudo`'lu ya da `sudo`'suz `chmod -R 777`
  - `git push --force` / `-f`, `git reset --hard`
  - `/dev/sd*` / `/dev/nvme*` aygıtlarına yönlendirme
  - `mkfs ...`
  - `dd of=...`
  - `curl ... | bash` / `wget ... | sh` desenleri
- Çıktı renksiz, pipe-safe'dir.

---

## Mimari

```text
stdin / argv (hata metni)
        │
        ▼
┌───────────────┐
│ cmd/niye      │  bayrak işleme, run, runInit
└───┬───┬───┬───┘
    │   │   │
    ▼   ▼   ▼
 config mask i18n        secret (init: gizli API anahtarı girişi)
    │     │
    │     ▼  maskelenmiş hata metni (en fazla 32 KB) + sistem promptu
    │   ┌──────────┐
    │   │ provider │  OpenAI-uyumlu / native Gemini
    │   └──────────┘
    │         │
    ▼         ▼
  config   yanıt {reason, fix}
  dosyası         │
                  ▼
            ┌──────────┐
            │ render   │  iki satırlık çıktı + tehlike uyarıları
            └──────────┘
```

| Yol | Amaç |
|-----|------|
| `cmd/niye/main.go` | Giriş noktası, bayrak işleme, ana mantık, alt komut yönetimi |
| `internal/config/config.go` | Yapılandırma yükleme (TOML, ortam değişkenleri, CLI bayrakları), öncelik, 0600 ile kaydetme |
| `internal/mask/mask.go` | Gizlilik maskeleme (ev dizini, kullanıcı adı, sırlar, Bearer token) |
| `internal/provider/provider.go` | Sağlayıcı arayüzü, ortak HTTP, JSON/düz-metin çıkarımı |
| `internal/provider/provider_openai.go` | OpenAI-uyumlu adaptör (OpenAI, OpenRouter, Ollama, Gemini uyumlu) |
| `internal/provider/provider_gemini.go` | Native Gemini API adaptörü |
| `internal/render/render.go` | İki satırlık çıktı formatı, tehlikeli komut uyarıları |
| `internal/i18n/i18n.go` | Türkçe/İngilizce mesaj tabloları |
| `internal/secret/secret.go` | Güvenli API anahtarı girişi (platform bağımsız arayüz) |
| `internal/secret/secret_linux.go` | Linux termios echo kapatma |
| `internal/secret/secret_darwin.go` | macOS termios echo kapatma |
| `internal/secret/secret_other.go` | Desteklenmeyen platformlar için geri dönüş |
| `LICENSE` | MIT lisansı |
| `go.mod` | Modül tanımı (`github.com/kuscadev/niye`, Go 1.22) |

---

## Teknik Kararlar

| Karar | Uygulama |
|-------|----------|
| TOML ayrıştırıcı | El yapımı minimal, sıfır bağımlılık (`internal/config`) |
| `niye init` | Sadece config yazar, canlı API doğrulaması yapmaz |
| API anahtarı girişi | `termios` ioctl ile gizli (Linux/macOS), geri dönüş görünür |
| Gemini kimlik doğrulama | `x-goog-api-key` başlığı; anahtar URL'ye konmaz, hata mesajlarına sızamaz |
| Girdi sınırı | 32 KB baş+son kırpma, tek satır stderr notu |
| Yanıt ayrıştırma | Önce katı JSON, sonra dengeli-süslü-parantez taraması, etiketli metin, kod bloğu, ilk-satır geri dönüşü |
| Yerleşim | `cmd/niye` + `internal/{config,mask,provider,render,i18n,secret}` |

---

## Geliştirme ve Testler

```bash
make check     # fmt + vet + test
make test      # tüm testleri çalıştırır
make vet       # go vet
make fmt       # gofmt
```

### Test Kapsamı

| Test dosyası | Kapsam |
|--------------|--------|
| `internal/mask/mask_test.go`, `extra_test.go` | Her maskeleme kuralı için tablo-odaklı durumlar + negatifler |
| `internal/provider/provider_test.go`, `key_leak_test.go` | httptest sunucuları: başarı, 401, 404, 429, bozuk JSON, zaman aşımı, bağlantı reddi, kimlik doğrulamasız Ollama, başlık doğrulaması, hatalarda anahtar yokluğu |
| `internal/config/config_test.go` | Öncelik matrisi, TOML gidiş-dönüş, 0600 modu, XDG_CONFIG_HOME |
| `internal/render/render_test.go`, `dangerous_test.go` | Birebir golden stringler, tr/en etiketleri, tehlikeli komut uyarıları |
| `cmd/niye/main_test.go`, `input_limit_test.go` | `run()` üzerinden E2E: yardım, sürüm, stdin, argüman, eksik anahtar, sağlayıcı hataları, init alt komutu, 32 KB kırpma, terminal kullanımı |
| `internal/secret/secret_test.go` | Pipeli giriş, prompt yazımı, terminal tespiti |

---

## Lisans

MIT License — `LICENSE` dosyasına bakın.
