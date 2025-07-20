package server

import (
	"net/http"
	"time"
)


var (
	ProxyPort string        
	TargetURL string        
	CacheTTL  = 5 * time.Minute 
	ShutdownTimeout = 15 * time.Second
	AutoShutdownDuration = 10 * time.Minute
)


type CacheEntry struct {
	Body         []byte       
	Headers      http.Header   
	Timestamp    time.Time     
	Expires      time.Time     
	LastModified string        
	ETag         string        
}


