package api

import (
	"crypto"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"code.posterity.life/srp/v2"
	cmw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/heyztb/lists/internal/cache"
	aes "github.com/heyztb/lists/internal/crypto"
	"github.com/heyztb/lists/internal/database"
	"github.com/heyztb/lists/internal/log"
	"github.com/heyztb/lists/internal/models"
	"golang.org/x/crypto/argon2"
)

func IdentityHandler(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(cmw.RequestIDKey).(string)
	log := log.Logger.With().Str("request_id", requestID).Logger()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Err(err).Any("request", r).Msg("failed to read request body")
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			render.Status(r, http.StatusRequestEntityTooLarge)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusRequestEntityTooLarge,
				Error:  "Content too large",
			})
			return
		}
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
	}
	req := &models.IdentityRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		log.Err(err).Bytes("body", body).Msg("failed to unmarshal request into identity request struct")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}
	user, err := database.Users(
		database.UserWhere.Identifier.EQ(req.Identifier),
	).One(r.Context(), database.DB)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Err(err).Msg("failed to fetch user from database")
		}
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}
	params := &srp.Params{
		Name:  "DH15-SHA256-Argon2",
		Group: srp.RFC5054Group3072,
		Hash:  crypto.SHA256,
		KDF: func(username string, password string, salt []byte) ([]byte, error) {
			p := []byte(username + ":" + password)
			key := argon2.IDKey(p, salt, 1, 64*1024, 4, 32)
			return key, nil
		},
	}
	verifierBytes, err := aes.AESDecrypt(aes.ServerEncryptionKey, user.Verifier)
	if err != nil {
		log.Err(err).Msg("failed to decrypt user verifier")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}
	srpServer, err := srp.NewServer(params, req.Identifier, user.Salt, verifierBytes)
	if err != nil {
		log.Err(err).Msg("failed to initialize srp server component")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}
	B := srpServer.B()
	// marshal the srp server object into binary that way we are able to cache it in memory
	// for use later on -- this is important because we must maintain the same A and B values in order to generate and validate the key proof
	srpServerBytes, err := srpServer.Save()
	if err != nil {
		log.Err(err).Msg("failed to marshal srp server object to json")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}
	err = cache.Cache.Set(
		fmt.Sprintf(cache.SRPServerKey, user.ID),
		srpServerBytes,
	)
	if err != nil {
		log.Err(err).Msg("failed to cache srp server object in memory")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &models.IdentityResponse{
		Status:          http.StatusOK,
		Salt:            hex.EncodeToString(user.Salt),
		EphemeralPublic: hex.EncodeToString(B),
	})
}
