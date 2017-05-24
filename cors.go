package webx

type CORS struct {
	Origin      string
	Methods     string
	Headers     string
	Credentials bool
	MaxAge      int
}

func PublicCORS() *CORS {
	return &CORS{
		Origin:      "*",
		Methods:     "HEAD,GET,POST,PUT,PATCH,DELETE",
		Headers:     "Accept,Accept-Encoding,Accept-Lang,Content-Type,Authorization,X-Requested-With,X-Method",
		Credentials: true,
		MaxAge:      60,
	}
}
