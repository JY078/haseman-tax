package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// StringSlice 用于 JSON 列与 []string 之间的自动序列化
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("StringSlice: unsupported type %T", value)
	}
	return json.Unmarshal(bytes, s)
}

// EcomSuiteTask 套图任务主表
type EcomSuiteTask struct {
	ID     uint64 `gorm:"primaryKey"               json:"id"`
	TaskID string `gorm:"type:varchar(64);uniqueIndex;not null" json:"task_id"`
	UserID uint64 `gorm:"index;not null"           json:"user_id"`

	ProductID    uint64 `gorm:"default:0" json:"product_id"`
	WorkflowType string `gorm:"type:varchar(30);default:'legacy'" json:"workflow_type"`

	Platform             string      `gorm:"type:varchar(50);default:''"  json:"platform"`
	Language             string      `gorm:"type:varchar(20);default:''"  json:"language"`
	CountryCode          string      `gorm:"type:varchar(20);default:''"  json:"country_code"`
	CountryName          string      `gorm:"type:varchar(100);default:''" json:"country_name"`
	OutputRatio          string      `gorm:"type:varchar(20);default:'1:1'" json:"output_ratio"`
	Brand                string      `gorm:"type:varchar(50);default:''" json:"brand"`
	DifferentiationMode  string      `gorm:"type:varchar(50);default:''" json:"differentiation_mode"`
	WhiteBgImageURL      string      `gorm:"type:varchar(500);default:''" json:"white_bg_image_url"`
	ProductReferenceURLs StringSlice `gorm:"type:jsonb;serializer:json" json:"product_reference_urls"`
	ReferenceGalleryID   uint64      `gorm:"default:0" json:"reference_gallery_id"`
	WaterFlowMode        string      `gorm:"type:varchar(20);default:''" json:"water_flow_mode"`
	FeatureMode          string      `gorm:"type:varchar(20);default:''" json:"feature_mode"`
	OutputTarget         string      `gorm:"type:varchar(20);default:'all'" json:"output_target"`
	IncludeListing       bool        `gorm:"not null;default:false" json:"include_listing"`
	SecondaryCount       int         `gorm:"default:0" json:"secondary_count"`
	DesktopAPlusCount    int         `gorm:"default:0" json:"desktop_a_plus_count"`
	MobileAPlusCount     int         `gorm:"default:0" json:"mobile_a_plus_count"`
	ProductName          string      `gorm:"type:varchar(500);default:''" json:"product_name"`
	ProductCategory      string      `gorm:"type:varchar(200);default:''" json:"product_category"`
	ProductBrand         string      `gorm:"type:varchar(200);default:''" json:"product_brand"`
	ProductFeatures      string      `gorm:"type:text;default:''" json:"product_features"`
	ProductProof         string      `gorm:"type:text;default:''" json:"product_proof"`
	ProductLength        string      `gorm:"type:varchar(64);default:''" json:"product_length"`
	ProductWidth         string      `gorm:"type:varchar(64);default:''" json:"product_width"`
	ProductHeight        string      `gorm:"type:varchar(64);default:''" json:"product_height"`
	ProductDimensionUnit string      `gorm:"type:varchar(20);default:''" json:"product_dimension_unit"`
	ProductWeight        string      `gorm:"type:varchar(64);default:''" json:"product_weight"`
	ProductWeightUnit    string      `gorm:"type:varchar(20);default:''" json:"product_weight_unit"`
	PackageLength        string      `gorm:"type:varchar(64);default:''" json:"package_length"`
	PackageWidth         string      `gorm:"type:varchar(64);default:''" json:"package_width"`
	PackageHeight        string      `gorm:"type:varchar(64);default:''" json:"package_height"`
	PackageDimensionUnit string      `gorm:"type:varchar(20);default:''" json:"package_dimension_unit"`
	PackageWeight        string      `gorm:"type:varchar(64);default:''" json:"package_weight"`
	PackageWeightUnit    string      `gorm:"type:varchar(20);default:''" json:"package_weight_unit"`
	PackageContents      string      `gorm:"type:text;default:''" json:"package_contents"`
	ListingPlatformRules string      `gorm:"type:text;default:''" json:"listing_platform_rules"`
	ListingQualityRules  string      `gorm:"type:text;default:''" json:"listing_quality_rules"`
	ListingKeywords      string      `gorm:"type:text;default:''" json:"listing_keywords"`
	BuyerQuestions       string      `gorm:"type:json;default:'[]'" json:"buyer_questions"`
	VisualStyle          string      `gorm:"type:text;default:''" json:"visual_style"`
	StyleRequirements    string      `gorm:"type:text;default:''" json:"style_requirements"`
	StyleReferenceURLs   StringSlice `gorm:"type:jsonb;serializer:json" json:"style_reference_urls"`
	DrawingModel         string      `gorm:"type:varchar(255);default:''" json:"drawing_model"`

	// 前端只能选 1 个 A+ 模板；nil 表示走随机分配
	SelectedAPlusTemplateID *uint64 `gorm:"default:null" json:"selected_a_plus_template_id"`
	UseBrandLogo            bool    `gorm:"not null;default:true" json:"use_brand_logo"`

	Status        string `gorm:"type:varchar(30);default:'pending'" json:"status"`
	SmartStage    string `gorm:"type:varchar(40);default:''"        json:"smart_stage"`
	CurrentAction string `gorm:"type:text;default:''"                json:"current_action"`
	ErrorMessage  string `gorm:"type:text;default:''"               json:"error_message"`

	// 大模型一次性规划结果快照（JSON 列）
	PlanSnapshot *EcomSuitePlanSnapshot `gorm:"type:json;serializer:json" json:"plan_snapshot,omitempty"`

	Images []EcomSuiteImage `gorm:"foreignKey:TaskID;references:TaskID" json:"images,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EcomSuiteTask) TableName() string { return "ecom_suite_tasks" }

// EcomSuiteImage 套图图片明细表
type EcomSuiteImage struct {
	ID     uint64 `gorm:"primaryKey" json:"id"`
	TaskID string `gorm:"type:varchar(64);index;not null" json:"task_id"`

	ImageType   string `gorm:"type:varchar(30);default:''"  json:"image_type"`
	ImageIndex  int    `gorm:"default:1"                    json:"image_index"`
	OutputRatio string `gorm:"type:varchar(20);default:''" json:"output_ratio"`
	Role        string `gorm:"type:text;default:''" json:"role"`

	Title                 string      `gorm:"type:varchar(500);default:''" json:"title"`
	Subtitle              string      `gorm:"type:varchar(500);default:''" json:"subtitle"`
	MustIncludeText       string      `gorm:"type:text;default:''"         json:"must_include_text"`
	LayoutInstruction     string      `gorm:"type:text;default:''"         json:"layout_instruction"`
	TypographyInstruction string      `gorm:"type:text;default:''"         json:"typography_instruction"`
	VisualStrategy        string      `gorm:"type:text;default:''"         json:"visual_strategy"`
	ReferenceRule         StringSlice `gorm:"type:json;serializer:json"    json:"reference_rule"`
	// 生图 prompt（大模型直接输出的完整提示词，不描述产品外观）
	PromptDraft string `gorm:"type:text;default:''"         json:"prompt"`
	PromptFinal string `gorm:"type:text;default:''"         json:"prompt_final"`
	// 颜色字段（从 global_style 基础上有具体变化时记录）
	PrimaryColor   string `gorm:"type:varchar(200);default:''" json:"primary_color"`
	SecondaryColor string `gorm:"type:varchar(200);default:''" json:"secondary_color"`

	// 仅 A+ 图有值
	APlusTemplateID *uint64 `gorm:"default:null" json:"a_plus_template_id"`
	IncludeLogo     bool    `gorm:"not null;default:false" json:"include_logo"`

	// 副图目的 / A+风格（从规划阶段继承）
	CTADirection string `gorm:"type:text;default:''" json:"cta_direction"`
	Style        string `gorm:"type:text;default:''" json:"style"`

	Status                string `gorm:"type:varchar(30);default:'pending'" json:"status"`
	ImageURL              string `gorm:"type:varchar(1000);default:''"      json:"image_url"`
	ReviewSourceImageURL  string `gorm:"type:varchar(1000);default:''"      json:"review_source_image_url,omitempty"`
	ErrorMessage          string `gorm:"type:text;default:''"               json:"error_message"`
	CandidateImageURL     string `gorm:"type:varchar(1000);default:''" json:"candidate_image_url,omitempty"`
	CandidateStatus       string `gorm:"type:varchar(30);default:''" json:"candidate_status,omitempty"`
	CandidatePrompt       string `gorm:"type:text;default:''" json:"candidate_prompt,omitempty"`
	CandidateErrorMessage string `gorm:"type:text;default:''" json:"candidate_error_message,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EcomSuiteImage) TableName() string { return "ecom_suite_images" }

type SmartSuiteConversation struct {
	ID        uint64      `gorm:"primaryKey" json:"id"`
	SessionID string      `gorm:"type:varchar(64);uniqueIndex;not null" json:"session_id"`
	UserID    uint64      `gorm:"index;not null" json:"user_id"`
	TaskID    string      `gorm:"type:varchar(64);default:''" json:"task_id"`
	TaskIDs   StringSlice `gorm:"type:jsonb;serializer:json" json:"task_ids"`

	Context  SmartSuiteConversationContext   `gorm:"type:jsonb;serializer:json" json:"context"`
	Messages []SmartSuiteConversationMessage `gorm:"type:jsonb;serializer:json" json:"messages"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SmartSuiteConversation) TableName() string { return "smart_suite_conversations" }

type SmartSuiteConversationContext struct {
	WhiteBgImageURL      string                 `json:"white_bg_image_url"`
	ProductReferenceURLs StringSlice            `json:"product_reference_urls"`
	ProductName          string                 `json:"product_name"`
	ProductCategory      string                 `json:"product_category"`
	ProductBrand         string                 `json:"product_brand"`
	ProductFeatures      string                 `json:"product_features"`
	ProductProof         string                 `json:"product_proof"`
	ProductLength        string                 `json:"product_length"`
	ProductWidth         string                 `json:"product_width"`
	ProductHeight        string                 `json:"product_height"`
	ProductDimensionUnit string                 `json:"product_dimension_unit"`
	ProductWeight        string                 `json:"product_weight"`
	ProductWeightUnit    string                 `json:"product_weight_unit"`
	PackageLength        string                 `json:"package_length"`
	PackageWidth         string                 `json:"package_width"`
	PackageHeight        string                 `json:"package_height"`
	PackageDimensionUnit string                 `json:"package_dimension_unit"`
	PackageWeight        string                 `json:"package_weight"`
	PackageWeightUnit    string                 `json:"package_weight_unit"`
	PackageContents      string                 `json:"package_contents"`
	ListingPlatformRules string                 `json:"listing_platform_rules"`
	ListingQualityRules  string                 `json:"listing_quality_rules"`
	ListingKeywords      string                 `json:"listing_keywords"`
	Platform             string                 `json:"platform"`
	PlatformName         string                 `json:"platform_name"`
	CountryCode          string                 `json:"country_code"`
	CountryName          string                 `json:"country_name"`
	Language             string                 `json:"language"`
	LanguageName         string                 `json:"language_name"`
	OutputTarget         string                 `json:"output_target"`
	RequestType          string                 `json:"request_type"`
	IncludeListing       bool                   `json:"include_listing"`
	RequirementFormOpen  bool                   `json:"requirement_form_open"`
	SecondaryCount       int                    `json:"secondary_count"`
	DesktopAPlusCount    int                    `json:"desktop_a_plus_count"`
	MobileAPlusCount     int                    `json:"mobile_a_plus_count"`
	VisualStyle          string                 `json:"visual_style"`
	StyleRequirements    string                 `json:"style_requirements"`
	StyleReferenceURLs   StringSlice            `json:"style_reference_urls"`
	DrawingModel         string                 `json:"drawing_model"`
	ActiveImageID        uint64                 `json:"active_image_id"`
	ListingEvaluation    map[string]interface{} `json:"listing_evaluation,omitempty"`
}

type SmartSuiteConversationMessage struct {
	ID            string   `json:"id"`
	Role          string   `json:"role"`
	Text          string   `json:"text,omitempty"`
	ImageURL      string   `json:"image_url,omitempty"`
	Kind          string   `json:"kind,omitempty"`
	PublicSummary string   `json:"public_summary,omitempty"`
	PublicActions []string `json:"public_actions,omitempty"`
	At            string   `json:"at"`
}

// ---------- 大模型规划结果快照结构 ----------

type EcomSuitePlanSnapshot struct {
	CreativeReview       *EcomSuiteCreativeReview     `json:"creative_review,omitempty"`
	GlobalStyle          EcomSuiteGlobalStyle         `json:"global_style"`
	StyleMaster          EcomSuiteStyleMaster         `json:"style_master,omitempty"`
	ProductFactMap       EcomSuiteProductFactMap      `json:"product_fact_map,omitempty"`
	BuyerConcerns        []EcomSuiteBuyerConcern      `json:"buyer_concerns,omitempty"`
	Listing              *EcomSuiteListing            `json:"listing,omitempty"`
	ListingError         string                       `json:"listing_error,omitempty"`
	ListingReview        *EcomSuiteListingReview      `json:"listing_review,omitempty"`
	ListingReviewError   string                       `json:"listing_review_error,omitempty"`
	ListingKeywords      []EcomSuiteKeywordCandidate  `json:"listing_keywords,omitempty"`
	ListingKeywordsReady bool                         `json:"listing_keywords_ready,omitempty"`
	ImagePlans           []EcomSuiteImagePlan         `json:"image_plans"`
	CreativeDirections   []EcomSuiteCreativeDirection `json:"creative_directions,omitempty"`
	Copywriting          []EcomSuiteCopywriting       `json:"copywriting,omitempty"`
	AuditChecks          []EcomSuiteAuditCheck        `json:"audit_checks,omitempty"`
	PlannedImageTypes    []string                     `json:"planned_image_types,omitempty"`
	// 阶段一：内容规划（每个图片的规划思路）
	ContentPlannings []EcomSuiteContentPlanning `json:"content_plannings"`
	// 阶段二：生图指令（每个图片的具体生成指令）
	GenerationInstructions []EcomSuiteGenerationInstruction `json:"generation_instructions"`
	AgentRuns              []EcomSuiteAgentRun              `json:"agent_runs,omitempty"`
}

// Creative review evaluates plans before any paid image generation.
type EcomSuiteCreativeReview struct {
	Suggestions      []EcomSuiteCreativeIssue `json:"suggestions,omitempty"`
	InputFingerprint string                   `json:"input_fingerprint"`
	Passed           bool                     `json:"passed"`
	Summary          string                   `json:"summary"`
	Checks           map[string]string        `json:"checks"`
	Issues           []EcomSuiteCreativeIssue `json:"issues"`
	RepairAttempts   int                      `json:"repair_attempts"`
}

type EcomSuiteCreativeIssue struct {
	Severity   string `json:"severity,omitempty"`
	ImageType  string `json:"image_type"`
	ImageIndex int    `json:"image_index"`
	Category   string `json:"category"`
	Problem    string `json:"problem"`
	Correction string `json:"correction"`
}

type EcomSuiteAgentRun struct {
	Turn            int       `json:"turn"`
	Agent           string    `json:"agent"`
	Reason          string    `json:"reason"`
	AvailableAgents []string  `json:"available_agents,omitempty"`
	Status          string    `json:"status"`
	Error           string    `json:"error,omitempty"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at,omitempty"`
}

// EcomSuiteListing is the synchronized marketplace copy produced from the
// same confirmed facts used by the image-suite agents.
type EcomSuiteListing struct {
	Platform                string                      `json:"platform,omitempty"`
	Title                   string                      `json:"title,omitempty"`
	Description             string                      `json:"description,omitempty"`
	Bullets                 []string                    `json:"bullets,omitempty"`
	SalientBullets          []string                    `json:"salient_bullets,omitempty"`
	BulletPoints            []string                    `json:"bullet_points,omitempty"`
	AboutThisProduct        string                      `json:"about_this_product,omitempty"`
	Highlights              []string                    `json:"highlights,omitempty"`
	SupplierBullets         []string                    `json:"supplier_bullets,omitempty"`
	MarketingCopy           string                      `json:"marketing_copy,omitempty"`
	LowesProductDescription string                      `json:"lowes_product_description,omitempty"`
	Features                []string                    `json:"features,omitempty"`
	Variants                []string                    `json:"variants,omitempty"`
	WhatsIncluded           []string                    `json:"whats_included,omitempty"`
	NotIncluded             []string                    `json:"not_included,omitempty"`
	Documents               []string                    `json:"documents,omitempty"`
	Attributes              []EcomSuiteListingAttribute `json:"attributes,omitempty"`
	Details                 []EcomSuiteListingAttribute `json:"details,omitempty"`
	SearchTerms             []string                    `json:"search_terms,omitempty"`
	Evidence                []EcomSuiteListingEvidence  `json:"evidence,omitempty"`
	OptimizationReasons     []EcomSuiteListingReason    `json:"optimization_reasons,omitempty"`
	KeywordCandidates       []EcomSuiteKeywordCandidate `json:"-"`
}

type EcomSuiteListingAttribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// EcomSuiteListingEvidence links each generated listing field back to the
// confirmed product facts that justify it. Fact IDs are validated server-side
// before a listing is persisted.
type EcomSuiteListingEvidence struct {
	Field   string   `json:"field"`
	FactIDs []string `json:"fact_ids,omitempty"`
	Source  string   `json:"source,omitempty"`
	Quote   string   `json:"quote,omitempty"`
}

type EcomSuiteListingReason struct {
	Field           string   `json:"field"`
	Reason          string   `json:"reason"`
	Rule            string   `json:"rule,omitempty"`
	PlatformBasis   string   `json:"platform_basis,omitempty"`
	WritingDecision string   `json:"writing_decision,omitempty"`
	ExpectedImpact  string   `json:"expected_impact,omitempty"`
	FactIDs         []string `json:"fact_ids,omitempty"`
}

// EcomSuiteListingReview is the independent platform compliance audit result.
type EcomSuiteListingReview struct {
	Platform         string   `json:"platform"`
	InputFingerprint string   `json:"input_fingerprint,omitempty"`
	ReviewedTitle    string   `json:"reviewed_title,omitempty"`
	Passed           bool     `json:"passed"`
	Issues           []string `json:"issues,omitempty"`
	Checks           []string `json:"checks,omitempty"`
	Summary          string   `json:"summary,omitempty"`
}

type EcomSuiteKeywordCandidate struct {
	Term              string   `json:"term"`
	Source            string   `json:"source,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	SearchVolume      int      `json:"search_volume,omitempty"`
	Competition       float64  `json:"competition,omitempty"`
	PlatformFrequency int      `json:"platform_frequency,omitempty"`
	SEOScore          int      `json:"seo_score,omitempty"`
	FactIDs           []string `json:"fact_ids,omitempty"`
	EvidenceURLs      []string `json:"evidence_urls,omitempty"`
}

type EcomSuiteProductFact struct {
	ID             string `json:"id"`
	Statement      string `json:"statement"`
	EvidenceSource string `json:"evidence_source"`
	Evidence       string `json:"evidence"`
}

type EcomSuiteProductInference struct {
	ID         string  `json:"id"`
	Statement  string  `json:"statement"`
	Basis      string  `json:"basis"`
	Confidence float64 `json:"confidence"`
	AllowedUse string  `json:"allowed_use"`
}

type EcomSuiteProductFactMap struct {
	ConfirmedFacts []EcomSuiteProductFact      `json:"confirmed_facts"`
	Inferences     []EcomSuiteProductInference `json:"reasonable_inferences"`
	// PlacementOptions are candidate install or staging sites inferred by the
	// product-fact agent from the product image and document. The backend never
	// derives them itself; it only carries them to downstream agents.
	PlacementOptions []EcomSuitePlacementOption `json:"placement_options,omitempty"`
}

// EcomSuitePlacementOption is one candidate place where the product may be
// installed or staged. Site and mount are free-form agent wording.
// ClaimSafe is true only when a user document confirms the mounting statement,
// so only then may a placement appear as a visible claim.
type EcomSuitePlacementOption struct {
	ID         string  `json:"id"`
	Site       string  `json:"site"`
	Mount      string  `json:"mount"`
	Source     string  `json:"source"`
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
	ClaimSafe  bool    `json:"claim_safe"`
}

type EcomSuiteBuyerConcern struct {
	ID                    string   `json:"id"`
	Priority              int      `json:"priority"`
	Question              string   `json:"question"`
	Reason                string   `json:"reason"`
	EvidenceFactIDs       []string `json:"evidence_fact_ids"`
	InferenceIDs          []string `json:"inference_ids,omitempty"`
	ProofMethod           string   `json:"proof_method"`
	RecommendedImageTypes []string `json:"recommended_image_types"`
}

type EcomSuiteStyleEvidence struct {
	Source string `json:"source"`
	Detail string `json:"detail"`
}

type EcomSuiteStyleMaster struct {
	GlobalStyle EcomSuiteGlobalStyle     `json:"global_style"`
	Evidence    []EcomSuiteStyleEvidence `json:"evidence"`
}

type EcomSuiteCreativeDirection struct {
	ImageType            string   `json:"image_type"`
	ImageIndex           int      `json:"image_index"`
	BuyerConcernIDs      []string `json:"buyer_concern_ids,omitempty"`
	BuyerQuestion        string   `json:"buyer_question"`
	PrimarySellingPoint  string   `json:"primary_selling_point"`
	AssignedFactIDs      []string `json:"assigned_fact_ids,omitempty"`
	AssignedProductFacts string   `json:"assigned_product_facts"`
	VisualEvidence       string   `json:"visual_evidence"`
	ShotScale            string   `json:"shot_scale"`
	SceneDirection       string   `json:"scene_direction"`
	// Placement is the concrete install or staging site chosen by the creative
	// director for this one image. PlacementOptionID links it back to the
	// product-fact agent's placement_options when one was used.
	Placement         string `json:"placement,omitempty"`
	PlacementOptionID string `json:"placement_option_id,omitempty"`
}

// EcomSuiteCopywriting is the immutable copy output for one suite image.
// Visual planning and image generation may consume these fields, but must not
// rewrite them.
type EcomSuiteCopywriting struct {
	ImageType       string   `json:"image_type"`
	ImageIndex      int      `json:"image_index"`
	Title           string   `json:"title"`
	Subtitle        string   `json:"subtitle"`
	MustIncludeText string   `json:"must_include_text"`
	EvidenceFactIDs []string `json:"evidence_fact_ids,omitempty"`
}

type EcomSuiteAuditCheck struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// 内容规划阶段输出（纯文本格式）
type EcomSuiteContentPlanning struct {
	ImageType       string `json:"image_type"`
	ImageIndex      int    `json:"image_index"`
	ContentType     string `json:"content_type,omitempty"`
	IncludeLogo     bool   `json:"include_logo"`
	CTADirection    string `json:"cta_direction"`
	PhysicalPoints  string `json:"physical_points"`
	UserValue       string `json:"user_value"`
	Emotional       string `json:"emotional"`
	SceneSuggestion string `json:"scene_suggestion"`
}

// 生图指令阶段输出（JSON格式）
type EcomSuiteGenerationInstruction struct {
	ImageType             string   `json:"image_type"`
	ImageIndex            int      `json:"image_index"`
	Role                  string   `json:"role"`
	ContentType           string   `json:"content_type,omitempty"`
	Title                 string   `json:"title"`
	Subtitle              string   `json:"subtitle"`
	MustIncludeText       string   `json:"must_include_text"`
	Style                 string   `json:"style"`
	Prompt                string   `json:"prompt"`
	LayoutInstruction     string   `json:"layout_instruction"`
	TypographyInstruction string   `json:"typography_instruction"`
	VisualStrategy        string   `json:"visual_strategy"`
	PrimaryColor          string   `json:"primary_color"`
	SecondaryColor        string   `json:"secondary_color"`
	IncludeLogo           bool     `json:"include_logo"`
	ReferenceRule         []string `json:"reference_rule"`

	// 兼容旧字段
	Goal         string `json:"goal,omitempty"`
	RequiredCopy string `json:"required_copy,omitempty"`
	CTADirection string `json:"cta_direction,omitempty"`
}

type EcomSuiteGlobalStyle struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	FontColor      string `json:"font_color"`
	BackgroundTone string `json:"background_tone"`
	Lighting       string `json:"lighting"`
	ColorSystem    string `json:"color_system"`
	Staging        string `json:"staging"`
	FontSystem     string `json:"font_system"`
	LabelSystem    string `json:"label_system"`
	TextureQuality string `json:"texture_quality"`
}

type EcomSuiteLayoutSpec struct {
	ProductOccupancyPercent int                   `json:"product_occupancy_percent"`
	ProductAnchor           string                `json:"product_anchor"`
	TextSafeArea            EcomSuiteLayoutRegion `json:"text_safe_area"`
	ComparisonMode          string                `json:"comparison_mode"`
	PanelCount              int                   `json:"panel_count,omitempty"`
	CameraAngle             string                `json:"camera_angle"`
	ForbiddenLayouts        []string              `json:"forbidden_layouts"`
}

type EcomSuiteLayoutRegion struct {
	XPercent      int `json:"x_percent"`
	YPercent      int `json:"y_percent"`
	WidthPercent  int `json:"width_percent"`
	HeightPercent int `json:"height_percent"`
}

type EcomSuiteImagePlan struct {
	ImageType            string   `json:"image_type"`
	ImageIndex           int      `json:"image_index"`
	Role                 string   `json:"role"`
	AssignedFactIDs      []string `json:"assigned_fact_ids,omitempty"`
	AssignedProductFacts string   `json:"assigned_product_facts,omitempty"`
	ShotScale            string   `json:"shot_scale,omitempty"`
	ContentType          string   `json:"content_type,omitempty"` // A+图内容类型，如 brand_opening, problem_insight 等
	IncludeLogo          bool     `json:"include_logo"`           // 是否将 logo 作为参考图

	// 副图专用：目的
	CTADirection string `json:"cta_direction"` // CTA方向：这张图要达成什么目标

	// 文案：标题+副标题（这是要渲染在图片上的实际文字）
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`

	// 画面描述
	Prompt string `json:"prompt"` // 提示词：完整的英文生图描述

	// 文案
	MustIncludeText string `json:"must_include_text"` // 必须的文案：title和subtitle，格式"标题|副文案"

	// 风格
	Style string `json:"style"` // 视觉风格描述

	// 其他字段（副图可能需要）
	LayoutInstruction     string              `json:"layout_instruction"`
	LayoutSpec            EcomSuiteLayoutSpec `json:"layout_spec,omitempty"`
	TypographyInstruction string              `json:"typography_instruction"`
	VisualStrategy        string              `json:"visual_strategy"`
	ReferenceRule         []string            `json:"reference_rule"`
	PrimaryColor          string              `json:"primary_color,omitempty"`
	SecondaryColor        string              `json:"secondary_color,omitempty"`
	// Placement carries the creative director's assigned install or staging
	// site through to the generation contract. It is agent-authored text.
	Placement         string `json:"placement,omitempty"`
	PlacementOptionID string `json:"placement_option_id,omitempty"`

	// 旧字段兼容（计划删除）
	PhysicalPoints  string `json:"physical_points,omitempty"`  // 旧：物理卖点
	UserValue       string `json:"user_value,omitempty"`       // 旧：用户价值
	Emotional       string `json:"emotional,omitempty"`        // 旧：情绪触发
	SceneSuggestion string `json:"scene_suggestion,omitempty"` // 旧：画面建议
	Goal            string `json:"goal,omitempty"`             // 旧：目标
	RequiredCopy    string `json:"required_copy,omitempty"`    // 旧：必须的文案
}
