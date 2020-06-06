package kitphishr

// custom struct for parsing phishtank urls
type PhishUrls struct {
	URL string `json:"url"`
}

type Response struct {
	StatusCode    int64
	Body          []byte
	URL           string
	ContentLength int64
	ContentType   string
}
