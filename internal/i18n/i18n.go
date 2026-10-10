package i18n

// Messages holds localized strings.
type Messages struct {
	// Usage
	Usage string

	// Errors
	ErrEmptyInput      string
	ErrMissingAPIKey   string
	ErrConfigParse     string
	ErrConfigWrite     string
	ErrProvider        string
	ErrTimeout         string
	ErrNetwork         string
	ErrUnauthorized    string
	ErrModelNotFound   string
	ErrRateLimited     string
	ErrInvalidLang     string
	ErrInvalidProvider string
	NoteInputTruncated string
	MarkInputTruncated string

	// Init
	InitProviderPrompt   string
	InitBaseURLPrompt    string
	InitModelPrompt      string
	InitAPIKeyPrompt     string
	InitLangPrompt       string
	InitTimeoutPrompt    string
	InitStoreKeyPrompt   string
	InitKeyNotStoredMsg  string
	InitConfigSavedMsg   string
	InitConfigExistsMsg  string
	InitOverwritePrompt  string
	InitEchoDisabledNote string
	InitEchoUnavailable  string

	// Labels
	LabelReason   string
	LabelFix      string
	WarningPrefix string
}

// TR returns Turkish messages.
func TR() Messages {
	return Messages{
		Usage: `Kullanım: niye [seçenekler] [hata_mesajı]
  niye init                    # Etkileşimli yapılandırma

Seçenekler:
  --lang tr|en       Çıktı dili (varsayılan: tr)
  --provider openai|gemini  Sağlayıcı (varsayılan: openai)
  --base-url URL     API temel URL
  --model MODEL      Model adı
  --api-key KEY      API anahtarı
  --timeout SÜRE     İstek zaman aşımı (varsayılan: 30s)
  --config DOSYA     Yapılandırma dosyası yolu
  -h, --help         Bu yardımı göster

Örnekler:
  go build ./... 2>&1 | niye
  niye "git: 'pushh' is not a git command."
  niye --lang en "error message"`,
		ErrEmptyInput:      "boş hata girişi: stdin'den veya argüman olarak hata mesajı sağlayın",
		ErrMissingAPIKey:   "API anahtarı bulunamadı. NIYE_API_KEY ayarla veya \"niye init\" çalıştır",
		ErrConfigParse:     "yapılandırma dosyası okunamadı: %v",
		ErrConfigWrite:     "yapılandırma yazılamadı: %v",
		ErrProvider:        "sağlayıcı hatası: %v",
		ErrTimeout:         "istek zaman aşımına uğradı (%v)",
		ErrNetwork:         "ağ hatası: %v",
		ErrUnauthorized:    "API anahtarı geçersiz veya yetkisiz (401/403)",
		ErrModelNotFound:   "model bulunamadı (404)",
		ErrRateLimited:     "rate limit aşıldı (429), lütfen bekleyin",
		ErrInvalidLang:     "geçersiz dil: %s (tr|en)",
		ErrInvalidProvider: "geçersiz provider: %s (openai|gemini)",
		NoteInputTruncated: "bilgi: girdi 32 KB sınırını aştı, orta kısmı atıldı (%d bayt atlandı)",
		MarkInputTruncated: "[...girdinin orta kısmı atıldı (%d bayt)...]",

		InitProviderPrompt:   "Sağlayıcı [openai|gemini] (varsayılan: openai): ",
		InitBaseURLPrompt:    "API Base URL (varsayılan: https://api.openai.com/v1): ",
		InitModelPrompt:      "Model (varsayılan: gpt-4o-mini): ",
		InitAPIKeyPrompt:     "API Anahtarı (gizli): ",
		InitLangPrompt:       "Dil [tr|en] (varsayılan: tr): ",
		InitTimeoutPrompt:    "Zaman aşımı (varsayılan: 30s): ",
		InitStoreKeyPrompt:   "Anahtarı config dosyasına yazmak istiyor musunuz? [E/h]: ",
		InitKeyNotStoredMsg:  "Anahtar diske yazılmadı. NIYE_API_KEY ortam değişkenini kullanın.",
		InitConfigSavedMsg:   "Yapılandırma kaydedildi: %s (mod 0600)",
		InitConfigExistsMsg:  "Yapılandırma dosyası zaten var: %s",
		InitOverwritePrompt:  "Üzerine yazmak istiyor musunuz? [e/H]: ",
		InitEchoDisabledNote: "(echo kapalı - anahtar ekranda görünmeyecek)",
		InitEchoUnavailable:  "(echo gizleme desteklenmiyor, anahtar görünür olacak)",

		LabelReason:   "Neden",
		LabelFix:      "Çözüm",
		WarningPrefix: "⚠ Uyarı",
	}
}

