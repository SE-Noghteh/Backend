package storage

import (
	"net/http"
	"time"
)

var httpClient = http.Client{Timeout: 90 * time.Second}
