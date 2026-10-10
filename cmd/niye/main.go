package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kuscadev/niye/internal/config"
	"github.com/kuscadev/niye/internal/i18n"
	"github.com/kuscadev/niye/internal/mask"
	"github.com/kuscadev/niye/internal/provider"
	"github.com/kuscadev/niye/internal/render"
	"github.com/kuscadev/niye/internal/secret"
)

// maxInputBytes caps the error text sent to the model.
const maxInputBytes = 32 * 1024

// truncateInput caps s at limit bytes, keeping the head and the tail and
// replacing the dropped middle with markerFmt (%d = omitted bytes).
// Cuts are adjusted to UTF-8 rune boundaries.
func truncateInput(s string, limit int, markerFmt string) (string, int, bool) {
	if len(s) <= limit {
		return s, 0, false
	}
	omitted := len(s) - limit
	marker := fmt.Sprintf(markerFmt, omitted)
	keep := limit - len(marker)
	if keep < 2 {
		return cutHead(s, limit), len(s) - limit, true
	}
	headLen := keep / 2
	tailLen := keep - headLen
	return cutHead(s, headLen) + marker + cutTail(s, tailLen), omitted, true
}

// cutHead returns the first n bytes of s, moved back to a rune boundary.
func cutHead(s string, n int) string {
	if n >= len(s) {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}

// cutTail returns the last n bytes of s, moved forward to a rune boundary.
func cutTail(s string, n int) string {
	if n >= len(s) {
		return s
	}
	start := len(s) - n
	for start < len(s) && !utf8.RuneStart(s[start]) {
		start++
	}
	return s[start:]
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// Check for subcommand first (before flag parsing)
	if len(args) > 1 && args[1] == "init" {
		return runInitSubcommand(args[2:], stdin, stdout, stderr)
	}

	// Parse flags for main command
	fs := flag.NewFlagSet("niye", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		flagLang       = fs.String("lang", "", "output language (tr|en)")
		flagProvider   = fs.String("provider", "", "provider (openai|gemini)")
		flagBaseURL    = fs.String("base-url", "", "API base URL")
		flagModel      = fs.String("model", "", "model name")
		flagAPIKey     = fs.String("api-key", "", "API key")
		flagTimeout    = fs.Duration("timeout", 0, "request timeout")
		flagConfigFile = fs.String("config", "", "config file path")
		flagHelp       = fs.Bool("help", false, "show help")
		flagVersion    = fs.Bool("version", false, "show version")
	)

	// Custom usage
	fs.Usage = func() {
		msg := i18n.GetMessages(*flagLang)
		fmt.Fprint(stderr, msg.Usage)
	}

	if err := fs.Parse(args[1:]); err != nil {
		return 1
	}

	if *flagHelp {
		fs.Usage()
		return 0
	}

	if *flagVersion {
		fmt.Fprintln(stdout, "niye 0.1.0")
		return 0
	}

	// Set language for messages
	lang := *flagLang
	if lang == "" {
		lang = "tr"
	}
	i18n.SetInitMessages(i18n.GetMessages(lang))

	// Get error text from argument or stdin
	var errorText string
	if fs.NArg() > 0 {
		errorText = strings.Join(fs.Args(), " ")
	} else {
		// No args and stdin is a terminal: don't block waiting for input.
		if secret.IsTerminal(stdin) {
			fs.Usage()
			return 1
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "stdin okuma hatası: %v\n", err)
			return 1
		}
		errorText = strings.TrimSpace(string(data))
	}

	if errorText == "" {
		msg := i18n.GetMessages(lang)
		fmt.Fprintln(stderr, msg.ErrEmptyInput)
		return 1
	}

	// Load configuration
	cfg, err := config.LoadConfig(*flagProvider, *flagBaseURL, *flagModel, *flagAPIKey, lang, *flagTimeout, *flagConfigFile)
	if err != nil {
		fmt.Fprintf(stderr, "yapılandırma hatası: %v\n", err)
		return 1
	}

	// Validate API key
	if err := config.ValidateConfig(cfg); err != nil {
		msg := i18n.GetMessages(lang)
		fmt.Fprintln(stderr, msg.ErrMissingAPIKey)
		return 1
	}

	// Mask sensitive data
	maskedError := mask.Mask(errorText)

	// Cap the model input at 32 KB, keeping head and tail.
	msg := i18n.GetMessages(lang)
	if truncated, omitted, ok := truncateInput(maskedError, maxInputBytes, msg.MarkInputTruncated); ok {
		maskedError = truncated
		fmt.Fprintf(stderr, msg.NoteInputTruncated+"\n", omitted)
	}

	// Create provider
	var p provider.Provider
	switch cfg.Provider {
	case "openai":
		p = provider.NewOpenAIProvider(cfg.BaseURL, cfg.APIKey, cfg.Model, cfg.Timeout)
	case "gemini":
		p = provider.NewGeminiProvider(cfg.BaseURL, cfg.APIKey, cfg.Model, cfg.Timeout)
	default:
		msg := i18n.GetMessages(lang)
		fmt.Fprintf(stderr, msg.ErrInvalidProvider, cfg.Provider)
		return 1
	}

	// Ask provider
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	ans, err := p.Ask(ctx, provider.Input{ErrorText: maskedError})
	if err != nil {
		msg := i18n.GetMessages(lang)
		errStr := err.Error()
		switch {
		case strings.Contains(errStr, "401") || strings.Contains(errStr, "403") || strings.Contains(errStr, "yetkisiz"):
			fmt.Fprintln(stderr, msg.ErrUnauthorized)
		case strings.Contains(errStr, "404") || strings.Contains(errStr, "bulunamadı"):
			fmt.Fprintln(stderr, msg.ErrModelNotFound)
		case strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit"):
			fmt.Fprintln(stderr, msg.ErrRateLimited)
		case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "zaman aşımı") || strings.Contains(errStr, "context deadline"):
			fmt.Fprintf(stderr, msg.ErrTimeout, cfg.Timeout)
		default:
			fmt.Fprintf(stderr, msg.ErrProvider, err)
		}
		return 2
	}

	// Render output
	output := render.RenderOutput(ans, lang)
	fmt.Fprintln(stdout, output)
	return 0
}

