package api

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/heyztb/lists/internal/cache"
	"github.com/heyztb/lists/internal/crypto"
	"github.com/heyztb/lists/internal/database"
	"github.com/heyztb/lists/internal/log"
	"github.com/heyztb/lists/internal/middleware"
	"github.com/heyztb/lists/internal/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

func UpdateNameHandler(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(cmw.RequestIDKey).(string)
	log := log.Logger.With().Str("request_id", requestID).Logger()
	userID, _, _, err := middleware.ReadContext(r)
	if err != nil {
		log.Err(err).Msg("error reading session context")
		render.Status(r, http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := database.FindUser(r.Context(), database.DB, userID)
	if err != nil {
		log.Err(err).Msg("error finding user from database")
		render.Status(r, http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		name = "John Doe"
	}
	if !strings.EqualFold(user.Name.String, name) {
		user.Name.SetValid(name)
		_, err = user.Update(r.Context(), database.DB, boil.Whitelist(database.UserColumns.Name))
		if err != nil {
			log.Err(err).Msg("error updating user's name in database")
			render.Status(r, http.StatusInternalServerError)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	render.Status(r, http.StatusOK)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Name updated successfully"))
}

func UpdateAvatarHandler(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(cmw.RequestIDKey).(string)
	log := log.Logger.With().Str("request_id", requestID).Logger()
	userID, _, _, err := middleware.ReadContext(r)
	if err != nil {
		log.Err(err).Msg("error reading session context")
		render.Status(r, http.StatusInternalServerError)
		w.Header().Add("HX-Redirect", "/login")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := database.FindUser(r.Context(), database.DB, userID)
	if err != nil {
		log.Err(err).Msg("error finding user from database")
		render.Status(r, http.StatusInternalServerError)
		w.Header().Add("HX-Redirect", "/500")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	avatar, header, err := r.FormFile("avatar")
	if err != nil {
		log.Err(err).Msg("error reading avatar from form data")
		render.Status(r, http.StatusBadRequest)
		w.Header().Add("HX-Redirect", "/app/settings")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer avatar.Close()
	buf := bytes.NewBuffer(nil)
	if _, err = io.Copy(buf, avatar); err != nil {
		log.Err(err).Msg("error reading avatar data into buffer")
		render.Status(r, http.StatusInternalServerError)
		w.Header().Add("HX-Redirect", "/500")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = os.MkdirAll(fmt.Sprintf("/var/lib/lists/images/%s", user.ID), fs.FileMode(os.O_RDWR))
	if err != nil {
		log.Err(err).Msg("error creating avatar path")
		render.Status(r, http.StatusInternalServerError)
		w.Header().Add("HX-Redirect", "/500")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	filename := filepath.Join(fmt.Sprintf("/var/lib/lists/images/%s", user.ID), header.Filename)
	err = os.WriteFile(filename, buf.Bytes(), 0600)
	if err != nil {
		log.Err(err).Msg("error writing file to disk")
		render.Status(r, http.StatusInternalServerError)
		w.Header().Add("HX-Redirect", "/500")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user.ProfilePicture.SetValid(filename)
	_, err = user.Update(r.Context(), database.DB, boil.Whitelist(database.UserColumns.ProfilePicture))
	if err != nil {
		log.Err(err).Msg("error updating user in database")
		render.Status(r, http.StatusInternalServerError)
		w.Header().Add("HX-Redirect", "/500")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("HX-Redirect", "/app/settings")
	w.WriteHeader(http.StatusOK)
}

func UpdateVerifierHandler(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(cmw.RequestIDKey).(string)
	log := log.Logger.With().Str("request_id", requestID).Logger()
	userID, _, _, err := middleware.ReadContext(r)
	if err != nil {
		log.Err(err).Msg("error reading session context")
		render.Status(r, http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user, err := database.FindUser(r.Context(), database.DB, userID)
	if err != nil {
		log.Err(err).Msg("error finding user from database")
		render.Status(r, http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	verifier := r.FormValue("v")
	if len(verifier) != 768 {
		render.Status(r, http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	verifierBytes, err := hex.DecodeString(verifier)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	salt := r.FormValue("s")
	if len(salt) != 24 {
		render.Status(r, http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	identifier := r.FormValue("identifier")
	var email *mail.Address = nil
	if identifier != "" && !strings.EqualFold(user.Identifier, identifier) {
		email, err = mail.ParseAddress(identifier)
		if err != nil {
			log.Err(err).Str("new identifier", identifier).Msg("error parsing new identifier value as valid email address")
			render.Status(r, http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = database.Users(database.UserWhere.Identifier.EQ(email.Address)).One(r.Context(), database.DB)
		if err == nil {
			log.Error().Msgf("cannot update user %s to identifier %s: already in use", userID, email.Address)
			render.Status(r, http.StatusBadRequest)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	encryptedVerifier, err := crypto.AESEncryptTableData(crypto.ServerEncryptionKey, verifierBytes)
	if err != nil {
		log.Err(err).Msg("failed to encrypt new user verifier")
		render.Status(r, http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	user.Salt = saltBytes
	user.Verifier = encryptedVerifier
	whitelist := []string{database.UserColumns.Salt, database.UserColumns.Verifier}
	if email != nil {
		user.Identifier = email.Address
		whitelist = append(whitelist, database.UserColumns.Identifier)
	}
	_, err = user.Update(r.Context(), database.DB, boil.Whitelist(whitelist...))
	if err != nil {
		log.Err(err).Msg("error updating user in database")
		render.Status(r, http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// delete session key from redis and invalidate cookie
	err = cache.Redis.Del(
		r.Context(),
		fmt.Sprintf(cache.RedisSessionKeyPrefix, userID),
	).Err()
	if err != nil {
		log.Err(err).Msg("error deleting shared key from redis")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "lists-session",
		Value:    "",
		Path:     "/",
		Domain:   "localhost", // TODO: change this
		Expires:  time.Unix(0, 0),
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

func DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	requestID, _ := r.Context().Value(cmw.RequestIDKey).(string)
	log := log.Logger.With().Str("request_id", requestID).Logger()
	userID, _, _, err := middleware.ReadContext(r)
	if err != nil {
		log.Err(err).Msg("error reading session context on logout")
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}
	err = cache.Redis.Del(
		r.Context(),
		fmt.Sprintf(cache.RedisSessionKeyPrefix, userID),
	).Err()
	if err != nil {
		log.Err(err).Msg("error deleting shared key from redis")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}
	_, err = database.Users(
		database.UserWhere.ID.EQ(userID),
	).DeleteAll(r.Context(), database.DB)
	if err != nil {
		log.Err(err).Msg("error deleting user from database")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "lists-session",
		Value:    "",
		Path:     "/",
		Domain:   "localhost", // TODO: change this
		Expires:  time.Unix(0, 0),
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
	})
	render.Status(r, http.StatusNoContent)
	// We trigger this endpoint with a DELETE request from an htmx augmented
	// button in the settings page of our app This header will trigger a redirect
	// on the client to the landing page
	w.Header().Add("HX-Redirect", "/")
	w.WriteHeader(http.StatusNoContent)
}
