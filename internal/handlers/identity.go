package handlers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/1Password/srp"
	"github.com/go-chi/render"
	"github.com/heyztb/lists-backend/internal/database"
	"github.com/heyztb/lists-backend/internal/models"
	"github.com/rs/zerolog/log"
)

func IdentityHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &response{
			Status:  http.StatusBadRequest,
			Message: "Bad request",
		})
	}
	req := &models.IdentityRequest{}
	if err := json.Unmarshal(body, &req); err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &response{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized",
		})
		return
	}
	user, err := database.Users(
		database.UserWhere.Identifier.EQ(req.Identifier),
	).One(r.Context(), database.DB)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &response{
			Status:  http.StatusUnauthorized,
			Message: "Unauthorized",
		})
		return
	}
	v := srp.NumberFromString(user.Verifier)
	srpServer := srp.NewServerStd(srp.KnownGroups[srp.RFC5054Group3072], v)
	if srpServer == nil {
		log.Error().Msg("failed to initialize srp server component")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &response{
			Status:  http.StatusInternalServerError,
			Message: "Internal server error",
		})
		return
	}
	A := srp.NumberFromString(req.EphemeralPublic)
	err = srpServer.SetOthersPublic(A)
	if err != nil {
		log.Err(err).Msg("invalid ephemeralPublicA from client")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &response{
			Status:  http.StatusInternalServerError,
			Message: "Internal server error",
		})
		return
	}
	B := srpServer.EphemeralPublic()
	// eagerly generating the shared key now despite the user not being fully authenticated yet
	_, err = srpServer.Key()
	if err != nil {
		log.Err(err).Msg("failed to generate shared key")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &response{
			Status:  http.StatusInternalServerError,
			Message: "Internal server error",
		})
		return
	}
	// marshal the srp server object into binary that way we are able to cache it in memory
	// for use later on -- this is important because we must maintain the same A and B values in order to generate and validate the key proof
	srpServerBytes, err := srpServer.MarshalBinary()
	if err != nil {
		log.Err(err).Msg("failed to marshal srp server object to binary")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &response{
			Status:  http.StatusInternalServerError,
			Message: "Internal server error",
		})
		return
	}
	err = database.Cache.Set(
		fmt.Sprintf(database.SRPServerKey, user.ID),
		srpServerBytes,
	)
	if err != nil {
		log.Err(err).Msg("failed to cache srp server object in memory")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &response{
			Status:  http.StatusInternalServerError,
			Message: "Internal server error",
		})
		return
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, &models.IdentityResponse{
		Salt:            user.Salt,
		EphemeralPublic: hex.EncodeToString(B.Bytes()),
	})
}
