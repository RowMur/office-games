package server

import (
	"context"
	"net/http"

	"github.com/RowMur/office-tournaments/models"
)

type contextKey string

const userContextKey = contextKey("user")

func userFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authCookie, err := r.Cookie("auth")
		if err != nil {
			if err != http.ErrNoCookie {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}

		if authCookie.Value == "" {
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}

		user := data.GetUser(authCookie.Value)
		if user == nil {
			w.Header().Set("Set-Cookie", "auth=; Max-Age=0")
			http.Redirect(w, r, "/sign-in", http.StatusFound)
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
