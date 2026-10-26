package router

import (
	"net/http"

	"advertising-image/backend/internal/handler"
	"advertising-image/backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

// maxRequestBodyBytes limits request body size to 100 MB.
const maxRequestBodyBytes = 100 * 1024 * 1024

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Reject oversized request bodies early to protect memory and bandwidth.
	// Only applies when a body is present (POST/PUT/PATCH); GET/HEAD have nil bodies.
	r.Use(func(c *gin.Context) {
		if c.Request.Body != nil && c.Request.Body != http.NoBody {
			if c.Request.ContentLength > maxRequestBodyBytes {
				c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
					"error": "请求体过大，最大支持 100 MB",
				})
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
		}
		c.Next()
	})

	// CORS middleware
	r.Use(middleware.CORSMiddleware())

	// Serve uploaded images as static files (public, no auth required)
	r.Static("/uploads", "./uploads")

	// API Group
	api := r.Group("/api")
	{
		// Internal routes (service-to-service, no user auth required)
		api.POST("/internal/image/generate", handler.GenerateImageInternal)
		api.POST("/internal/chat", handler.ChatInternal)
		api.POST("/internal/stp-render/complete", handler.UpdateRenderResult) // Python Worker 调用
		api.POST("/internal/stp-render/single/complete", handler.UpdateStpSingleRenderResult)
		api.POST("/internal/tax-data/raw/upload", handler.UploadTaxDataFromRPA)
		api.POST("/internal/tax-data/tiktok/upload", handler.UploadTikTokTaxData)
		api.POST("/internal/tax-data/tiktok/ads/upload", handler.UploadTikTokTaxAds)
		api.POST("/internal/tax-data/tiktok/fbt/upload", handler.UploadTikTokTaxFBT)
		api.POST("/internal/tax-data/tiktok/settlements/upload", handler.UploadTikTokSettlements)
		api.POST("/internal/tax-data/tiktok/orders/upload", handler.UploadTikTokOrders)
		api.POST("/internal/tax-data/amazon/shipments/upload", handler.UploadAmazonTaxShipments)
		api.POST("/internal/tax-data/amazon/finance/upload", handler.UploadAmazonTaxFinance)

		// Auth
		api.POST("/login", handler.Login)
		api.POST("/logout", handler.Logout)
		api.POST("/register", handler.Register)
		api.POST("/send-verification-code", handler.SendVerificationCode)
		api.GET("/email-register-enabled", handler.CheckEmailRegisterEnabled)
		api.GET("/maintenance-status", handler.CheckMaintenanceMode)
		api.GET("/brand-settings", handler.GetBrandSettings)
		api.POST("/forgot-password", handler.ForgotPassword)
		api.POST("/reset-password", handler.ResetPassword)

		// Public: Prompt read-only (模板库无需登录)
		api.GET("/prompt-groups", handler.GetPromptGroups)
		api.GET("/prompts", handler.GetPrompts)
		api.GET("/product-library", handler.GetProductLibrary)

		// Public: Image Proxy（前端 <img> 标签展示 OSS 图片时需要，避免 CORS）
		api.GET("/proxy", handler.ProxyImage)

		// Protected Routes
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			protected.GET("/buyer-reviews", handler.ListBuyerReviews)
			protected.POST("/buyer-reviews/import", handler.ImportBuyerReviews)
			protected.POST("/buyer-reviews/import-workbook", handler.ImportBuyerReviewWorkbook)
			protected.PUT("/buyer-reviews/:id/video-link", handler.UpdateBuyerReviewVideoLink)
			protected.POST("/buyer-reviews/:id/video", handler.UploadBuyerReviewVideo)
			protected.GET("/image-violation-examples", handler.ListImageViolationExamples)
			protected.POST("/image-violation-examples", handler.CreateImageViolationExample)
			protected.GET("/tax-data", handler.RequireTaxDataAccess(), handler.ListTaxData)
			protected.GET("/tax-data/catalog", handler.RequireTaxDataAccess(), handler.ListTaxCatalog)
			protected.POST("/tax-data/catalog", handler.RequireTaxDataAccess(), handler.CreateTaxCatalogEntry)
			protected.GET("/finance/customs-declarations", handler.RequireTaxDataAccess(), handler.ListCustomsDeclarations)
			protected.GET("/finance/customs-declarations/summary", handler.RequireTaxDataAccess(), handler.SummarizeCustomsDeclarations)
			protected.GET("/finance/customs-declarations/export", handler.RequireTaxDataAccess(), handler.ExportCustomsDeclarations)
			protected.POST("/finance/customs-declarations", handler.RequireTaxDataAccess(), handler.UploadCustomsDeclaration)
			protected.GET("/finance/customs-declarations/:id", handler.RequireTaxDataAccess(), handler.GetCustomsDeclaration)
			protected.GET("/finance/customs-declarations/:id/file", handler.RequireTaxDataAccess(), handler.DownloadCustomsDeclaration)
			protected.GET("/tax-data/rules", handler.RequireTaxDataAccess(), handler.ListTaxRules)
			protected.POST("/tax-data/rules", handler.RequireTaxDataAccess(), handler.CreateTaxRule)
			protected.GET("/tax-data/quarters", handler.RequireTaxDataAccess(), handler.ListTaxQuarters)
			protected.GET("/finance/exchange-rates", handler.RequireExchangeRateAccess(), handler.ListTaxExchangeRates)
			protected.POST("/finance/exchange-rates/refresh", handler.RequireExchangeRateAccess(), handler.RequireExchangeRateRefresh(), handler.RefreshTaxExchangeRates)
			protected.PUT("/tax-data/months/:month/rate", handler.RequireTaxDataAccess(), handler.UpdateTaxMonthRate)
			protected.GET("/tax-data/raw", handler.RequireTaxDataAccess(), handler.ListTaxRaw)
			protected.GET("/tax-data/raw/filter-options", handler.RequireTaxDataAccess(), handler.ListTaxRawFilterOptions)
			protected.GET("/tax-data/imports", handler.RequireTaxDataAccess(), handler.ListTaxImports)
			protected.GET("/tax-data/imports/:id/file", handler.RequireTaxDataAccess(), handler.DownloadTaxImport)
			protected.GET("/tax-data/downloads", handler.RequireTaxDataAccess(), handler.ListTaxDownloads)
			protected.GET("/tax-data/downloads/:id/file", handler.RequireTaxDataAccess(), handler.DownloadTaxHistoryFile)
			protected.GET("/tax-data/export", handler.RequireTaxDataAccess(), handler.ExportTaxReport)
			protected.GET("/tax-data/batches/:id/raw", handler.RequireTaxDataAccess(), handler.ListTaxRaw)
			protected.GET("/tax-data/batches/:id/processed", handler.RequireTaxDataAccess(), handler.ListTaxProcessed)
			protected.GET("/tax-data/tiktok/details", handler.RequireTaxDataAccess(), handler.ListTikTokTaxDetails)
			protected.GET("/tax-data/tiktok/quarters", handler.RequireTaxDataAccess(), handler.ListTikTokTaxQuarters)
			protected.GET("/tax-data/tiktok/ads", handler.RequireTaxDataAccess(), handler.ListTikTokTaxAds)
			protected.GET("/tax-data/tiktok/fbt", handler.RequireTaxDataAccess(), handler.ListTikTokTaxFBT)
			protected.GET("/tax-data/tiktok/settlements", handler.RequireTaxDataAccess(), handler.ListTikTokSettlements)
			protected.GET("/tax-data/tiktok/orders", handler.RequireTaxDataAccess(), handler.ListTikTokOrders)
			protected.POST("/tax-data/tiktok/details/upload", handler.RequireTaxDataAccess(), handler.UploadTikTokTaxData)
			protected.POST("/tax-data/tiktok/ads/upload", handler.RequireTaxDataAccess(), handler.UploadTikTokTaxAds)
			protected.POST("/tax-data/tiktok/fbt/upload", handler.RequireTaxDataAccess(), handler.UploadTikTokTaxFBT)
			protected.POST("/tax-data/tiktok/settlements/upload", handler.RequireTaxDataAccess(), handler.UploadTikTokSettlements)
			protected.POST("/tax-data/tiktok/orders/upload", handler.RequireTaxDataAccess(), handler.UploadTikTokOrders)
			protected.GET("/tax-data/amazon/filter-options", handler.RequireTaxDataAccess(), handler.ListAmazonTaxFilterOptions)
			protected.GET("/tax-data/amazon/product-categories", handler.RequireTaxDataAccess(), handler.ListAmazonProductCategories)
			protected.POST("/tax-data/amazon/product-categories", handler.RequireTaxDataAccess(), handler.SaveAmazonProductCategory)
			protected.PUT("/tax-data/amazon/product-categories/:id", handler.RequireTaxDataAccess(), handler.SaveAmazonProductCategory)
			protected.DELETE("/tax-data/amazon/product-categories/:id", handler.RequireTaxDataAccess(), handler.DeleteAmazonProductCategory)
			protected.GET("/tax-data/amazon/:kind", handler.RequireTaxDataAccess(), handler.ListAmazonTaxRows)
			protected.GET("/tax-data/amazon/export", handler.RequireTaxDataAccess(), handler.ExportAmazonTaxReport)
			protected.POST("/tax-data/amazon/shipments/upload", handler.RequireTaxDataAccess(), handler.UploadAmazonTaxShipments)
			protected.POST("/tax-data/amazon/finance/upload", handler.RequireTaxDataAccess(), handler.UploadAmazonTaxFinance)
			protected.POST("/tax-data/raw/upload", handler.RequireTaxDataAccess(), handler.UploadTaxDataFromRPA)
			protected.GET("/tax-data/tiktok/filter-options", handler.RequireTaxDataAccess(), handler.ListTikTokTaxFilterOptions)
			protected.GET("/tax-data/tiktok/export", handler.RequireTaxDataAccess(), handler.ExportTikTokTaxReport)
			protected.GET("/me", handler.GetProfile)
			protected.PUT("/me/password", handler.ChangePassword)
			// Heartbeat: AuthMiddleware validates JWT + session existence, returns 401 if session expired/kicked
			protected.POST("/heartbeat", handler.Heartbeat)
			protected.GET("/me/sessions", handler.GetMySessions)
			protected.DELETE("/me/sessions/:id", handler.RevokeSession)
			protected.POST("/me/send-bind-email-code", handler.SendBindEmailCode)
			protected.PUT("/me/email", handler.BindEmail)
			protected.GET("/menus", handler.GetMenus)
			protected.GET("/daily-calls", handler.ListDailyCalls)
			protected.POST("/daily-calls", handler.CreateDailyCall)
			protected.GET("/daily-calls/:id", handler.GetDailyCall)
			protected.PUT("/daily-calls/:id/registration", handler.UpdateDailyCallRegistration)
			protected.PUT("/daily-calls/:id/review", handler.UpdateDailyCallReview)
			// Work requests (all authenticated users can create and view;
			// handlers enforce creator, assignee and administrator actions).
			protected.GET("/work-requests", handler.ListWorkRequests)
			protected.POST("/work-requests", handler.CreateWorkRequest)
			protected.GET("/work-requests/users", handler.ListWorkRequestUsers)
			protected.POST("/work-requests/reference-images", handler.UploadWorkRequestReference)
			protected.GET("/work-requests/:id", handler.GetWorkRequest)
			protected.PUT("/work-requests/:id", handler.UpdateWorkRequest)
			protected.POST("/work-requests/:id/assign", handler.AssignWorkRequest)
			protected.POST("/work-requests/:id/return", handler.ReturnWorkRequestToPending)
			protected.POST("/work-requests/:id/progress", handler.UpdateWorkRequestProgress)
			protected.POST("/work-requests/:id/submit-review", handler.SubmitWorkRequestReview)
			protected.POST("/work-requests/:id/approve", handler.ApproveWorkRequest)
			protected.POST("/work-requests/:id/reject", handler.RejectWorkRequest)
			protected.POST("/work-requests/:id/cancel", handler.CancelWorkRequest)
			protected.DELETE("/work-requests/:id", handler.DeleteWorkRequest)
			protected.GET("/purchase-plans", handler.ListPurchasePlans)
			protected.POST("/purchase-plans", handler.CreatePurchasePlan)
			protected.GET("/purchase-plans/products", handler.SearchPurchasePlanProducts)
			protected.GET("/download-options", handler.GetDownloadOptions)
			protected.GET("/product-spus", handler.SearchProductSPUs)
			protected.GET("/product-skus", handler.SearchProductSKUs)
			protected.GET("/product-prices", handler.ListProductPrices)
			protected.GET("/product-prices/history", handler.ListProductPriceHistory)
			protected.GET("/product-prices/amazon-fetch-targets", handler.ListAmazonPriceFetchTargets)
			protected.POST("/product-prices/amazon/:id/fetch", handler.FetchAmazonProductPrice)
			// Kept out of the /amazon/:id/ prefix: gin rejects a static segment
			// where a wildcard is already registered.
			protected.POST("/product-prices/fetch-batch", handler.StartAmazonPriceBatchFetch)
			protected.GET("/product-prices/fetch-batch/:jobId", handler.GetAmazonPriceBatchFetch)
			protected.POST("/product-prices/fetch-batch/:jobId/cancel", handler.CancelAmazonPriceBatchFetch)
			protected.PUT("/product-prices/:platform/:id", handler.UpdateProductPrice)
			protected.GET("/product-catalog/spus", handler.AdminListProductSPUs)
			protected.POST("/product-catalog/spus", handler.AdminCreateProductSPU)
			protected.PUT("/product-catalog/spus/:id", handler.AdminUpdateProductSPU)
			protected.DELETE("/product-catalog/spus/:id", middleware.RequireAdmin(), handler.AdminDeleteProductSPU)
			protected.POST("/product-catalog/spus/batch-delete", middleware.RequireAdmin(), handler.AdminBatchDeleteProductSPUs)
			protected.POST("/product-catalog/spus/:id/model", handler.AdminUploadProductSPUModel)
			protected.POST("/product-catalog/spus/:id/attachments", handler.AdminUploadProductSPUAttachment)
			protected.PUT("/product-catalog/spus/:id/attachments/:attachment_id", handler.AdminUpdateProductSPUAttachment)
			protected.DELETE("/product-catalog/spus/:id/attachments/:attachment_id", middleware.RequireAdmin(), handler.AdminDeleteProductSPUAttachment)
			protected.POST("/product-catalog/skus", handler.AdminCreateProductSKU)
			protected.PUT("/product-catalog/skus/:id", handler.AdminUpdateProductSKU)
			protected.PUT("/product-catalog/skus/:id/platform-skus", handler.UpdateProductSKUPlatformSKUs)
			protected.DELETE("/product-catalog/skus/:id", middleware.RequireAdmin(), handler.AdminDeleteProductSKU)
			protected.POST("/product-catalog/import", handler.ImportProductCatalog)
			protected.POST("/order-sku-mappings/resolve", handler.ResolveOrderSKUMappings)
			protected.POST("/order-sku-mappings/import", handler.ImportOrderSKUMappings)
			protected.GET("/order-sku-mappings", handler.ListOrderSKUMappings)
			protected.GET("/order-sku-mappings/group", handler.GetOrderSKUMappingGroup)
			protected.PUT("/order-sku-mappings", handler.UpsertOrderSKUMapping)
			protected.DELETE("/order-sku-mappings/:id", handler.DeactivateOrderSKUMapping)
			protected.GET("/order-field-mappings/:platform", handler.GetOrderFieldMappings)
			protected.PUT("/order-field-mappings/:platform", handler.ReplaceOrderFieldMappings)
			protected.POST("/order-tracking/import", handler.ImportOrderTrackingItems)
			protected.GET("/order-conversions", handler.ListOrderConversions)
			protected.GET("/order-conversions/:id/:file", handler.DownloadOrderConversion)
			protected.POST("/order-tracking/generated/check", handler.CheckGeneratedOrderTrackingItems)
			protected.POST("/order-tracking/mcf-excluded/notify", handler.NotifyMCFExcludedOrders)
			protected.POST("/order-tracking/mcf-email", handler.SendMCFEmail)
			protected.GET("/order-tracking", handler.ListOrderTrackingItems)
			protected.GET("/order-tracking/exportable", handler.ListExportableOrderTrackingItems)
			protected.POST("/order-tracking/export", handler.ExportOrderTrackingItems)
			protected.PUT("/order-tracking/:id", handler.UpdateOrderTrackingNumber)
			protected.PUT("/order-tracking/:id/status", handler.UpdateOrderTrackingStatus)
			protected.GET("/assets", handler.ListAssets)
			protected.GET("/asset-categories", handler.ListAssetCategories)
			protected.POST("/asset-categories", handler.CreateAssetCategory)
			protected.POST("/asset-categories/repair", handler.RepairAssetCategories)
			protected.POST("/asset-categories/reorder", handler.ReorderAssetCategories)
			protected.PUT("/asset-categories/:id", handler.UpdateAssetCategory)
			protected.DELETE("/asset-categories/:id", handler.DeleteAssetCategory)
			protected.POST("/assets", handler.CreateAsset)
			protected.POST("/assets/move-category", handler.MoveAssetCategoryAssets)
			protected.POST("/assets/move-style-references", handler.MoveSceneStyleReferenceAssets)
			protected.PUT("/assets/:id", handler.UpdateAsset)
			protected.DELETE("/assets/:id", handler.DeleteAsset)
			protected.POST("/assets/favorite", handler.SetAssetFavorite)
			protected.POST("/assets/usage", handler.RecordAssetUsage)

			admin := protected.Group("/")
			admin.Use(middleware.RequireAdmin())
			{
				// Menu Management (菜单管理)
				admin.GET("/admin/menus", handler.GetAllMenusAdmin)
				admin.POST("/admin/menus", handler.CreateMenu)
				admin.PUT("/admin/menus/:id", handler.UpdateMenu)
				admin.DELETE("/admin/menus/:id", handler.DeleteMenu)

				admin.GET("/settings", handler.GetSettings)
				admin.POST("/settings", handler.UpdateSettings)

				// Dashboard
				admin.GET("/dashboard/stats", handler.GetDashboardStats)
				admin.GET("/usage-logs/image-monthly-report/pdf", handler.GetMonthlyImageReportPDF)
				admin.POST("/usage-logs/image-monthly-report/send", handler.SendMonthlyImageReport)

				// User Management
				admin.GET("/users", handler.GetUsers)
				admin.POST("/users", handler.CreateUser)
				admin.PUT("/users/:id", handler.UpdateUser)
				admin.DELETE("/users/:id", handler.DeleteUser)
				admin.GET("/departments", handler.ListDepartments)
				admin.POST("/departments", handler.CreateDepartment)
				admin.PUT("/departments/:id", handler.UpdateDepartment)
				admin.DELETE("/departments/:id", handler.DeleteDepartment)

				// Model Management
				protected.GET("/models", handler.GetAiModels)
				admin.POST("/models", handler.CreateAiModel)
				admin.PUT("/models/:id", handler.UpdateAiModel)
				admin.DELETE("/models/:id", handler.DeleteAiModel)

				// Scene Style Library
				admin.GET("/admin/scene-style-categories", handler.AdminListSceneStyleCategories)
				admin.POST("/admin/scene-style-categories", handler.AdminCreateSceneStyleCategory)
				admin.PUT("/admin/scene-style-categories/:id", handler.AdminUpdateSceneStyleCategory)
				admin.DELETE("/admin/scene-style-categories/:id", handler.AdminDeleteSceneStyleCategory)
				admin.GET("/admin/scene-style-tags", handler.AdminListSceneStyleTags)
				admin.POST("/admin/scene-style-tags", handler.AdminCreateSceneStyleTag)
				admin.PUT("/admin/scene-style-tags/:id", handler.AdminUpdateSceneStyleTag)
				admin.DELETE("/admin/scene-style-tags/:id", handler.AdminDeleteSceneStyleTag)
				admin.GET("/admin/scene-generation/queue-overview", handler.ListAdminSceneQueueOverview)
				admin.POST("/admin/scene-style-references", handler.AdminCreateSceneStyleReference)
				admin.PUT("/admin/scene-style-references/:id", handler.AdminUpdateSceneStyleReference)
				admin.DELETE("/admin/scene-style-references/:id", handler.AdminDeleteSceneStyleReference)

				// Product Color Families
				admin.GET("/admin/product-color-families", handler.AdminListProductColorFamilies)
				admin.POST("/admin/product-color-families", handler.AdminCreateProductColorFamily)
				admin.PUT("/admin/product-color-families/:id", handler.AdminUpdateProductColorFamily)
				admin.DELETE("/admin/product-color-families/:id", handler.AdminDeleteProductColorFamily)
				admin.GET("/admin/reference-galleries", handler.ListReferenceGalleries)
				admin.POST("/admin/reference-galleries", handler.AdminCreateReferenceGallery)
				admin.PUT("/admin/reference-galleries/:id", handler.AdminUpdateReferenceGallery)
				admin.DELETE("/admin/reference-galleries/:id", handler.AdminDeleteReferenceGallery)
				admin.GET("/admin/product-spus", handler.AdminListProductSPUs)
				admin.GET("/admin/product-spus/:id", handler.AdminGetProductSPU)
				admin.POST("/admin/product-spus", handler.AdminCreateProductSPU)
				admin.PUT("/admin/product-spus/:id", handler.AdminUpdateProductSPU)
				admin.DELETE("/admin/product-spus/:id", handler.AdminDeleteProductSPU)
				admin.GET("/admin/product-spus/:id/attachments", handler.AdminListProductSPUAttachments)
				admin.POST("/admin/product-spus/:id/model", handler.AdminUploadProductSPUModel)
				admin.POST("/admin/product-spus/:id/attachments", handler.AdminUploadProductSPUAttachment)
				admin.PUT("/admin/product-spus/:id/attachments/:attachment_id", handler.AdminUpdateProductSPUAttachment)
				admin.DELETE("/admin/product-spus/:id/attachments/:attachment_id", handler.AdminDeleteProductSPUAttachment)
				admin.GET("/admin/product-skus", handler.AdminListProductSKUs)
				admin.POST("/admin/product-skus", handler.AdminCreateProductSKU)
				admin.PUT("/admin/product-skus/:id", handler.AdminUpdateProductSKU)
				admin.DELETE("/admin/product-skus/:id", handler.AdminDeleteProductSKU)
				admin.POST("/admin/product-skus/:id/files", handler.AdminUploadProductSKUFile)
				admin.GET("/admin/listing-keywords", handler.ListAdminListingKeywords)
				admin.POST("/admin/listing-keywords/expand", handler.ExpandAdminListingKeywords)
				admin.PUT("/admin/listing-keywords/:id", handler.UpdateAdminListingKeyword)
				admin.DELETE("/admin/listing-keywords/:id", handler.DeleteAdminListingKeyword)

				// Role Management
				admin.GET("/roles", handler.GetRoles)
				admin.POST("/roles", handler.CreateRole)
				admin.PUT("/roles/:id", handler.UpdateRole)
				admin.DELETE("/roles/:id", handler.DeleteRole)
				admin.GET("/all-menus", handler.GetAllMenus)
			}

			protected.GET("/scene-style-tags", handler.ListSceneStyleTags)
			protected.GET("/scene-style-categories", handler.ListSceneStyleCategories)
			protected.GET("/product-color-families", handler.ListProductColorFamilies)
			protected.GET("/reference-galleries", handler.ListReferenceGalleries)
			protected.GET("/reference-gallery-asset-keys", handler.GetReferenceGalleryAssetKeys)
			protected.POST("/scene-generation/tasks", handler.CreateSceneGenerationTask)
			protected.GET("/scene-generation/tasks", handler.ListSceneGenerationTasks)
			protected.GET("/scene-generation/tasks/:id", handler.GetSceneGenerationTask)
			protected.POST("/scene-generation/tasks/:id/save", handler.SaveSceneGenerationTaskToDesign)
			protected.POST("/scene-generation/tasks/:id/images/:sort/edit", handler.StartSceneGenerationImageEdit)
			protected.POST("/scene-generation/tasks/:id/images/:sort/accept", handler.AcceptSceneGenerationImageCandidate)
			protected.POST("/scene-generation/tasks/:id/images/:sort/reject", handler.RejectSceneGenerationImageCandidate)
			protected.DELETE("/scene-generation/tasks/:id", handler.DeleteSceneGenerationTask)
			protected.POST("/images/ensure-oss", handler.EnsureImageOSS)
			protected.GET("/workflow-projects", handler.ListWorkflowProjects)
			protected.POST("/workflow-projects", handler.CreateWorkflowProject)
			protected.GET("/workflow-projects/:project_id", handler.GetWorkflowProject)
			protected.PUT("/workflow-projects/:project_id", handler.UpdateWorkflowProject)
			protected.DELETE("/workflow-projects/:project_id", handler.DeleteWorkflowProject)
			protected.GET("/workflow-templates", handler.ListWorkflowTemplates)
			protected.POST("/workflow-templates", handler.CreateWorkflowTemplate)
			protected.DELETE("/workflow-templates/:template_id", handler.DeleteWorkflowTemplate)

			// 电商套图任务
			protected.POST("/ecom-suite/tasks", handler.CreateEcomSuiteTask)
			protected.POST("/smart-suite/tasks", handler.CreateSmartSuiteTask)
			protected.POST("/smart-suite/listing/evaluate", handler.EvaluateSmartSuiteListing)
			protected.GET("/smart-suite/conversations", handler.ListSmartSuiteConversations)
			protected.DELETE("/smart-suite/conversations/:session_id", handler.DeleteSmartSuiteConversation)
			protected.GET("/smart-suite/conversation", handler.GetSmartSuiteConversation)
			protected.POST("/smart-suite/conversation/turn", handler.HandleSmartSuiteConversationTurn)
			protected.POST("/smart-suite/tasks/:task_id/stop", handler.StopSmartSuiteTask)
			protected.POST("/smart-suite/tasks/:task_id/images/:image_id/edit", handler.StartSmartSuiteImageEdit)
			protected.POST("/smart-suite/tasks/:task_id/images/:image_id/accept", handler.AcceptSmartSuiteImageCandidate)
			protected.POST("/smart-suite/tasks/:task_id/images/:image_id/reject", handler.RejectSmartSuiteImageCandidate)
			protected.GET("/ecom-suite/tasks", handler.ListEcomSuiteTasks)
			protected.GET("/admin/ecom-suite/tasks", handler.ListAdminEcomSuiteTasks)
			protected.GET("/admin/ecom-suite/tasks/:task_id", handler.GetAdminEcomSuiteTask)
			protected.GET("/ecom-suite/tasks/:task_id", handler.GetEcomSuiteTask)

			// AI Chat & Image Generation
			protected.POST("/chat", handler.Chat)
			protected.POST("/image/generate", handler.GenerateImage)
			protected.POST("/image/generate-async", handler.CreateImageGenerationTask)
			protected.GET("/image/generate-async/:task_id", handler.GetImageGenerationTask)
			protected.POST("/video/generate", handler.GenerateVideo)
			protected.GET("/video/tasks/:model/:task_id", handler.GetVideoTask)

			// Image Batch CRUD (异步任务批次管理)
			protected.POST("/image/batches", handler.CreateImageBatch)
			protected.GET("/image/batches", handler.GetImageBatches)
			protected.PUT("/image/batches/:batch_key/items/:item_id", handler.UpdateImageBatchItem)
			protected.PUT("/image/batches/:batch_key/status", handler.UpdateImageBatchStatus)

			protected.POST("/image/generate-prompt-variations", handler.GeneratePromptVariations)

			// Multi-angle Generation (多角度生成)
			protected.POST("/multi-angle/analyze", handler.AnalyzeProductImage)
			protected.POST("/multi-angle/generate", handler.GenerateMultiAngleImage)
			protected.POST("/multi-angle/tasks", handler.CreateMultiAngleTask)
			protected.GET("/multi-angle/tasks", handler.ListMultiAngleTasks)
			protected.GET("/multi-angle/tasks/:task_id", handler.GetMultiAngleTask)

			// Image to prompt (图片转提示词，对话/视觉模型)
			protected.POST("/image-to-prompt/generate", handler.ImageToPrompt)

			// Eraser (一键消除文字/水印)
			protected.POST("/eraser", handler.EraserImage)

			// Remove Background (去除背景)
			protected.POST("/remove-bg", handler.RemoveBg)

			// Image Translate (图片文字翻译)
			protected.POST("/image-translate", handler.ImageTranslate)

			// White Background Image (白底图生成)
			protected.POST("/white-bg/tasks", handler.CreateWhiteBgTask)
			protected.GET("/white-bg/tasks", handler.ListWhiteBgTasks)
			protected.GET("/white-bg/tasks/:id", handler.GetWhiteBgTask)
			protected.POST("/white-bg/tasks/:id/save", handler.SaveWhiteBgTaskToDesign)
			protected.DELETE("/white-bg/tasks/:id", handler.DeleteWhiteBgTask)

			// Product Replace (产品替换)
			protected.POST("/product-replace/tasks", handler.CreateProductReplaceTask)
			protected.GET("/product-replace/tasks", handler.ListProductReplaceTasks)
			protected.GET("/product-replace/tasks/:id", handler.GetProductReplaceTask)
			protected.POST("/product-replace/tasks/:id/save", handler.SaveProductReplaceTaskToDesign)

			// Depth of Field Generation (景深图生成)
			protected.POST("/depth/tasks", handler.CreateDepthTask)
			protected.GET("/depth/tasks", handler.ListDepthTasks)
			protected.GET("/depth/tasks/:id", handler.GetDepthTask)

			// Light Shadow Optimization (光影优化)
			protected.POST("/light-shadow/tasks", handler.CreateLightShadowTask)
			protected.GET("/light-shadow/tasks", handler.ListLightShadowTasks)
			protected.GET("/light-shadow/tasks/:id", handler.GetLightShadowTask)

			// Image Upload (通用图片上传)
			protected.POST("/upload/image", handler.UploadImage)

			// STP File Upload (STP 文件上传)
			protected.POST("/stp/model", handler.UploadStp)

			// Prompt Management (提示词管理，写操作需登录)
			protected.POST("/prompt-groups", handler.CreatePromptGroup)
			protected.PUT("/prompt-groups/reorder", handler.ReorderPromptGroups)
			protected.PUT("/prompt-groups/:id", handler.UpdatePromptGroup)
			protected.DELETE("/prompt-groups/:id", handler.DeletePromptGroup)
			protected.POST("/prompts", handler.CreatePrompt)
			protected.PUT("/prompts/reorder", handler.ReorderPrompts)
			protected.PUT("/prompts/:id", handler.UpdatePrompt)
			protected.DELETE("/prompts/:id", handler.DeletePrompt)
			admin.GET("/admin/prompts", handler.GetAdminPrompts)
			admin.PUT("/admin/prompts/:id/status", handler.UpdatePromptStatus)

			// Product Prompt Management (产品图提示词管理)
			protected.GET("/product-prompts", handler.GetProductPrompts)
			protected.POST("/product-prompts", handler.CreateProductPrompt)
			protected.PUT("/product-prompts/:id", handler.UpdateProductPrompt)
			protected.DELETE("/product-prompts/:id", handler.DeleteProductPrompt)

			// STP File Upload & Render Status (六面图渲染)
			protected.POST("/product-prompts/:id/stp", handler.UploadStpFile)
			protected.GET("/product-prompts/:id/render-status", handler.GetRenderStatus)

			// STP Single Render (白底图)
			protected.POST("/stp-render/single", handler.CreateStpRenderTask)
			protected.GET("/stp-render/single/:task_id", handler.GetStpRenderTask)

			// STP Mesh Preview (获取 STP 的 3D mesh 数据用于预览)
			protected.GET("/stp/mesh-preview", handler.GetStpMeshPreview)
			protected.GET("/stp/mesh-preview/:task_id", handler.GetStpMeshPreviewTask)

			// Product Image Task (产品图生成任务)
			protected.POST("/product-image/analyze", handler.AnalyzeProductImageWithPrompt)

			// File Download（代理下载 uploads 文件，设置 Content-Disposition: attachment）
			protected.GET("/download", handler.DownloadUploadFile)
			protected.POST("/image-metadata/synthetic-performer", handler.AddSyntheticPerformerMetadataImage)
			protected.POST("/image-metadata/synthetic-performer/batch", handler.AddSyntheticPerformerMetadataBatch)
			protected.POST("/downloads/archive-to-nas", handler.UploadDownloadArchiveToNAS)

			// Usage Logs
			protected.GET("/usage-logs", handler.GetUsageLogs)
			protected.POST("/usage-logs/record-download", handler.RecordUsageLogDownload)
			protected.GET("/usage-logs/:id/download", handler.DownloadUsageLogImage)
			protected.POST("/usage-logs/:id/save", handler.SaveUsageLogToDesign)
			admin.GET("/usage-logs/image-user-stats", handler.GetImageUsageUserStats)

			// Announcement Management (公告管理)
			protected.GET("/announcements", handler.GetAnnouncements)
			protected.POST("/announcements", handler.CreateAnnouncement)
			protected.PUT("/announcements/:id", handler.UpdateAnnouncement)
			protected.DELETE("/announcements/:id", handler.DeleteAnnouncement)

			// Announcement User (用户端公告)
			protected.GET("/me/unread-announcements", handler.GetUnreadAnnouncements)
			protected.POST("/announcements/:id/read", handler.MarkAnnouncementRead)

			// Feedback (意见反馈)
			protected.POST("/feedbacks", handler.SubmitFeedback)
			admin.GET("/admin/feedbacks", handler.GetFeedbacks)
			admin.PUT("/admin/feedbacks/:id", handler.UpdateFeedback)
			admin.DELETE("/admin/feedbacks/:id", handler.DeleteFeedback)

			protected.GET("/ping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "pong"})
			})
		}
	}

	return r
}
