package notifysink

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// profile is one provider as the sink plays it: its routes, the secret
// values its requests carry, and its answers. The answers follow the
// shapes each provider file's header cites.
type profile struct {
	id       providers.ID
	prefix   string
	patterns []string
	secrets  func(r *http.Request, sent providers.Request) []string
	ok       func(w http.ResponseWriter, r *http.Request)
	limited  func(w http.ResponseWriter, wait time.Duration)
	refused  func(w http.ResponseWriter)
	gone     func(w http.ResponseWriter)
}

func pathValues(names ...string) func(*http.Request, providers.Request) []string {
	return func(r *http.Request, _ providers.Request) []string {
		out := make([]string, 0, len(names))
		for _, n := range names {
			out = append(out, r.PathValue(n))
		}
		return out
	}
}

func headerValues(names ...string) func(*http.Request, providers.Request) []string {
	return func(r *http.Request, _ providers.Request) []string {
		out := make([]string, 0, len(names))
		for _, n := range names {
			out = append(out, r.Header.Get(n))
		}
		return out
	}
}

func status(code int, text string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) { writeText(w, code, text) }
}

func plainLimit(w http.ResponseWriter, wait time.Duration) {
	retryAfter(w, wait)
	writeText(w, http.StatusTooManyRequests, "Too Many Requests")
}

