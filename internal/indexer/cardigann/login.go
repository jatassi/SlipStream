package cardigann

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// LoginHandler manages authentication with indexer sites.
type LoginHandler struct {
	httpClient *http.Client
	jar        *cookiejar.Jar
	logger     zerolog.Logger
	baseURL    string
	userAgent  string
}

// NewLoginHandler creates a new login handler.
func NewLoginHandler(baseURL string, logger zerolog.Logger) (*LoginHandler, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	return &LoginHandler{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		jar:       jar,
		logger:    logger.With().Str("component", "login").Logger(),
		baseURL:   strings.TrimSuffix(baseURL, "/"),
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}, nil
}

// Authenticate performs authentication based on the login block configuration.
func (h *LoginHandler) Authenticate(ctx context.Context, login *LoginBlock, settings map[string]string) error {
	if login == nil {
		return nil // No authentication required
	}

	switch strings.ToLower(login.Method) {
	case "post":
		return h.loginPOST(ctx, login, settings)
	case "form":
		return h.loginForm(ctx, login, settings)
	case "cookie":
		return h.loginCookie(ctx, login, settings)
	case "oneurl":
		return h.loginOneURL(ctx, login, settings)
	case "":
		// No login method specified, assume no auth needed
		return nil
	default:
		return fmt.Errorf("unsupported login method: %s", login.Method)
	}
}

// loginPOST performs POST-based authentication.
func (h *LoginHandler) loginPOST(ctx context.Context, login *LoginBlock, settings map[string]string) error {
	loginURL := h.baseURL + login.Path

	// Build form data
	formData := url.Values{}
	engine := NewTemplateEngine()
	tmplCtx := NewTemplateContext()
	tmplCtx.Config = settings

	for key, tmpl := range login.Inputs {
		val, err := engine.Evaluate(tmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate input %s: %w", key, err)
		}
		formData.Set(key, val)
	}

	h.logger.Debug().Str("url", loginURL).Msg("Performing POST login")

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", h.userAgent)

	// Add custom headers
	for key, val := range login.Headers {
		evaluated, _ := engine.Evaluate(string(val), tmplCtx)
		req.Header.Set(key, evaluated)
	}

	// Execute request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Check for error selectors
	if len(login.Error) > 0 {
		htmlSel, err := NewHTMLSelector(body)
		if err == nil {
			for _, errSel := range login.Error {
				if htmlSel.Exists(errSel.Selector) {
					errMsg := "Login failed"
					if errSel.Message != nil {
						if errSel.Message.Text != "" {
							errMsg = errSel.Message.Text
						} else if errSel.Message.Selector != "" {
							errMsg = htmlSel.FindText(errSel.Message.Selector)
						}
					}
					return fmt.Errorf("login error: %s", errMsg)
				}
			}
		}
	}

	h.logger.Debug().Int("status", resp.StatusCode).Msg("POST login completed")

	return nil
}

// loginForm performs form-based authentication with selector inputs.
func (h *LoginHandler) loginForm(ctx context.Context, login *LoginBlock, settings map[string]string) error {
	// First, fetch the login page to get form fields
	loginPageURL := h.baseURL + login.Path

	h.logger.Debug().Str("url", loginPageURL).Msg("Fetching login page")

	req, err := http.NewRequestWithContext(ctx, "GET", loginPageURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create login page request: %w", err)
	}

	req.Header.Set("User-Agent", h.userAgent)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch login page: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse the page
	htmlSel, err := NewHTMLSelector(body)
	if err != nil {
		return fmt.Errorf("failed to parse login page: %w", err)
	}

	// Find the form
	formSelector := login.Form
	if formSelector == "" {
		formSelector = "form"
	}

	formSel := htmlSel.Select(formSelector)
	if formSel.Length() == 0 {
		return fmt.Errorf("login form not found: %s", formSelector)
	}

	// Get form action URL
	formAction, _ := formSel.Attr("action")
	if formAction == "" {
		formAction = login.Path
	}
	if !strings.HasPrefix(formAction, "http") {
		if strings.HasPrefix(formAction, "/") {
			formAction = h.baseURL + formAction
		} else {
			formAction = h.baseURL + "/" + formAction
		}
	}

	// Build form data
	formData := url.Values{}
	engine := NewTemplateEngine()
	tmplCtx := NewTemplateContext()
	tmplCtx.Config = settings

	// Add selector inputs (extract values from the page)
	for name, selDef := range login.SelectorInputs {
		val := ExtractText(htmlSel.Select(selDef.Selector), selDef.Attribute)
		if len(selDef.Filters) > 0 {
			val, _ = ApplyFilters(val, selDef.Filters)
		}
		formData.Set(name, val)
	}

	// Add static inputs
	for key, tmpl := range login.Inputs {
		val, err := engine.Evaluate(tmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate input %s: %w", key, err)
		}
		formData.Set(key, val)
	}

	h.logger.Debug().Str("url", formAction).Msg("Submitting login form")

	// Submit the form
	req, err = http.NewRequestWithContext(ctx, "POST", formAction, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create form submit request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", h.userAgent)
	req.Header.Set("Referer", loginPageURL)

	resp, err = h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("form submit failed: %w", err)
	}
	defer resp.Body.Close()

	h.logger.Debug().Int("status", resp.StatusCode).Msg("Form login completed")

	return nil
}

