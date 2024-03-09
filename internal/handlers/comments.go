package handlers

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/heyztb/lists-backend/internal/crypto"
	"github.com/heyztb/lists-backend/internal/database"
	"github.com/heyztb/lists-backend/internal/log"
	"github.com/heyztb/lists-backend/internal/middleware"
	"github.com/heyztb/lists-backend/internal/models"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

func GetCommentsHandler(w http.ResponseWriter, r *http.Request) {
	userID, _, key, err := middleware.ReadContext(r)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}

	listID := r.URL.Query().Get("list_id")
	itemID := r.URL.Query().Get("item_id")

	if listID == "" && itemID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	if listID != "" && itemID != "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	queryMods := []qm.QueryMod{
		database.CommentWhere.UserID.EQ(userID),
	}

	if listID != "" {
		listIDInt, err := strconv.ParseInt(listID, 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusBadRequest,
				Error:  "Bad request",
			})
			return
		}
		listIDUint := uint64(listIDInt)
		queryMods = append(queryMods, database.CommentWhere.ListID.EQ(null.Uint64From(listIDUint)))
	}

	if itemID != "" {
		itemIDInt, err := strconv.ParseInt(itemID, 10, 64)
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusBadRequest,
				Error:  "Bad request",
			})
			return
		}
		itemIDUint := uint64(itemIDInt)
		queryMods = append(queryMods, database.CommentWhere.ListID.EQ(null.Uint64From(itemIDUint)))
	}

	comments, err := database.Comments(queryMods...).All(r.Context(), database.DB)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusNotFound,
				Error:  "Not found",
			})
			return
		}
		log.Err(err).Msg("failed to fetch comments from database")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	encryptedJSON, err := crypto.AESEncrypt(key, comments)
	if err != nil {
		log.Err(err).Msg("failed to encrypt comments")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &models.SuccessResponse{
		Status: http.StatusOK,
		Data:   base64.RawStdEncoding.EncodeToString(encryptedJSON),
	})
}

func GetCommentHandler(w http.ResponseWriter, r *http.Request) {
	userID, _, key, err := middleware.ReadContext(r)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}

	commentID := chi.URLParam(r, "comment")
	commentIDInt, err := strconv.ParseInt(commentID, 10, 64)
	if err != nil {
		log.Err(err).Str("comment", commentID).Msg("invalid comment ID")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	queryMods := []qm.QueryMod{
		database.CommentWhere.ID.EQ(uint64(commentIDInt)),
		database.CommentWhere.UserID.EQ(userID),
	}

	comment, err := database.Comments(queryMods...).One(r.Context(), database.DB)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusNotFound,
				Error:  "Not found",
			})
			return
		}
		log.Err(err).Msg("failed to fetch comment from database")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	encryptedJSON, err := crypto.AESEncrypt(key, comment)
	if err != nil {
		log.Err(err).Msg("failed to encrypt comments")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &models.SuccessResponse{
		Status: http.StatusOK,
		Data:   base64.RawStdEncoding.EncodeToString(encryptedJSON),
	})
}

func CreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	userID, _, key, err := middleware.ReadContext(r)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Err(err).Any("request", r).Msg("failed to read request body")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	request := &models.CreateCommentRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		log.Err(err).Bytes("body", body).Msg("failed to unmarshal request into CreateCommentRequest struct")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	if request.ItemID == "" && request.ListID == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	if request.ItemID != "" && request.ListID != "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	if request.Content == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	comment := &database.Comment{
		UserID:  userID,
		Content: request.Content,
	}

	if request.ItemID != "" {
		itemIDInt, err := strconv.ParseInt(request.ItemID, 10, 64)
		if err != nil {
			log.Err(err).Str("item", request.ItemID).Msg("invalid item ID")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusBadRequest,
				Error:  "Bad request",
			})
			return
		}
		comment.ItemID = null.Uint64From(uint64(itemIDInt))
	}

	if request.ListID != "" {
		listIDInt, err := strconv.ParseInt(request.ListID, 10, 64)
		if err != nil {
			log.Err(err).Str("list", request.ListID).Msg("invalid list ID")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusBadRequest,
				Error:  "Bad request",
			})
			return
		}
		comment.ListID = null.Uint64From(uint64(listIDInt))
	}

	if err = comment.Insert(r.Context(), database.DB, boil.Infer()); err != nil {
		log.Err(err).Msg("failed to save comment to database")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	encryptedJSON, err := crypto.AESEncrypt(key, comment)
	if err != nil {
		log.Err(err).Msg("failed to encrypt comment")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &models.SuccessResponse{
		Status: http.StatusOK,
		Data:   base64.RawStdEncoding.EncodeToString(encryptedJSON),
	})
}

func UpdateCommentHandler(w http.ResponseWriter, r *http.Request) {
	userID, _, key, err := middleware.ReadContext(r)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}

	commentID := chi.URLParam(r, "comment")
	commentIDInt, err := strconv.ParseInt(commentID, 10, 64)
	if err != nil {
		log.Err(err).Str("comment", commentID).Msg("invalid comment ID")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Err(err).Any("request", r).Msg("failed to read request body")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	request := &models.UpdateCommentRequest{}
	if err := json.Unmarshal(body, &request); err != nil {
		log.Err(err).Bytes("body", body).Msg("failed to unmarshal request into UpdateCommentRequest struct")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	if request.Content == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	queryMods := []qm.QueryMod{
		database.CommentWhere.ID.EQ(uint64(commentIDInt)),
		database.CommentWhere.UserID.EQ(userID),
	}

	comment, err := database.Comments(queryMods...).One(r.Context(), database.DB)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, &models.ErrorResponse{
				Status: http.StatusNotFound,
				Error:  "Not found",
			})
			return
		}
		log.Err(err).Msg("failed to fetch comment from database")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	encryptedJSON, err := crypto.AESEncrypt(key, comment)
	if err != nil {
		log.Err(err).Msg("failed to encrypt comment")
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusInternalServerError,
			Error:  "Internal server error",
		})
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, &models.SuccessResponse{
		Status: http.StatusOK,
		Data:   base64.RawStdEncoding.EncodeToString(encryptedJSON),
	})
}

func DeleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	userID, _, _, err := middleware.ReadContext(r)
	if err != nil {
		render.Status(r, http.StatusUnauthorized)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusUnauthorized,
			Error:  "Unauthorized",
		})
		return
	}

	commentID := chi.URLParam(r, "comment")
	commentIDInt, err := strconv.ParseInt(commentID, 10, 64)
	if err != nil {
		log.Err(err).Str("comment", commentID).Msg("invalid comment ID")
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusBadRequest,
			Error:  "Bad request",
		})
		return
	}

	queryMods := []qm.QueryMod{
		database.CommentWhere.ID.EQ(uint64(commentIDInt)),
		database.CommentWhere.UserID.EQ(userID),
	}

	rowsAff, err := database.Comments(queryMods...).DeleteAll(r.Context(), database.DB)
	if rowsAff == 0 {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, &models.ErrorResponse{
			Status: http.StatusNotFound,
			Error:  "Not found",
		})
		return
	}

	render.NoContent(w, r)
}
