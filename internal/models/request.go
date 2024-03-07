package models

import (
	"encoding/json"
	"time"

	"github.com/heyztb/lists-backend/internal/database"
	"github.com/volatiletech/null/v8"
)

// requests.go contains struct types for incoming request bodies

type RegistrationRequest struct {
	Identifier string `json:"identifier"`
	Salt       string `json:"salt"` // hex
	Verifier   string `json:"verifier"`
}

type IdentityRequest struct {
	Identifier      string `json:"identifier"`
	EphemeralPublic string `json:"A"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Proof      string `json:"proof"`
}

type CreateListRequest struct {
	ParentID   *uint64 `json:"parent_id"`
	Name       string  `json:"name"`
	IsFavorite bool    `json:"is_favorite"`
}

type CreateSectionRequest struct {
	Name   string `json:"name"`
	ListID uint64 `json:"list_id"`
}

type UpdateSectionRequest struct {
	Name string `json:"name"`
}

type GetItemsRequest struct {
	Completed bool    `json:"completed"`
	ListID    *uint64 `json:"list_id"`
	SectionID *uint64 `json:"section_id"`
	Label     *string `json:"label"`
}

type CreateItemRequest struct {
	ListID      *uint64        `json:"list_id"`
	SectionID   *uint64        `json:"section_id"`
	ParentID    *uint64        `json:"parent_id"`
	Content     string         `json:"content"`
	Description *string        `json:"description"`
	Labels      *[]string      `json:"labels"`
	Priority    *int           `json:"priority"`
	DueDate     *time.Time     `json:"due_date"`
	DueString   *string        `json:"due_string"`
	Duration    *time.Duration `json:"duration"`
}

type UpdateItemRequest struct {
	Content     *string        `json:"content"`
	Description *string        `json:"description"`
	Labels      *[]string      `json:"labels"`
	Position    *int           `json:"position"`
	Priority    *int           `json:"priority"`
	DueDate     *time.Time     `json:"due_date"`
	DueString   *string        `json:"due_string"`
	Duration    *time.Duration `json:"duration"`
}

func (r *UpdateItemRequest) UpdateItem(item *database.Item) error {
	if r.Content != nil {
		item.Content = *r.Content
	}

	if r.Description != nil {
		item.Description = null.StringFromPtr(r.Description)
	}

	if r.Labels != nil {
		labelsJson, err := json.Marshal(*r.Labels)
		if err != nil {
			return err
		}
		item.Labels = null.JSONFrom(labelsJson)
	}

	if r.Priority != nil {
		item.Priority = *r.Priority
	}

	// TODO: Due dates

	return nil
}
