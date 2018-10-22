package http

import (
	"net/http"

	"github.com/Pigmice2733/peregrine-backend/internal/store"
	jwt "github.com/dgrijalva/jwt-go"
)

type contextKey string

const (
	keyRolesContext   contextKey = "pigmice_roles"
	keySubjectContext contextKey = "pigmice_subject"
)

// Claims holds the standard jwt claims plus the pigmice roles claim.
type Claims struct {
	Roles store.Roles `json:"pigmiceRoles"`
	jwt.StandardClaims
}

// GetSubject retrieves the subject from the http context.
func GetSubject(r *http.Request) string {
	contextSubject := r.Context().Value(keySubjectContext)
	if contextSubject == nil {
		return ""
	}

	sub, ok := contextSubject.(string)
	if !ok {
		return ""
	}

	return sub
}

// GetRoles retrieves the roles from the http context.
func GetRoles(r *http.Request) store.Roles {
	contextRoles := r.Context().Value(keyRolesContext)
	if contextRoles == nil {
		return store.Roles{}
	}

	roles, ok := contextRoles.(store.Roles)
	if !ok {
		return store.Roles{}
	}

	return roles
}
