package model

import "time"

type SceneStyleTag struct {
	ID          uint                  `gorm:"primaryKey" json:"id"`
	Name        string                `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Description string                `gorm:"type:varchar(500);default:''" json:"description"`
	Sort        int                   `gorm:"default:0" json:"sort"`
	References  []SceneStyleReference `gorm:"foreignKey:TagID" json:"references,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

func (SceneStyleTag) TableName() string {
	return "scene_style_tags"
}

type SceneStyleReference struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SceneID    uint      `gorm:"index;not null;default:0" json:"scene_id"`
	TagID      uint      `gorm:"index;not null" json:"tag_id"`
	Title      string    `gorm:"type:varchar(255);default:''" json:"title"`
	ImageURL   string    `gorm:"type:varchar(500);not null" json:"image_url"`
	PromptHint string    `gorm:"type:text;default:''" json:"prompt_hint"`
	Sort       int       `gorm:"default:0" json:"sort"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (SceneStyleReference) TableName() string {
	return "scene_style_references"
}

type SceneGenerationTask struct {
	ID                   uint                   `gorm:"primaryKey" json:"id"`
	UserID               uint                   `gorm:"index;not null" json:"user_id"`
	TaskID               string                 `gorm:"type:varchar(64);uniqueIndex;not null" json:"task_id"`
	Status               string                 `gorm:"type:varchar(20);default:'queued'" json:"status"`
	ErrorMessage         string                 `gorm:"type:text;default:''" json:"error_message"`
	IsSavedToDesign      bool                   `gorm:"default:false" json:"is_saved_to_design"`
	SavedAt              *time.Time             `json:"saved_at"`
	StyleName            string                 `gorm:"type:varchar(100);default:''" json:"style_name"`
	ReferenceImageURL    string                 `gorm:"type:varchar(500);default:''" json:"reference_image_url"`
	ReferenceImageTitle  string                 `gorm:"type:varchar(255);default:''" json:"reference_image_title"`
	ProductCategory      string                 `gorm:"type:varchar(100);default:''" json:"product_category"`
	CountryCode          string                 `gorm:"type:varchar(20);default:''" json:"country_code"`
	CountryName          string                 `gorm:"type:varchar(100);default:''" json:"country_name"`
	ProductColorFamily           string `gorm:"type:varchar(100);default:''" json:"product_color_family"`
	ProductColorFamilyImageURL   string `gorm:"type:varchar(500);default:''" json:"product_color_family_image_url"`
	AspectRatio          string                 `gorm:"type:varchar(20);default:'1:1'" json:"aspect_ratio"`
	ModelName            string                 `gorm:"type:varchar(100);default:''" json:"model_name"`
	ImageSize            string                 `gorm:"type:varchar(20);default:''" json:"image_size"`
	ImageCount           int                    `gorm:"default:1" json:"image_count"`
	SubjectRatio         string                 `gorm:"type:varchar(20);default:''" json:"subject_ratio"`
	CompositionTemplate  string                 `gorm:"type:varchar(50);default:''" json:"composition_template"`
	PrimaryToneHex       string                 `gorm:"type:varchar(20);default:''" json:"primary_tone_hex"`
	FrameScope           string                 `gorm:"type:varchar(20);default:''" json:"frame_scope"`
	LightingMood         string                 `gorm:"type:varchar(50);default:''" json:"lighting_mood"`
	SpaceMood            string                 `gorm:"type:varchar(50);default:''" json:"space_mood"`
	MaterialFocus        string                 `gorm:"type:varchar(50);default:''" json:"material_focus"`
	StylingDensity       string                 `gorm:"type:varchar(50);default:''" json:"styling_density"`
	BrightnessPreference string                 `gorm:"type:varchar(50);default:''" json:"brightness_preference"`
	Prompt               string                 `gorm:"type:text;default:''" json:"prompt"`
	ProductImageURL      string                 `gorm:"type:varchar(500);default:''" json:"product_image_url"`
	Images               []SceneGenerationImage `gorm:"foreignKey:TaskIDRef;references:TaskID" json:"image_urls,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

func (SceneGenerationTask) TableName() string {
	return "scene_generation_tasks"
}

type SceneGenerationImage struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	TaskIDRef             string    `gorm:"type:varchar(64);index;not null" json:"task_id"`
	Sort                  int       `gorm:"default:0" json:"sort"`
	ImageURL              string    `gorm:"type:varchar(500);not null" json:"image_url"`
	CandidateImageURL     string    `gorm:"type:varchar(500);default:''" json:"candidate_image_url"`
	CandidateStatus       string    `gorm:"type:varchar(20);default:''" json:"candidate_status"`
	CandidateErrorMessage string    `gorm:"type:text;default:''" json:"candidate_error_message"`
	CandidatePrompt       string    `gorm:"type:text;default:''" json:"candidate_prompt"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (SceneGenerationImage) TableName() string {
	return "scene_generation_images"
}

type SceneStyleCategory struct {
	ID        uint     `gorm:"primaryKey" json:"id"`
	Name      string   `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Icon      string   `gorm:"type:varchar(50);default:''" json:"icon"`
	Sort      int      `gorm:"default:0" json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SceneStyleCategory) TableName() string {
	return "scene_style_categories"
}