var profiles = []profile{
	{
		id:       "discord",
		patterns: []string{"POST /api/webhooks/{id}/{token}"},
		secrets:  pathValues("token"),
		ok: func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("wait") != "true" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"id": "1180000000000000001", "channel_id": "1180000000000000002", "webhook_id": r.PathValue("id")})
		},
		limited: func(w http.ResponseWriter, wait time.Duration) {
			retryAfter(w, wait)
			w.Header().Set("X-RateLimit-Limit", "5")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Scope", "shared")
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"message": "The resource is being rate limited.", "retry_after": wait.Seconds(), "global": false})
		},
		refused: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "Invalid Webhook Token", "code": 50027})
		},
		gone: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusNotFound, map[string]any{"message": "Unknown Webhook", "code": 10015})
		},
	},
	{
		id:       "telegram",
		prefix:   "/telegram",
		patterns: []string{"POST /telegram/{bot}/sendMessage", "POST /telegram/{bot}/sendPhoto"},
		secrets: func(r *http.Request, _ providers.Request) []string {
			return []string{strings.TrimPrefix(r.PathValue("bot"), "bot")}
		},
		ok: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": map[string]any{"message_id": 1}})
		},
		limited: func(w http.ResponseWriter, wait time.Duration) {
			secs := int(wait.Seconds())
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"ok": false, "error_code": 429, "description": "Too Many Requests: retry after " + strconv.Itoa(secs), "parameters": map[string]int{"retry_after": secs}})
		},
		refused: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error_code": 400, "description": "Bad Request: chat not found"})
		},
		gone: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error_code": 404, "description": "Not Found"})
		},
	},
	{
		id:       "slack",
		patterns: []string{"POST /services/{team}/{bot}/{token}"},
		secrets:  pathValues("token"),
		ok:       func(w http.ResponseWriter, _ *http.Request) { writeText(w, http.StatusOK, "ok") },
		limited:  plainLimit,
		refused:  status(http.StatusForbidden, "action_prohibited"),
		gone:     status(http.StatusGone, "channel_is_archived"),
	},
	{
		id:       "pushover",
		prefix:   "/pushover",
		patterns: []string{"POST /pushover/1/messages.json"},
		secrets: func(_ *http.Request, sent providers.Request) []string {
			f := fields(sent)
			return []string{f["token"], f["user"]}
		},
		ok: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Limit-App-Limit", "10000")
			w.Header().Set("X-Limit-App-Remaining", "9999")
			writeJSON(w, http.StatusOK, map[string]any{"status": 1, "request": "647d2300-702c-4b38-8b2f-d56326ae460b"})
		},
		limited: func(w http.ResponseWriter, wait time.Duration) {
			w.Header().Set("X-Limit-App-Limit", "10000")
			w.Header().Set("X-Limit-App-Remaining", "0")
			w.Header().Set("X-Limit-App-Reset", strconv.FormatInt(time.Now().Add(wait).Unix(), 10))
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"status": 0, "errors": []string{"application is over its monthly message limit"}, "request": "647d2300-702c-4b38-8b2f-d56326ae460b"})
		},
		refused: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"user": "invalid", "errors": []string{"user identifier is not a valid user, group, or subscribed user key"}, "status": 0, "request": "647d2300-702c-4b38-8b2f-d56326ae460b"})
		},
		gone: status(http.StatusNotFound, "Not Found"),
	},
	{
		id:       "gotify",
		prefix:   "/gotify",
		patterns: []string{"POST /gotify/message"},
		secrets:  headerValues("X-Gotify-Key"),
		ok: func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"id": 25, "appid": 5, "date": "2026-11-16T14:02:00Z"})
		},
		limited: plainLimit,
		refused: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Unauthorized", "errorCode": 401, "errorDescription": "you need to provide a valid access token or user credentials to access this api"})
		},
		gone: status(http.StatusNotFound, "404 page not found"),
	},
	{
		id:       "ntfy",
		prefix:   "/ntfy",
		patterns: []string{"POST /ntfy/{topic}", "PUT /ntfy/{topic}"},
		secrets:  pathValues("topic"),
		ok: func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{"id": "sPs71M8A2T", "time": time.Now().Unix(), "event": "message", "topic": r.PathValue("topic")})
		},
		limited: func(w http.ResponseWriter, wait time.Duration) {
			retryAfter(w, wait)
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"code": 42901, "http": 429, "error": "limit reached: too many requests", "link": "https://ntfy.sh/docs/publish/#limitations"})
		},
		refused: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusForbidden, map[string]any{"code": 40301, "http": 403, "error": "forbidden", "link": "https://ntfy.sh/docs/publish/#authentication"})
		},
		gone: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusNotFound, map[string]any{"code": 40401, "http": 404, "error": "page not found"})
		},
	},
	{
		id:       "pushbullet",
		prefix:   "/pushbullet",
		patterns: []string{"POST /pushbullet/v2/pushes"},
		secrets:  headerValues("Access-Token"),
		ok: func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Ratelimit-Limit", "16384")
			w.Header().Set("X-Ratelimit-Remaining", "16383")
			writeJSON(w, http.StatusOK, map[string]any{"iden": "ujpah72o0sjAoRtnM0jc", "active": true, "type": "note"})
		},
		limited: func(w http.ResponseWriter, wait time.Duration) {
			w.Header().Set("X-Ratelimit-Limit", "16384")
			w.Header().Set("X-Ratelimit-Remaining", "0")
			w.Header().Set("X-Ratelimit-Reset", strconv.FormatInt(time.Now().Add(wait).Unix(), 10))
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": map[string]string{"code": "too_many_requests", "type": "invalid_request", "message": "You have been ratelimited."}})
		},
		refused: func(w http.ResponseWriter) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "invalid_access_token", "type": "invalid_request", "message": "Access token is missing or invalid."}})
		},
		gone: status(http.StatusNotFound, "Not Found"),
	},
	{
		id:       "apprise",
		prefix:   "/apprise",
		patterns: []string{"POST /apprise/notify/{key...}"},
		secrets: func(r *http.Request, sent providers.Request) []string {
			out := []string{r.PathValue("key")}
			if urls := fields(sent)["urls"]; urls != "" {
				out = append(out, strings.Split(urls, ",")...)
			}
			return out
		},
		ok:      func(w http.ResponseWriter, _ *http.Request) { writeText(w, http.StatusOK, "Notification(s) sent.") },
		limited: plainLimit,
		refused: status(http.StatusUnauthorized, "Unauthorized"),
		gone:    status(http.StatusNotFound, "There is no configuration with this key."),
	},
	{
		id:       "webhook",
		prefix:   "/hook",
		patterns: []string{"/hook/{path...}"},
		secrets: func(r *http.Request, _ providers.Request) []string {
			path := r.PathValue("path")
			out := []string{path[strings.LastIndex(path, "/")+1:]}
			for _, vs := range r.URL.Query() {
				out = append(out, vs...)
			}
			for name := range r.Header {
				if strings.HasPrefix(name, "X-") {
					out = append(out, r.Header.Get(name))
				}
			}
			return out
		},
		ok:      func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		limited: plainLimit,
		refused: status(http.StatusUnauthorized, "Unauthorized"),
		gone:    status(http.StatusNotFound, "Not Found"),
	},
}