// EN returns English messages.
func EN() Messages {
	return Messages{
		Usage: `Usage: niye [options] [error_message]
  niye init                    # Interactive configuration

Options:
  --lang tr|en       Output language (default: tr)
  --provider openai|gemini  Provider (default: openai)
  --base-url URL     API base URL
  --model MODEL      Model name
  --api-key KEY      API key
  --timeout DURATION Request timeout (default: 30s)
  --config FILE      Config file path
  -h, --help         Show this help

Examples:
  go build ./... 2>&1 | niye
  niye "git: 'pushh' is not a git command."
  niye --lang en "error message"`,
		ErrEmptyInput:      "empty error input: provide error message via stdin or argument",
		ErrMissingAPIKey:   "API key not found. Set NIYE_API_KEY or run \"niye init\"",
		ErrConfigParse:     "failed to parse config: %v",
		ErrConfigWrite:     "failed to write config: %v",
		ErrProvider:        "provider error: %v",
		ErrTimeout:         "request timed out (%v)",
		ErrNetwork:         "network error: %v",
		ErrUnauthorized:    "API key invalid or unauthorized (401/403)",
		ErrModelNotFound:   "model not found (404)",
		ErrRateLimited:     "rate limited (429), please wait",
		ErrInvalidLang:     "invalid language: %s (tr|en)",
		ErrInvalidProvider: "invalid provider: %s (openai|gemini)",
		NoteInputTruncated: "note: input exceeded the 32 KB limit, middle part truncated (%d bytes omitted)",
		MarkInputTruncated: "[...middle of input truncated (%d bytes)...]",

		InitProviderPrompt:   "Provider [openai|gemini] (default: openai): ",
		InitBaseURLPrompt:    "API Base URL (default: https://api.openai.com/v1): ",
		InitModelPrompt:      "Model (default: gpt-4o-mini): ",
		InitAPIKeyPrompt:     "API Key (hidden): ",
		InitLangPrompt:       "Language [tr|en] (default: tr): ",
		InitTimeoutPrompt:    "Timeout (default: 30s): ",
		InitStoreKeyPrompt:   "Store key in config file? [Y/n]: ",
		InitKeyNotStoredMsg:  "Key not written to disk. Use NIYE_API_KEY environment variable.",
		InitConfigSavedMsg:   "Configuration saved: %s (mode 0600)",
		InitConfigExistsMsg:  "Config file already exists: %s",
		InitOverwritePrompt:  "Overwrite? [y/N]: ",
		InitEchoDisabledNote: "(echo disabled - key won't be visible)",
		InitEchoUnavailable:  "(echo hiding not supported, key will be visible)",

		LabelReason:   "Cause",
		LabelFix:      "Fix",
		WarningPrefix: "⚠ Warning",
	}
}

// Init messages for secret input.
var (
	EchoDisabledNote = "(echo kapalı - anahtar ekranda görünmeyecek)"
	EchoUnavailable  = "(echo gizleme desteklenmiyor, anahtar görünür olacak)"
)

// SetInitMessages overrides the secret-input notes for the active language.
func SetInitMessages(m Messages) {
	EchoDisabledNote = m.InitEchoDisabledNote
	EchoUnavailable = m.InitEchoUnavailable
}

// GetMessages returns messages for the given language.
func GetMessages(lang string) Messages {
	if lang == "en" {
		return EN()
	}
	return TR()
}
