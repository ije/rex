package router

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/ije/gox/valid"
)

func TestRouter(t *testing.T) {
	router := New()

	router.Validates(map[string]Validate{
		"number": valid.IsNumber,
		"email":  valid.IsEmail,
	})

	routed := 0
	expectRouted := 0

	router.Handle("GET", "/", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/users/:name", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"name", "gopher"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/user/{id:number}", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"id", "123"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/:version/user", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"version", "v1"}}
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

	router.Handle("POST", "/subs/ {email: email} ", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"email", "go@gmail.com"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("POST", "/(repos|repo)/:id", func(w http.ResponseWriter, r *http.Request, params Params) {
		routed++
		want := Params{{"id", "rex"}}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	request(router, "GET", "/") // ✓
	expectRouted++

	request(router, "GET", "/users/gopher") // ✓
	expectRouted++

	request(router, "GET", "/user/123") // ✓
	expectRouted++

	request(router, "GET", "/user/123+") // x

	request(router, "GET", "/v1/user") // ✓
	expectRouted++

	request(router, "GET", "/assets/scripts/main.dist.js") // ✓
	expectRouted++

	request(router, "POST", "/subs/go@gmail.com") // ✓
	expectRouted++

	request(router, "POST", "/subs/gmail.com") // x

	request(router, "POST", "/repo/rex") // ✓
	expectRouted++

	request(router, "POST", "/repos/rex") // ✓
	expectRouted++

	if routed != expectRouted {
		t.Fatalf("routed %d but expect %d", routed, expectRouted)
	}
}

type vResponseWriter struct{}

func (m *vResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

func (m *vResponseWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (m *vResponseWriter) WriteString(s string) (n int, err error) {
	return len(s), nil
}

func (m *vResponseWriter) WriteHeader(int) {}

func request(router *Router, method string, path string) {
	w := new(vResponseWriter)
	req, _ := http.NewRequest(method, path, nil)
	router.ServeHTTP(w, req)
}
