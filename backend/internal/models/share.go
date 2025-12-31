package models

import "time"

type Share struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	ResourceType string     `json:"resource_type"` // 'trade', 'plan'
	ResourceID   int64      `json:"resource_id"`
	ShareType    string     `json:"share_type"`    // 'public', 'specific'
	Token        string     `json:"token"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	SharedWith   []int64    `json:"shared_with,omitempty"` // User IDs
}

type ShareCreate struct {
	ResourceType string   `json:"resource_type" binding:"required,oneof=trade plan"`
	ResourceID   int64    `json:"resource_id" binding:"required"`
	ShareType    string   `json:"share_type" binding:"required,oneof=public specific"`
	SharedWith   []int64  `json:"shared_with"`
	ExpiresIn    *float64 `json:"expires_in_hours"` // Optional
}
