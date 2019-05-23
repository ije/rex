package router

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/ije/gox/valid"
)

func TestRouter(t *testing.T) {
	router := New()
	router.AddValidate("email", valid.IsEmail)

	var routed int
	var expectRouted int

	router.Handle("GET", "/", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		routed++
		want := map[string]string{}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/user/:name", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		routed++
		want := map[string]string{"name": "gopher"}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/user/:name/{age:number}", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		routed++
		want := map[string]string{"name": "gopher", "age": "20"}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/:version/user", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		routed++
		want := map[string]string{"version": "v1"}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("GET", "/assets/*", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		routed++
		want := map[string]string{"path": "/scripts/main.dist.js"}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	router.Handle("POST", "/send/{email:email}", func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		routed++
		want := map[string]string{"email": "go@gmail.com"}
		if !reflect.DeepEqual(params, want) {
			t.Fatalf("invalid params: want %v, got %v", want, params)
		}
	})

	request(router, "GET", "/") // ✓
	expectRouted++

	request(router, "GET", "/user/gopher") // ✓
	expectRouted++

	request(router, "GET", "/user/gopher/20") // ✓
	expectRouted++

	request(router, "GET", "/user/gopher/20+") // x

	request(router, "GET", "/v1/user") // ✓
	expectRouted++

	request(router, "GET", "/assets/scripts/main.dist.js") // ✓
	expectRouted++

	request(router, "POST", "/send/go@gmail.com") // ✓
	expectRouted++

	request(router, "POST", "/send/gmail.com") // x

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
