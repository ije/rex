package webx

type CORS struct {
	Origin      string
	Methods     []string
	Headers     []string
	Credentials bool
	MaxAge      int // seconds
}

func PublicCORS() *CORS {
	return &CORS{
		Origin:      "*",
		Methods:     []string{"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
		Headers:     []string{"Accept", "Accept-Encoding", "Accept-Lang", "Content-Type", "Authorization", "X-Requested-With"},
		Credentials: true,
		MaxAge:      60,
	}
}