// loginCookie uses user-provided cookies for authentication.
func (h *LoginHandler) loginCookie(ctx context.Context, login *LoginBlock, settings map[string]string) error {
	engine := NewTemplateEngine()
	tmplCtx := NewTemplateContext()
	tmplCtx.Config = settings

	// Get cookie string from settings
	cookieStr := ""
	if tmpl, ok := login.Inputs["cookie"]; ok {
		var err error
		cookieStr, err = engine.Evaluate(tmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate cookie input: %w", err)
		}
	} else if c, ok := settings["cookie"]; ok {
		cookieStr = c
	}

	if cookieStr == "" {
		return fmt.Errorf("no cookie provided for cookie authentication")
	}

	// Parse the base URL
	baseURL, err := url.Parse(h.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// Parse and set cookies
	cookies := parseCookieString(cookieStr)
	h.jar.SetCookies(baseURL, cookies)

	h.logger.Debug().Int("cookies", len(cookies)).Msg("Set cookies for authentication")

	return nil
}

// loginOneURL uses a single URL with embedded authentication (like RSS feeds with API keys).
func (h *LoginHandler) loginOneURL(ctx context.Context, login *LoginBlock, settings map[string]string) error {
	// For oneurl method, the URL itself contains authentication
	// We just need to validate that the required settings are present
	engine := NewTemplateEngine()
	tmplCtx := NewTemplateContext()
	tmplCtx.Config = settings

	for key, tmpl := range login.Inputs {
		val, err := engine.Evaluate(tmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("failed to evaluate input %s: %w", key, err)
		}
		if val == "" {
			return fmt.Errorf("required setting %s is empty", key)
		}
	}

	h.logger.Debug().Msg("OneURL authentication validated")

	return nil
}

// Test verifies that authentication was successful.
func (h *LoginHandler) Test(ctx context.Context, login *LoginBlock) error {
	if login == nil || login.Test.Path == "" {
		return nil // No test defined
	}

	testURL := h.baseURL + login.Test.Path

	h.logger.Debug().Str("url", testURL).Msg("Testing authentication")

	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("User-Agent", h.userAgent)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("test request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("test request returned status %d", resp.StatusCode)
	}

	// Check for success selector if specified
	if login.Test.Selector != "" {
		body, _ := io.ReadAll(resp.Body)
		htmlSel, err := NewHTMLSelector(body)
		if err != nil {
			return fmt.Errorf("failed to parse test response: %w", err)
		}

		if !htmlSel.Exists(login.Test.Selector) {
			return fmt.Errorf("login test failed: selector %s not found", login.Test.Selector)
		}
	}

	h.logger.Debug().Msg("Authentication test passed")

	return nil
}

// GetHTTPClient returns the HTTP client with cookies set.
func (h *LoginHandler) GetHTTPClient() *http.Client {
	return h.httpClient
}

// GetCookies returns the current cookies for the base URL.
func (h *LoginHandler) GetCookies() []*http.Cookie {
	baseURL, err := url.Parse(h.baseURL)
	if err != nil {
		return nil
	}
	return h.jar.Cookies(baseURL)
}

// SetCookies sets cookies for the base URL.
func (h *LoginHandler) SetCookies(cookies []*http.Cookie) {
	baseURL, err := url.Parse(h.baseURL)
	if err != nil {
		return
	}
	h.jar.SetCookies(baseURL, cookies)
}

// parseCookieString parses a cookie string like "name1=value1; name2=value2".
func parseCookieString(cookieStr string) []*http.Cookie {
	var cookies []*http.Cookie

	pairs := strings.Split(cookieStr, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}

		cookies = append(cookies, &http.Cookie{
			Name:  strings.TrimSpace(parts[0]),
			Value: strings.TrimSpace(parts[1]),
		})
	}

	return cookies
}
