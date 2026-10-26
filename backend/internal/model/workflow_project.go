package model

import "time"

type WorkflowProject struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID string    `gorm:"size:80;not null;index:idx_workflow_projects_user_project,unique" json:"project_id"`
	UserID    uint      `gorm:"not null;index;index:idx_workflow_projects_user_project,unique" json:"user_id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	State     string    `gorm:"type:jsonb;not null;default:'{}'" json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WorkflowTemplate stores a reusable workflow snapshot. System templates are
// shared by all users; user templates are private to their owner.
type WorkflowTemplate struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	Summary   string    `gorm:"size:500" json:"summary"`
	Scope     string    `gorm:"size:20;not null;index" json:"scope"`
	State     string    `gorm:"type:jsonb;not null;default:'{}'" json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
