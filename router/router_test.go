package router

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/ije/gox/valid"
)

func TestRouter(t *testing.T) {
	router := New()
	router.SetValidateFn("number", valid.IsNumber)
	router.SetValidateFn("email", valid.IsEmail)

	var routed int
	var expectRouted int

	router.Handle("GET", "/", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/posts/:slug", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"slug", "hello-world"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/post/:id[number]", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"id", "123"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/:version/posts", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"version", "ver2"}, {"ver", "ver2"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/:ver/users", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"version", "v2"}, {"ver", "v2"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/assets/*", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"path", "/scripts/main.dist.js"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("POST", "/(works|work)/:id", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"id", "rex"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("POST", "/subs/:email[email]", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"email", "rex@gmail.com"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	request(router, "GET", "/") // ✓
	expectRouted++

	request(router, "GET", "/posts/hello-world") // ✓
	expectRouted++

	request(router, "GET", "/post/123") // ✓
	expectRouted++

	request(router, "GET", "/post/123+") // x

	request(router, "GET", "/ver2/posts") // ✓
	expectRouted++

	request(router, "GET", "/v2/users") // ✓
	expectRouted++

	request(router, "GET", "/assets/scripts/main.dist.js") // ✓
	expectRouted++

	request(router, "POST", "/work/rex") // ✓
	expectRouted++

	request(router, "POST", "/works/rex") // ✓
	expectRouted++

	request(router, "POST", "/subs/rex@gmail.com") // ✓
	expectRouted++

	request(router, "POST", "/subs/gmail.com") // x

	if routed != expectRouted {
		t.Fatalf("routed %d but expect %d", routed, expectRouted)
	}
}

func request(router *Router, method string, path string) {
	req, _ := http.NewRequest(method, path, nil)
	router.ServeHTTP(new(nullWriter), req)
}

type nullWriter struct{}

func (m *nullWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *nullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *nullWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *nullWriter) WriteHeader(int) {}