// runInitSubcommand handles the "niye init" subcommand with its own flag parsing.
func runInitSubcommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	// Parse flags for init subcommand
	fs := flag.NewFlagSet("niye init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		flagLang       = fs.String("lang", "", "output language (tr|en)")
		flagConfigFile = fs.String("config", "", "config file path")
		flagHelp       = fs.Bool("help", false, "show help")
	)

	// Custom usage for init
	fs.Usage = func() {
		msg := i18n.GetMessages(*flagLang)
		fmt.Fprint(stderr, msg.Usage)
	}

	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *flagHelp {
		fs.Usage()
		return 0
	}

	// Set language for messages
	lang := *flagLang
	if lang == "" {
		lang = "tr"
	}
	i18n.SetInitMessages(i18n.GetMessages(lang))

	// Now run the interactive init with parsed flags as defaults
	return runInit(fs.Args(), stdin, stdout, stderr, lang, *flagConfigFile)
}

func runInit(args []string, stdin io.Reader, stdout, stderr io.Writer, lang string, configFile string) int {
	msg := i18n.GetMessages(lang)
	reader := bufio.NewReader(stdin)

	// Check if config exists
	configPath := configFile
	if configPath == "" {
		var err error
		configPath, err = config.ConfigFilePath()
		if err != nil {
			fmt.Fprintf(stderr, "config yolu hatası: %v\n", err)
			return 1
		}
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Fprintf(stderr, msg.InitConfigExistsMsg+"\n", configPath)
		if secret.IsTerminal(stdin) {
			fmt.Fprint(stderr, msg.InitOverwritePrompt)
			line, _ := reader.ReadString('\n')
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "e") &&
				!strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "y") {
				fmt.Fprintln(stderr, "iptal edildi")
				return 0
			}
		}
	}

	cfg := config.DefaultConfig()

	// Provider
	fmt.Fprint(stderr, msg.InitProviderPrompt)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		cfg.Provider = line
	}

	// Base URL
	fmt.Fprint(stderr, msg.InitBaseURLPrompt)
	line, _ = reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		cfg.BaseURL = line
	}

	// Model
	fmt.Fprint(stderr, msg.InitModelPrompt)
	line, _ = reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		cfg.Model = line
	}

	// API Key (hidden)
	apiKey, err := secret.PromptSecret(msg.InitAPIKeyPrompt, reader)
	if err != nil {
		fmt.Fprintf(stderr, "anahtar okunamadı: %v\n", err)
		return 1
	}
	cfg.APIKey = apiKey

	// Language
	fmt.Fprint(stderr, msg.InitLangPrompt)
	line, _ = reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		cfg.Lang = line
	}

	// Timeout
	fmt.Fprint(stderr, msg.InitTimeoutPrompt)
	line, _ = reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line != "" {
		if d, err := time.ParseDuration(line); err == nil {
			cfg.Timeout = d
		}
	}

	// Ask if store key
	storeKey := true
	if secret.IsTerminal(stdin) {
		fmt.Fprint(stderr, msg.InitStoreKeyPrompt)
		line, _ = reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "h" || line == "n" || line == "no" {
			storeKey = false
		}
	}

	if !storeKey {
		cfg.APIKey = ""
		fmt.Fprintln(stderr, msg.InitKeyNotStoredMsg)
	}

	// Save config
	if err := config.SaveConfigToPath(cfg, configFile); err != nil {
		fmt.Fprintf(stderr, msg.ErrConfigWrite, err)
		return 1
	}

	fmt.Fprintf(stderr, msg.InitConfigSavedMsg+"\n", configPath)
	return 0
}

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}
