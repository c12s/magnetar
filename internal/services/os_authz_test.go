package services

import (
	"context"
	"errors"
	"testing"

	"github.com/c12s/magnetar/internal/domain"
	"github.com/golang-jwt/jwt/v5"
)

const testKey = "test-key"

func ctxWithPermissions(t *testing.T, perms string) context.Context {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"permissions": perms}).SignedString([]byte(testKey))
	if err != nil {
		t.Fatal(err)
	}
	return context.WithValue(context.Background(), "authz-token", token)
}

func TestAuthorizeOsUpdate(t *testing.T) {
	repo := &fakeNodeRepo{
		nodes: map[string]bool{"n1": true, "pool1": true},
		orgs:  map[string]string{"n1": "c12s"},
	}
	svc, _ := NewOsService(repo)
	authz := NewAuthZService(testKey)

	cases := []struct {
		name  string
		perms string
		node  string
		want  error
	}{
		{"node grant", "os.update|node|n1", "n1", nil},
		{"org grant covers the node's org", "os.update|org|c12s", "n1", nil},
		{"other org's grant", "os.update|org|other", "n1", domain.ErrForbidden},
		{"other node's grant", "os.update|node|n2", "n1", domain.ErrForbidden},
		{"org grant does not cover pool nodes", "os.update|org|c12s", "pool1", domain.ErrForbidden},
		{"node grant on a pool node", "os.update|node|pool1", "pool1", nil},
		{"unknown node", "os.update|org|c12s", "ghost", domain.ErrNodeNotFound},
		{"another permission", "node.delete|org|c12s", "n1", domain.ErrForbidden},
	}
	for _, c := range cases {
		err := svc.AuthorizeOsUpdate(ctxWithPermissions(t, c.perms), authz, domain.NodeId{Value: c.node})
		if !errors.Is(err, c.want) && !(err == nil && c.want == nil) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
}
