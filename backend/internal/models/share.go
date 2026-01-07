package models

import "time"

type Share struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	ResourceType string     `json:"resource_type"` // 'trade', 'plan', 'account', 'batch'
	ResourceID   int64      `json:"resource_id"`   // for trade, plan, account
	ResourceIDs  string     `json:"resource_ids"`  // for batch (JSON array of IDs)
	ShareType    string     `json:"share_type"`    // 'public', 'specific'
	Token        string     `json:"token"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	SharedWith   []int64    `json:"shared_with,omitempty"` // User IDs
}

type BatchResourceIDs struct {
	Trades []int64 `json:"trades"`
	Plans  []int64 `json:"plans"`
}

type ShareCreate struct {
	ResourceType string   `json:"resource_type" binding:"required,oneof=trade plan account batch"`
	ResourceID   int64    `json:"resource_id"`
	ResourceIDs  any      `json:"resource_ids"` // Can be []int64 or BatchResourceIDs
	ShareType    string   `json:"share_type" binding:"required,oneof=public specific"`
	SharedWith   []int64  `json:"shared_with"`
	ExpiresIn    *float64 `json:"expires_in_hours"` // Optional
}
