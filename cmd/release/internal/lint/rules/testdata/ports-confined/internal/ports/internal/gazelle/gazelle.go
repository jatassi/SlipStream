package gazelle

import "net/http"

func request() { _, _ = http.NewRequest(http.MethodGet, "/", nil) }
