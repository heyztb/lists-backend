package middleware

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/go-chi/render"
	"github.com/heyztb/lists-backend/internal/crypto"
	"github.com/rs/zerolog/log"
)

// Decryption middleware reads the user's session key from the request context and uses it to decrypt the incoming request body.
func Decryption(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := r.Context().Value(SessionKeyCtxKey).([]byte)
		if !ok {
			log.Error().Msg("decrypt middleware reached without session key")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, &authMiddlewareResponse{
				Status:  http.StatusBadRequest,
				Message: "Unauthorized",
			})
		}
		encodedBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Err(err).Msg("failed to read request body")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, &authMiddlewareResponse{
				Status:  http.StatusUnauthorized,
				Message: "Unauthorized",
			})
		}
		encryptedBody, err := base64.RawStdEncoding.DecodeString(string(encodedBody))
		if err != nil {
			log.Err(err).Msg("failed to read request body")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, &authMiddlewareResponse{
				Status:  http.StatusUnauthorized,
				Message: "Unauthorized",
			})
		}
		decryptedBody, err := crypto.AESDecrypt(key, encryptedBody)
		if err != nil {
			log.Err(err).Msg("failed to decrypt request body")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, &authMiddlewareResponse{
				Status:  http.StatusUnauthorized,
				Message: "Unauthorized",
			})
		}
		r.ContentLength = int64(len(decryptedBody))
		r.Body = io.NopCloser(bytes.NewBuffer(decryptedBody))
		next.ServeHTTP(w, r)
	})
}
