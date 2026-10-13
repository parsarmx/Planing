package rest_test

import (
	"bytes"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"io"
	"net/http/httptest"
	"planningpoker/app"
	"planningpoker/handler/controllers/rest"
	"planningpoker/handler/routers"
	"planningpoker/internal/config"
	"planningpoker/internal/producers"
	"planningpoker/internal/realtime"
	"planningpoker/internal/repositories"
	"planningpoker/internal/services"
	"testing"
)

func TestHTTPAuthCookieOriginAndPermission(t *testing.T) {
	s, err := repositories.NewFile("")
	if err != nil {
		t.Fatal(err)
	}
	h := realtime.New()
	cfg := &config.Config{Environment: "development", Origins: []string{"https://poker.test"}}
	a := app.Framework(zap.NewNop())
	auth := services.NewAuth(s, producers.DevSMS{}, cfg, h)
	rooms := services.NewRooms(s, h)
	routers.Register(a, rest.New(auth, rooms, h, cfg), cfg)
	call := func(method, path, body, cookie, origin string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := a.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, _ := io.ReadAll(res.Body)
		if res.StatusCode != status {
			t.Fatalf("%s %s expected %d got %d: %s", method, path, status, res.StatusCode, data)
		}
		return data
	}
	call("GET", "/api/rooms", "", "", "", 401)
	call("POST", "/api/auth/otp", `{"phone":"09120000001"}`, "", "https://evil.test", 403)
	data := call("POST", "/api/auth/otp", `{"phone":"09120000001"}`, "", "https://poker.test", 200)
	var ch map[string]any
	if err = json.Unmarshal(data, &ch); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"phone": ch["phone"], "code": ch["dev_code"]})
	req := httptest.NewRequest("POST", "/api/auth/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := a.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.StatusCode)
	}
	cookies := res.Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != 3 || cookies[0].Path != "/" {
		t.Fatalf("unsafe session cookie: %+v", cookies)
	}
	cookie := cookies[0].Name + "=" + cookies[0].Value
	call("GET", "/api/me", "", cookie, "", 200)
	call("PATCH", "/api/me", `{"name":"QA Host","avatar":"blue","role":"QA"}`, cookie, "https://poker.test", 200)
	call("POST", "/api/rooms", `{"name":"QA Room","deck":"fibonacci"}`, cookie, "https://poker.test", 201)
	call("POST", "/api/auth/logout", `{}`, cookie, "https://poker.test", 204)
	call("GET", "/api/me", "", cookie, "", 401)
	req = httptest.NewRequest("POST", "/api/auth/otp", bytes.NewBufferString("phone=09120000001"))
	req.Header.Set("Content-Type", "text/plain")
	res, err = a.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != fiber.StatusUnsupportedMediaType {
		t.Fatal("non-JSON mutation accepted")
	}
}

func TestLANOriginsAcceptMutationsAndRejectOtherHosts(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOWED_ORIGINS", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	s, err := repositories.NewFile("")
	if err != nil {
		t.Fatal(err)
	}
	h := realtime.New()
	a := app.Framework(zap.NewNop())
	auth := services.NewAuth(s, producers.DevSMS{}, cfg, h)
	rooms := services.NewRooms(s, h)
	routers.Register(a, rest.New(auth, rooms, h, cfg), cfg)
	for _, tc := range []struct {
		origin string
		status int
	}{
		{"http://192.168.15.87:8080", 400},
		{"http://192.168.15.87:5173", 400},
		{"http://127.0.0.1:8080", 400},
		{"http://localhost:5173", 400},
		{"http://192.168.14.1:8080", 403},
		{"https://evil.test", 403},
	} {
		req := httptest.NewRequest("POST", "/api/auth/otp", bytes.NewBufferString(`{"phone":"invalid"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", tc.origin)
		res, err := a.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != tc.status {
			t.Errorf("origin %s: expected %d, got %d: %s", tc.origin, tc.status, res.StatusCode, data)
		}
	}
}
