package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
)

// OIDCMiddleware validates JWT access tokens against an OIDC provider (Auth0).
type OIDCMiddleware struct {
	verifier            *oidc.IDTokenVerifier
	resourceMetadataURL string
	logger              *zap.Logger
}

// NewOIDCMiddleware creates an OIDC middleware that verifies JWTs using the
// provider's JWKS endpoint (discovered via OIDC discovery from issuerURL).
// The audience parameter is checked against the JWT's "aud" claim.
func NewOIDCMiddleware(ctx context.Context, issuerURL, audience, resourceMetadataURL string, logger *zap.Logger) (*OIDCMiddleware, error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, err
	}

	// go-oidc's ClientID config field is checked against the "aud" claim.
	// For Auth0 custom APIs, the audience (API identifier) appears in "aud".
	verifier := provider.Verifier(&oidc.Config{ClientID: audience})

	return &OIDCMiddleware{
		verifier:            verifier,
		resourceMetadataURL: resourceMetadataURL,
		logger:              logger,
	}, nil
}

// Authenticate is an http.Handler middleware that validates Bearer JWT tokens.
// On failure it returns 401 with a WWW-Authenticate header pointing to the
// protected resource metadata endpoint (RFC 9728).
func (m *OIDCMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		m.logger.Debug("OIDC auth check",
			zap.String("path", r.URL.Path),
			zap.Bool("has_bearer", strings.HasPrefix(auth, "Bearer ")),
			zap.Int("auth_header_len", len(auth)),
		)

		if !strings.HasPrefix(auth, "Bearer ") {
			m.logger.Warn("OIDC: missing or invalid Bearer token",
				zap.String("path", r.URL.Path),
				zap.String("auth_header_prefix", safePrefix(auth, 20)),
			)
			m.unauthorized(w)
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")

		m.logger.Debug("OIDC: verifying token",
			zap.Int("token_len", len(token)),
			zap.String("token_prefix", safePrefix(token, 20)),
		)

		idToken, err := m.verifier.Verify(r.Context(), token)
		if err != nil {
			m.logger.Warn("OIDC token verification failed",
				zap.Error(err),
				zap.String("component", "oidc"),
				zap.Int("token_len", len(token)),
				zap.String("token_prefix", safePrefix(token, 20)),
			)
			m.unauthorized(w)
			return
		}

		var claims struct {
			Email string `json:"email"`
			Sub   string `json:"sub"`
		}
		if err := idToken.Claims(&claims); err != nil {
			m.logger.Error("failed to parse JWT claims", zap.Error(err))
			m.unauthorized(w)
			return
		}

		// Write identity into the shared RequestInfo (placed in context by HTTPLogger).
		if ri := RequestInfoFromContext(r.Context()); ri != nil {
			ri.UserID = claims.Sub
			if claims.Email != "" {
				ri.SessionID = claims.Email
			}
		}

		// Also set context values so handlers can read user identity directly.
		ctx := r.Context()
		ctx = context.WithValue(ctx, UserIDKey, claims.Sub)
		if claims.Email != "" {
			ctx = context.WithValue(ctx, SessionIDKey, claims.Email)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// safePrefix returns the first n characters of s (or all of s if shorter).
// Used for logging token prefixes without exposing the full token.
func safePrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (m *OIDCMiddleware) unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+m.resourceMetadataURL+`"`)
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
