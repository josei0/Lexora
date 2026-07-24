package handler

import (
	"strings"
	"testing"
	"time"
)

func TestHostCookieInvariants(t *testing.T) {
	a := New(nil, nil, nil, nil, 0, true, nil) // secure = prod

	c := a.cookie("val", time.Hour)
	for _, want := range []string{"__Host-lexora_rt=", "Path=/;", "Secure", "HttpOnly", "SameSite=Strict"} {
		if !strings.Contains(c, want) {
			t.Fatalf("cookie kehilangan %q: %s", want, c)
		}
	}
	if strings.Contains(c, "Domain") {
		t.Fatalf("__Host- tak boleh set Domain: %s", c)
	}
	if !strings.Contains(a.adminCookie("v", time.Hour), "__Host-lexora_admin_rt=") {
		t.Fatalf("nama cookie admin salah (sesi harus terpisah)")
	}

	// dev: nama polos
	dev := New(nil, nil, nil, nil, 0, false, nil).cookie("v", time.Hour)
	if strings.Contains(dev, "__Host-") {
		t.Fatalf("dev cookie tak boleh __Host- tanpa Secure: %s", dev)
	}
}
