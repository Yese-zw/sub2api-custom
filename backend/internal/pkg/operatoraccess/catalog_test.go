package operatoraccess

import "testing"

func TestResolve(t *testing.T) {
	cases := []struct {
		method, path, perm string
		delegable          bool
	}{
		{"GET", "/api/v1/admin/users", "users.view", true},
		{"PUT", "/api/v1/admin/users/:id", "users.edit", true},
		{"DELETE", "/api/v1/admin/users/:id", "users.delete", true},
		{"POST", "/api/v1/admin/users/:id/balance", "users.balance", true},
		{"POST", "/api/v1/admin/users/:id/auth-identities", "", false},
		{"GET", "/api/v1/admin/dashboard/profit-trend", "profit.view", true},
		{"GET", "/api/v1/admin/accounts/data", "", false},
		{"POST", "/api/v1/admin/accounts/:id/refresh", "accounts.edit", true},
		{"GET", "/api/v1/admin/settings", "", false},
		{"PUT", "/api/v1/admin/settings", "", false},
		{"GET", "/api/v1/admin/backups", "", false},
		{"GET", "/api/v1/admin/users-evil", "", false},
		{"GET", "/api/v1/admin/payment/config", "", false},
		{"POST", "/api/v1/admin/payment/orders/:id/refund", "orders.refund", true},
		{"GET", "/api/v1/other", "", false},
	}
	for _, c := range cases {
		perm, ok := Resolve(c.method, c.path)
		if perm != c.perm || ok != c.delegable {
			t.Errorf("%s %s = (%q,%v), want (%q,%v)", c.method, c.path, perm, ok, c.perm, c.delegable)
		}
	}
}

func TestOverridesReferenceValidPermissions(t *testing.T) {
	for key, perm := range overrides {
		if perm == adminOnly {
			continue
		}
		if _, ok := validPermissions[perm]; !ok {
			t.Errorf("override %q targets unknown permission %q", key, perm)
		}
	}
}

func TestSanitize(t *testing.T) {
	got := Sanitize([]string{"users.edit", "bogus.x", "users.edit", " accounts.delete "})
	want := []string{"accounts.delete", "accounts.view", "users.edit", "users.view"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestAllowed(t *testing.T) {
	perms := []string{"users.view"}
	if !Allowed(perms, "GET", "/api/v1/admin/users") {
		t.Error("view should be allowed")
	}
	if Allowed(perms, "PUT", "/api/v1/admin/users/:id") {
		t.Error("edit should be denied")
	}
	if !Allowed(perms, "GET", "/api/v1/admin/groups/all") {
		t.Error("lookup should be allowed with any permission")
	}
	if Allowed(nil, "GET", "/api/v1/admin/groups/all") {
		t.Error("lookup denied without permissions")
	}
	if !Allowed(nil, "GET", "/api/v1/admin/compliance") {
		t.Error("baseline allowed")
	}
}
