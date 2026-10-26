package database

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"advertising-image/backend/internal/config"
	"advertising-image/backend/internal/model"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
		config.AppConfig.Database.Host,
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.DBName,
		config.AppConfig.Database.Port,
		config.AppConfig.Database.SSLMode,
	)
	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		log.Fatalf("Failed to load Asia/Shanghai timezone: %v", locErr)
	}
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc: func() time.Time {
			return time.Now().In(loc)
		},
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	if sqlDB, e := DB.DB(); e == nil {
		sqlDB.SetMaxOpenConns(20)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
		sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	}
	log.Println("Connected to PostgreSQL successfully")
}

func RunMigrations() {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		config.AppConfig.Database.User,
		config.AppConfig.Database.Password,
		config.AppConfig.Database.Host,
		config.AppConfig.Database.Port,
		config.AppConfig.Database.DBName,
		config.AppConfig.Database.SSLMode,
	)

	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Printf("Skip SQL migrations: %v", err)
	} else {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			var dirtyErr migrate.ErrDirty
			if errors.As(err, &dirtyErr) {
				log.Printf("Warning: migration dirty at version %d, forcing previous version and retrying...", dirtyErr.Version)
				forceVersion := int(dirtyErr.Version) - 1
				if forceVersion < 0 {
					forceVersion = -1
				}
				if forceErr := m.Force(forceVersion); forceErr != nil {
					log.Fatalf("Failed to force migration version: %v", forceErr)
				}
				if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
					log.Fatalf("Failed to run migrations after dirty fix: %v", retryErr)
				}
			} else {
				log.Fatalf("Failed to run migrations: %v", err)
			}
		}
		log.Println("Database migrations applied successfully")
	}
	// Keep the user flag available even when an older deployment has already
	// advanced its migration table without applying the long-login migration.
	if err := DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS long_login BOOLEAN NOT NULL DEFAULT FALSE`).Error; err != nil {
		log.Fatalf("Failed to ensure users.long_login column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE`).Error; err != nil {
		log.Fatalf("Failed to ensure users.enabled column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS finance_permissions jsonb NOT NULL DEFAULT '{}'::jsonb`).Error; err != nil {
		log.Fatalf("Failed to ensure users.finance_permissions column: %v", err)
	}

	runUsageLogAutoMigrate()
	runPromptAutoMigrate()
	runProductPromptAutoMigrate()
	runSceneAutoMigrate()
	runImageTaskAutoMigrate()
	runSceneStyleCategoryAutoMigrate()
	runEcomSuiteAutoMigrate()
	runListingKeywordAutoMigrate()
	if err := DB.AutoMigrate(&model.BuyerReview{}); err != nil {
		log.Fatalf("Failed to migrate buyer reviews: %v", err)
	}
	if err := migrateBuyerReviewFingerprints(); err != nil {
		log.Fatalf("Failed to migrate buyer review fingerprints: %v", err)
	}
	runReferenceGalleryAutoMigrate()
	runProductColorFamilyAutoMigrate()
	runDepartmentAutoMigrate()
	runUserOrderPlatformAutoMigrate()
	runWhiteBgAutoMigrate()
	runMultiAngleAutoMigrate()
	runDepthAutoMigrate()
	runLightShadowAutoMigrate()
	runImageBatchAutoMigrate()
	runWorkflowProjectAutoMigrate()
	runProductCatalogAutoMigrate()
	runOrderSKUMappingAutoMigrate()
	runAssetAutoMigrate()
	runWorkRequestAutoMigrate()
	if err := DB.AutoMigrate(&model.ImageViolationExample{}); err != nil {
		log.Fatalf("Failed to auto migrate image violation examples: %v", err)
	}
	runTaxDataAutoMigrate()
	if err := DB.AutoMigrate(&model.CustomsDeclaration{}); err != nil {
		log.Fatalf("Failed to migrate customs declarations: %v", err)
	}
	if err := DB.AutoMigrate(&model.DailyCall{}); err != nil {
		log.Fatalf("Failed to auto migrate DailyCall schema: %v", err)
	}
	for _, column := range []string{"name", "brief_of_issue", "quality_result", "reason"} {
		if err := DB.Exec("ALTER TABLE daily_calls ALTER COLUMN " + column + " DROP NOT NULL").Error; err != nil {
			log.Printf("Failed to make daily_calls.%s nullable: %v", column, err)
		}
	}
}

// Keep legacy financial-row indexes from rejecting user-confirmed reimports.
func dropAmazonFinanceDedupIndexes(db *gorm.DB) error {
	return db.Exec(`DO $$
	DECLARE
		target regclass := 'amazon_tax_finances'::regclass;
		entry record;
	BEGIN
		FOR entry IN SELECT conname FROM pg_constraint WHERE conrelid = target AND contype = 'u'
		LOOP
			EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I', target, entry.conname);
		END LOOP;
		FOR entry IN
			SELECT indexrelid::regclass AS index_name FROM pg_index
			WHERE indrelid = target AND indisunique AND NOT indisprimary
		LOOP
			EXECUTE format('DROP INDEX %s', entry.index_name);
		END LOOP;
	END $$`).Error
}

func runTaxDataAutoMigrate() {
	if err := DB.AutoMigrate(&model.TaxCatalogEntry{}, &model.TaxDataBatch{}, &model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxFeeRule{}, &model.TaxQuarter{}, &model.TaxImportLog{}, &model.TaxDownloadLog{}, &model.TaxExchangeRate{}, &model.TikTokTaxBatch{}, &model.TikTokTaxTransactionSummary{}, &model.TikTokTaxRawRow{}, &model.TikTokTaxDetail{}, &model.TikTokTaxQuarter{}, &model.TikTokTaxAdBatch{}, &model.TikTokTaxAd{}, &model.TikTokTaxFBTBatch{}, &model.TikTokTaxFBT{}, &model.TikTokSettlementBatch{}, &model.TikTokSettlement{}, &model.TikTokOrderBatch{}, &model.TikTokOrder{}, &model.AmazonTaxShipment{}, &model.AmazonTaxFinance{}, &model.AmazonProductCategory{}); err != nil {
		log.Fatalf("Failed to migrate tax data: %v", err)
	}
	if err := dropAmazonFinanceDedupIndexes(DB); err != nil {
		log.Fatalf("Failed to drop Amazon finance identity indexes: %v", err)
	}
	for _, table := range []string{"tax_data_batches", "tax_raw_rows", "tax_import_logs", "tax_download_logs", "tik_tok_tax_batches", "tik_tok_tax_raw_rows", "tik_tok_tax_details", "tik_tok_tax_ad_batches", "tik_tok_tax_ads", "tik_tok_tax_fbt_batches", "tik_tok_tax_fbts", "tik_tok_tax_transaction_summaries", "tik_tok_settlement_batches", "tik_tok_settlements", "tik_tok_order_batches", "tik_tok_orders", "amazon_tax_shipments", "amazon_tax_finances"} {
		if err := DB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS store_id varchar(100)", table)).Error; err != nil {
			log.Fatalf("Failed to add tax store column: %v", err)
		}
		if err := DB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS site_id varchar(20)", table)).Error; err != nil {
			log.Fatalf("Failed to add tax site column: %v", err)
		}
	}
	// Older installations created Amazon report tables before the computed period
	// columns were part of the model. Keep the quarterly summary query compatible
	// with those databases during startup migration.
	if err := DB.Exec(`ALTER TABLE amazon_tax_shipments ADD COLUMN IF NOT EXISTS month varchar(7);
		ALTER TABLE amazon_tax_shipments ADD COLUMN IF NOT EXISTS quarter varchar(7);
		ALTER TABLE amazon_tax_finances ADD COLUMN IF NOT EXISTS month varchar(7);
		ALTER TABLE amazon_tax_finances ADD COLUMN IF NOT EXISTS quarter varchar(7);
		ALTER TABLE tax_processed_rows ADD COLUMN IF NOT EXISTS month varchar(7);
		ALTER TABLE tax_processed_rows ADD COLUMN IF NOT EXISTS quarter varchar(7);`).Error; err != nil {
		log.Fatalf("Failed to migrate Amazon report periods: %v", err)
	}
	if err := DB.Exec(`DROP INDEX IF EXISTS uni_tik_tok_tax_batches_file_hash;
		DROP INDEX IF EXISTS tax_batch_file;
		DROP INDEX IF EXISTS tax_raw_identity;
		DROP INDEX IF EXISTS uni_tik_tok_tax_raw_rows_fingerprint;
		DROP INDEX IF EXISTS uni_tik_tok_tax_ad_batches_file_hash;
		DROP INDEX IF EXISTS uni_tik_tok_tax_fbt_batches_file_hash;
		DROP INDEX IF EXISTS uni_tik_tok_order_batches_file_hash;
		DROP INDEX IF EXISTS uni_tik_tok_settlement_batches_file_hash;
		DROP INDEX IF EXISTS uni_amazon_tax_shipments_fingerprint;
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tax_batch_site_file ON tax_data_batches (store_id, site_id, file_hash);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tax_raw_site_identity ON tax_raw_rows (store_id, site_id, fingerprint);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tiktok_tax_batch_site_file ON tik_tok_tax_batches (store_id, site_id, file_hash);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tiktok_tax_raw_site_identity ON tik_tok_tax_raw_rows (store_id, site_id, fingerprint);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tiktok_tax_ad_batch_site_file ON tik_tok_tax_ad_batches (store_id, site_id, file_hash);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tiktok_tax_fbt_batch_site_file ON tik_tok_tax_fbt_batches (store_id, site_id, file_hash);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tiktok_order_batch_site_file ON tik_tok_order_batches (store_id, site_id, file_hash);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_tiktok_settlement_batch_site_file ON tik_tok_settlement_batches (store_id, site_id, file_hash);
		DROP INDEX IF EXISTS idx_amazon_shipment_site_identity`).Error; err != nil {
		log.Fatalf("Failed to migrate tax site identity indexes: %v", err)
	}
	for _, item := range []struct{ table, store string }{
		{"tik_tok_tax_batches", "Forious"}, {"tik_tok_tax_raw_rows", "Forious"}, {"tik_tok_tax_details", "Forious"}, {"tik_tok_tax_ads", "Forious"}, {"tik_tok_tax_fbts", "Forious"}, {"tik_tok_settlements", "Forious"}, {"tik_tok_orders", "Forious"}, {"tik_tok_tax_transaction_summaries", "Forious"}, {"amazon_tax_shipments", "sunoutly"}, {"amazon_tax_finances", "sunoutly"},
	} {
		if err := DB.Exec(fmt.Sprintf("UPDATE %s SET store_id = ? WHERE store_id IS NULL OR store_id = ''", item.table), item.store).Error; err != nil {
			log.Fatalf("Failed to backfill tax stores: %v", err)
		}
	}
	for _, table := range []string{"tax_data_batches", "tax_raw_rows", "tax_import_logs", "tax_download_logs", "tik_tok_tax_batches", "tik_tok_tax_raw_rows", "tik_tok_tax_details", "tik_tok_tax_ad_batches", "tik_tok_tax_ads", "tik_tok_tax_fbt_batches", "tik_tok_tax_fbts", "tik_tok_tax_transaction_summaries", "tik_tok_settlement_batches", "tik_tok_settlements", "tik_tok_order_batches", "tik_tok_orders", "amazon_tax_shipments", "amazon_tax_finances"} {
		if err := DB.Exec(fmt.Sprintf("UPDATE %s SET site_id = 'US' WHERE site_id IS NULL OR site_id = ''", table)).Error; err != nil {
			log.Fatalf("Failed to backfill tax sites: %v", err)
		}
	}
	if err := DB.Exec(`DROP INDEX IF EXISTS uni_tik_tok_tax_transaction_summaries_quarter; DROP INDEX IF EXISTS idx_tiktok_transaction_summary_store; CREATE UNIQUE INDEX IF NOT EXISTS idx_tiktok_transaction_summary_store ON tik_tok_tax_transaction_summaries (store_id, site_id, quarter)`).Error; err != nil {
		log.Fatalf("Failed to migrate TikTok transaction stores: %v", err)
	}
	// Shipment reports are shared by all sites of one store. Remove historical
	// cross-site duplicates before enforcing the store-scoped identity.
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DROP INDEX IF EXISTS idx_amazon_shipment_identity;
			DROP INDEX IF EXISTS amazon_shipment_site_identity;
			DROP INDEX IF EXISTS idx_amazon_shipment_site_identity;
			DROP INDEX IF EXISTS idx_amazon_shipment_store_identity`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`WITH ranked AS (
			SELECT id, ROW_NUMBER() OVER (
				PARTITION BY store_id,
					COALESCE(BTRIM(data->>'亚马逊订单编号'), ''),
					COALESCE(BTRIM(data->>'货件编号'), ''),
					COALESCE(BTRIM(data->>'货件商品编号'), '')
				ORDER BY id
			) AS row_number
			FROM amazon_tax_shipments
		)
		DELETE FROM amazon_tax_shipments
		WHERE id IN (SELECT id FROM ranked WHERE row_number > 1)`).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE UNIQUE INDEX idx_amazon_shipment_store_identity ON amazon_tax_shipments
			(store_id,
			 (COALESCE(BTRIM(data->>'亚马逊订单编号'), '')),
			 (COALESCE(BTRIM(data->>'货件编号'), '')),
			 (COALESCE(BTRIM(data->>'货件商品编号'), '')))`).Error
	}); err != nil {
		log.Fatalf("Failed to index Amazon shipment identities: %v", err)
	}

	// A lookup index supports settlement import deduplication without deleting historical duplicates.
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_tiktok_settlement_identity_lookup ON tik_tok_settlements
		(statement_date, (COALESCE(BTRIM(data->>'Statement ID'), '')), (COALESCE(BTRIM(data->>'Order/adjustment ID'), '')))`).Error; err != nil {
		log.Fatalf("Failed to index TikTok settlement identities: %v", err)
	}
	// Backfill TikTok originals imported by RPA.
	for _, item := range []struct{ table, kind, count string }{{"tik_tok_tax_batches", "details", "raw_rows"}, {"tik_tok_tax_ad_batches", "ads", "row_count"}} {
		sql := fmt.Sprintf(`INSERT INTO tax_import_logs (platform, report_type, batch_id, importer, file_name, file_content, source_rows, imported_rows, skipped_rows, status, message, created_at)
   SELECT 'tiktok', ?, b.id, 'RPA', b.file_name, b.file_content, b.%s, b.%s, 0, 'locked', '', b.created_at FROM %s b
   WHERE NOT EXISTS (SELECT 1 FROM tax_import_logs l WHERE l.platform = 'tiktok' AND l.report_type = ? AND l.batch_id = b.id)`, item.count, item.count, item.table)
		if err := DB.Exec(sql, item.kind, item.kind).Error; err != nil {
			log.Fatalf("Failed to backfill TikTok import logs: %v", err)
		}
	}
	if err := DB.Model(&model.TaxImportLog{}).Where("platform = ? AND importer <> ?", "tiktok", "RPA").Update("importer", "RPA").Error; err != nil {
		log.Fatalf("Failed to normalize TikTok importers: %v", err)
	}
	// Preserve audit access to files imported before import logging was introduced.
	if err := DB.Exec(`INSERT INTO tax_import_logs
		(batch_id, store_id, importer, file_name, file_content, source_rows, imported_rows, skipped_rows, status, message, created_at)
		SELECT b.id, b.store_id, '历史导入（身份未记录）', b.file_name, b.file_content, b.raw_rows, b.raw_rows, 0, b.status, '', b.created_at
		FROM tax_data_batches b WHERE NOT EXISTS (SELECT 1 FROM tax_import_logs l WHERE l.batch_id = b.id)`).Error; err != nil {
		log.Fatalf("Failed to backfill tax import logs: %v", err)
	}
	rules, err := model.DefaultTaxFeeRules()
	if err != nil {
		log.Fatalf("Invalid tax rules: %v", err)
	}
	for _, rule := range rules {
		if err := DB.Where("match_key = ?", rule.MatchKey).FirstOrCreate(&rule).Error; err != nil {
			log.Fatalf("Failed to seed tax rules: %v", err)
		}
	}
}

func runListingKeywordAutoMigrate() {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS listing_keywords (
			id bigserial PRIMARY KEY,
			platform varchar(50) NOT NULL,
			country_code varchar(20) NOT NULL DEFAULT '',
			category varchar(200) NOT NULL DEFAULT '',
			term varchar(255) NOT NULL,
			stage varchar(20) NOT NULL DEFAULT 'expanded',
			status varchar(20) NOT NULL DEFAULT 'candidate',
			tags jsonb NOT NULL DEFAULT '[]'::jsonb,
			source varchar(40) NOT NULL DEFAULT 'operator',
			usage_count integer NOT NULL DEFAULT 0,
			notes text NOT NULL DEFAULT '',
			last_used_at timestamptz,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_listing_keywords_scope_term ON listing_keywords(platform, country_code, category, term)`,
		`CREATE INDEX IF NOT EXISTS idx_listing_keywords_lookup ON listing_keywords(platform, country_code, category, status, stage)`,
	}
	for _, statement := range statements {
		if err := DB.Exec(statement).Error; err != nil {
			log.Fatalf("Failed to update listing keywords schema: %v", err)
		}
	}
	log.Println("Listing keyword library auto migration completed")
}

// runWorkRequestAutoMigrate keeps the user-facing request list available when a
// deployment has already advanced the SQL migration version without running
// the work request migration file. Keep this as SQL rather than GORM
// AutoMigrate: WorkRequest preloads User, and AutoMigrate would inspect the
// related users table and potentially alter legacy constraints.
func runWorkRequestAutoMigrate() {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS work_requests (
			id bigserial PRIMARY KEY,
			work_no varchar(32) NOT NULL UNIQUE,
			type varchar(20) NOT NULL DEFAULT 'video' CHECK (type = 'video'),
			title varchar(200) NOT NULL,
			status varchar(30) NOT NULL DEFAULT 'pending',
			creator_id bigint NOT NULL,
			assignee_id bigint,
			progress integer NOT NULL DEFAULT 0,
			progress_before_review integer NOT NULL DEFAULT 10,
			content jsonb NOT NULL DEFAULT '{}'::jsonb,
			delivery_note text NOT NULL DEFAULT '',
			delivery_paths jsonb NOT NULL DEFAULT '[]'::jsonb,
			rejection_reason text NOT NULL DEFAULT '',
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_work_requests_work_no ON work_requests(work_no)`,
		`CREATE INDEX IF NOT EXISTS idx_work_requests_type ON work_requests(type)`,
		`CREATE INDEX IF NOT EXISTS idx_work_requests_status ON work_requests(status)`,
		`CREATE INDEX IF NOT EXISTS idx_work_requests_creator_id ON work_requests(creator_id)`,
		`CREATE INDEX IF NOT EXISTS idx_work_requests_assignee_id ON work_requests(assignee_id)`,
		`CREATE INDEX IF NOT EXISTS idx_work_requests_created_at ON work_requests(created_at DESC)`,
	} {
		if err := DB.Exec(stmt).Error; err != nil {
			log.Fatalf("Failed to update work request schema: %v (stmt=%s)", err, stmt)
		}
	}
	log.Println("Work request auto migration completed")
}

func runUserOrderPlatformAutoMigrate() {
	if err := DB.AutoMigrate(&model.UserOrderPlatform{}); err != nil {
		log.Fatalf("Failed to auto migrate user order platforms: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS order_platforms_configured boolean NOT NULL DEFAULT false`).Error; err != nil {
		log.Fatalf("Failed to add users.order_platforms_configured: %v", err)
	}
	if err := DB.Exec(`
		INSERT INTO user_order_platforms (user_id, platform)
		SELECT id, CASE
			WHEN UPPER(TRIM(platform)) = 'TEMU' THEN 'TEMU'
			WHEN UPPER(REPLACE(TRIM(platform), ' ', '')) IN ('TIKTOK', 'TIKTOKSHOP', 'TK') THEN 'TIKTOK'
			WHEN UPPER(REPLACE(TRIM(platform), '''', '')) IN ('LOWES', 'LOWE') THEN 'LOWES'
		END
		FROM users
		WHERE order_platforms_configured = false
			AND UPPER(REPLACE(REPLACE(TRIM(platform), ' ', ''), '''', '')) IN ('TEMU', 'TIKTOK', 'TIKTOKSHOP', 'TK', 'LOWES', 'LOWE')
		ON CONFLICT (user_id, platform) DO NOTHING
	`).Error; err != nil {
		log.Fatalf("Failed to backfill user order platforms: %v", err)
	}
	if err := DB.Model(&model.User{}).Where("order_platforms_configured = false").Update("order_platforms_configured", true).Error; err != nil {
		log.Fatalf("Failed to mark user order platforms as migrated: %v", err)
	}
	log.Println("User order platform auto migration completed")
}

func runOrderSKUMappingAutoMigrate() {
	// The old schema allowed only one row per platform SKU. Drop that index first
	// so AutoMigrate can create the platform SKU + Amazon SKU mapping key.
	if err := DB.Exec(`DROP INDEX IF EXISTS idx_order_sku_mapping_platform_sku`).Error; err != nil {
		log.Fatalf("Failed to drop legacy order SKU mapping index: %v", err)
	}
	if err := DB.Exec(`DROP INDEX IF EXISTS idx_order_sku_mappings_current_amazon`).Error; err != nil {
		log.Fatalf("Failed to drop legacy current Amazon SKU index: %v", err)
	}
	if err := DB.Exec(`DROP INDEX IF EXISTS idx_order_tracking_platform_order_sku`).Error; err != nil {
		log.Fatalf("Failed to drop legacy order tracking item index: %v", err)
	}
	if err := DB.AutoMigrate(&model.OrderSKUMapping{}, &model.OrderFieldMapping{}, &model.OrderTrackingItem{}, &model.OrderConversion{}); err != nil {
		log.Fatalf("Failed to auto migrate order SKU mapping schema: %v", err)
	}
	if err := DB.Exec(`
		WITH current_ranked AS (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY platform, platform_sku ORDER BY id ASC) AS position
			FROM order_sku_mappings WHERE status = 'active' AND is_current = true
		)
		UPDATE order_sku_mappings SET is_current = false
		WHERE id IN (SELECT id FROM current_ranked WHERE position > 1)
	`).Error; err != nil {
		log.Fatalf("Failed to normalize current order SKU mappings: %v", err)
	}
	if err := DB.Exec(`
		WITH ranked AS (
			SELECT id, platform, platform_sku, ROW_NUMBER() OVER (PARTITION BY platform, platform_sku ORDER BY id ASC) AS position
			FROM order_sku_mappings WHERE status = 'active'
		)
		UPDATE order_sku_mappings AS mappings SET is_current = true
		FROM ranked
		WHERE mappings.id = ranked.id AND ranked.position = 1
			AND NOT EXISTS (
				SELECT 1 FROM order_sku_mappings current_mapping
				WHERE current_mapping.platform = ranked.platform AND current_mapping.platform_sku = ranked.platform_sku
					AND current_mapping.status = 'active' AND current_mapping.is_current = true
			)
	`).Error; err != nil {
		log.Fatalf("Failed to backfill current order SKU mappings: %v", err)
	}
	if err := DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_order_sku_mappings_current_platform ON order_sku_mappings(platform, platform_sku) WHERE status = 'active' AND is_current = true`).Error; err != nil {
		log.Fatalf("Failed to create current order SKU mapping index: %v", err)
	}
	if err := DB.Model(&model.OrderTrackingItem{}).
		Where("tracking_number <> '' AND status = ?", "pending_shipment").
		Update("status", "shipped").Error; err != nil {
		log.Fatalf("Failed to backfill order tracking statuses: %v", err)
	}
	log.Println("Order mapping auto migration completed")
}

func runUsageLogAutoMigrate() {
	if err := DB.AutoMigrate(&model.UsageLog{}); err != nil {
		log.Fatalf("Failed to auto migrate usage_log schema: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS is_saved_to_design boolean NOT NULL DEFAULT false`).Error; err != nil {
		log.Fatalf("Failed to add usage_logs.is_saved_to_design column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS download_count integer NOT NULL DEFAULT 0`).Error; err != nil {
		log.Fatalf("Failed to add usage_logs.download_count column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS saved_at timestamptz`).Error; err != nil {
		log.Fatalf("Failed to add usage_logs.saved_at column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS reference_image_url varchar(1000) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add usage_logs.reference_image_url column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS reference_image_urls text NOT NULL DEFAULT '[]'`).Error; err != nil {
		log.Fatalf("Failed to add usage_logs.reference_image_urls column: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_logs_status ON usage_logs(status)`).Error; err != nil {
		log.Fatalf("Failed to create idx_usage_logs_status: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_logs_user_id ON usage_logs(user_id)`).Error; err != nil {
		log.Fatalf("Failed to create idx_usage_logs_user_id: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_logs_created_at ON usage_logs(created_at DESC)`).Error; err != nil {
		log.Fatalf("Failed to create idx_usage_logs_created_at: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_usage_logs_user_status ON usage_logs(user_id, status)`).Error; err != nil {
		log.Fatalf("Failed to create idx_usage_logs_user_status: %v", err)
	}
	if err := DB.Exec(`UPDATE usage_logs SET request_body = '', response_body = '' WHERE request_body != '' OR response_body != ''`).Error; err != nil {
		log.Printf("Warning: Failed to clear usage_logs payload columns: %v", err)
	} else {
		log.Println("Cleared usage_logs request_body/response_body data")
	}
	log.Println("Usage log auto migration completed")
}

func runProductPromptAutoMigrate() {
	if err := DB.AutoMigrate(&model.ProductPrompt{}); err != nil {
		log.Fatalf("Failed to auto migrate product_prompt schema: %v", err)
	}
	productPromptDDL := []string{
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS color varchar(120) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS material varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS product_params text NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS usage_scenarios text NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS package_contents text NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS installation_steps text NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS installation_notes text NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS forbidden_fabrication text NOT NULL DEFAULT ''`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS detail_images text NOT NULL DEFAULT '[]'`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS structure_images text NOT NULL DEFAULT '[]'`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS size_images text NOT NULL DEFAULT '[]'`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS scenario_images text NOT NULL DEFAULT '[]'`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS accessory_images text NOT NULL DEFAULT '[]'`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS package_images text NOT NULL DEFAULT '[]'`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS available_scopes text NOT NULL DEFAULT '[]'`,
		`ALTER TABLE product_prompts ADD COLUMN IF NOT EXISTS white_bg_image_url varchar(500) NOT NULL DEFAULT ''`,
	}
	for _, ddl := range productPromptDDL {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Fatalf("Failed to update product_prompts schema: %v", err)
		}
	}
	log.Println("GORM product_prompt auto migration completed")
}

func runPromptAutoMigrate() {
	if err := DB.AutoMigrate(&model.PromptGroup{}, &model.Prompt{}); err != nil {
		log.Fatalf("Failed to auto migrate prompt schema: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts ALTER COLUMN group_id DROP NOT NULL`).Error; err != nil {
		log.Fatalf("Failed to relax prompts.group_id not-null constraint: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts ALTER COLUMN group_id SET DEFAULT 0`).Error; err != nil {
		log.Fatalf("Failed to set prompts.group_id default: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts DROP CONSTRAINT IF EXISTS prompts_group_id_fkey`).Error; err != nil {
		log.Fatalf("Failed to drop prompts.group_id foreign key: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts DROP CONSTRAINT IF EXISTS fk_prompt_groups_prompts`).Error; err != nil {
		log.Fatalf("Failed to drop legacy prompts group foreign key: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts ADD COLUMN IF NOT EXISTS images text NOT NULL DEFAULT '[]'`).Error; err != nil {
		log.Fatalf("Failed to add prompts.images column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts ADD COLUMN IF NOT EXISTS status varchar(20) NOT NULL DEFAULT 'approved'`).Error; err != nil {
		log.Fatalf("Failed to add prompts.status column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts ADD COLUMN IF NOT EXISTS source varchar(20) NOT NULL DEFAULT 'admin'`).Error; err != nil {
		log.Fatalf("Failed to add prompts.source column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE prompts ADD COLUMN IF NOT EXISTS shape varchar(20) NOT NULL DEFAULT 'square'`).Error; err != nil {
		log.Fatalf("Failed to add prompts.shape column: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_prompts_shape ON prompts(shape)`).Error; err != nil {
		log.Fatalf("Failed to create prompts.shape index: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_prompts_status ON prompts(status)`).Error; err != nil {
		log.Fatalf("Failed to create prompts.status index: %v", err)
	}
	log.Println("GORM prompt auto migration completed")
}

func runSceneAutoMigrate() {
	if err := DB.AutoMigrate(
		&model.SceneStyleTag{},
		&model.SceneStyleReference{},
		&model.SceneGenerationTask{},
		&model.SceneGenerationImage{},
		&model.Setting{},
	); err != nil {
		log.Fatalf("Failed to auto migrate scene schema: %v", err)
	}
	if err := DB.Exec(`INSERT INTO settings (key, value, description, created_at, updated_at)
		SELECT 'scene_generation_user_running_image_limit', '8', '单用户场景图运行中图片数上限', NOW(), NOW()
		WHERE NOT EXISTS (SELECT 1 FROM settings WHERE key = 'scene_generation_user_running_image_limit')`).Error; err != nil {
		log.Fatalf("Failed to seed scene generation image limit setting: %v", err)
	}
	if err := DB.Exec(`INSERT INTO menus (parent_id, title, path, icon, sort, created_at, updated_at)
		SELECT 0, '任务队列', '/task-queue', 'icon-history', 65, NOW(), NOW()
		WHERE NOT EXISTS (SELECT 1 FROM menus WHERE path = '/task-queue')`).Error; err != nil {
		log.Fatalf("Failed to seed task queue menu: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_tasks ADD COLUMN IF NOT EXISTS is_saved_to_design boolean NOT NULL DEFAULT false`).Error; err != nil {
		log.Fatalf("Failed to add is_saved_to_design column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_tasks ADD COLUMN IF NOT EXISTS saved_at timestamptz`).Error; err != nil {
		log.Fatalf("Failed to add saved_at column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_tasks ADD COLUMN IF NOT EXISTS reference_image_url varchar(500) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add reference_image_url column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_tasks ADD COLUMN IF NOT EXISTS reference_image_title varchar(255) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add reference_image_title column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_tasks ADD COLUMN IF NOT EXISTS country_code varchar(20) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add country_code column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_tasks ADD COLUMN IF NOT EXISTS country_name varchar(100) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add country_name column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_images ADD COLUMN IF NOT EXISTS candidate_image_url varchar(500) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add candidate_image_url column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_images ADD COLUMN IF NOT EXISTS candidate_status varchar(20) NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add candidate_status column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_images ADD COLUMN IF NOT EXISTS candidate_error_message text NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add candidate_error_message column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_images ADD COLUMN IF NOT EXISTS candidate_prompt text NOT NULL DEFAULT ''`).Error; err != nil {
		log.Fatalf("Failed to add candidate_prompt column: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_generation_images ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now()`).Error; err != nil {
		log.Fatalf("Failed to add scene_generation_images.updated_at column: %v", err)
	}
	if err := DB.Exec(`UPDATE scene_generation_images SET updated_at = created_at WHERE updated_at IS NULL`).Error; err != nil {
		log.Fatalf("Failed to backfill scene_generation_images.updated_at: %v", err)
	}
	if err := DB.Exec(`INSERT INTO settings (key, value, created_at, updated_at)
		SELECT 'gpt_image_base_url', '', NOW(), NOW()
		WHERE NOT EXISTS (SELECT 1 FROM settings WHERE key = 'gpt_image_base_url')`).Error; err != nil {
		log.Fatalf("Failed to seed gpt_image_base_url setting: %v", err)
	}
	log.Println("GORM scene auto migration completed")
}

func runImageTaskAutoMigrate() {
	if err := DB.AutoMigrate(&model.ImageGenerationTask{}); err != nil {
		log.Fatalf("Failed to auto migrate image generation task schema: %v", err)
	}
	if err := DB.Exec(`UPDATE image_generation_tasks SET ref_images_data = '' WHERE ref_images_data IS NOT NULL AND ref_images_data <> ''`).Error; err != nil {
		log.Fatalf("Failed to clear image generation task ref image data: %v", err)
	}
	log.Println("GORM image generation task auto migration completed")
}

func runEcomSuiteAutoMigrate() {
	// Skip AutoMigrate for ecom suite tables - they are managed via SQL migrations
	// to ensure correct constraint naming (PostgreSQL uses <table>_<column>_key format)
	// AutoMigrate would try to reconcile the unique index with GORM's naming convention

	taskDDL := []string{
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS task_id varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS user_id bigint NOT NULL DEFAULT 0`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_id bigint NOT NULL DEFAULT 0`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS workflow_type varchar(30) NOT NULL DEFAULT 'legacy'`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS platform varchar(50) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS language varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS country_code varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS country_name varchar(100) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS output_ratio varchar(20) NOT NULL DEFAULT '1:1'`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS brand varchar(50) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS differentiation_mode varchar(50) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS white_bg_image_url varchar(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_reference_urls jsonb NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS reference_gallery_id bigint NOT NULL DEFAULT 0`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS water_flow_mode varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS feature_mode varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS output_target varchar(20) NOT NULL DEFAULT 'all'`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS include_listing boolean NOT NULL DEFAULT false`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS secondary_count integer NOT NULL DEFAULT 0`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS desktop_a_plus_count integer NOT NULL DEFAULT 0`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS mobile_a_plus_count integer NOT NULL DEFAULT 0`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_name varchar(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_category varchar(200) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_brand varchar(200) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_features text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_proof text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_length varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_width varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_height varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_dimension_unit varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_weight varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS product_weight_unit varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS package_length varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS package_width varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS package_height varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS package_dimension_unit varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS package_weight varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS package_weight_unit varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS package_contents text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS listing_platform_rules text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS listing_quality_rules text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS listing_keywords text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS buyer_questions jsonb NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS visual_style text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS style_requirements text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS style_reference_urls jsonb NOT NULL DEFAULT '[]'::jsonb`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS drawing_model varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS selected_a_plus_template_id bigint`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS use_brand_logo boolean NOT NULL DEFAULT true`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS status varchar(30) NOT NULL DEFAULT 'pending'`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS smart_stage varchar(40) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS current_action text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS error_message text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS plan_snapshot jsonb`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now()`,
		`ALTER TABLE ecom_suite_tasks ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now()`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_ecom_suite_tasks_task_id ON ecom_suite_tasks(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ecom_suite_tasks_user_id ON ecom_suite_tasks(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ecom_suite_tasks_status ON ecom_suite_tasks(status)`,
		`CREATE INDEX IF NOT EXISTS idx_ecom_suite_tasks_created_at ON ecom_suite_tasks(created_at)`,
	}
	for _, ddl := range taskDDL {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Fatalf("Failed to update ecom_suite_tasks schema: %v", err)
		}
	}

	imageDDL := []string{
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS task_id varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS image_type varchar(30) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS image_index integer NOT NULL DEFAULT 1`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS output_ratio varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS title varchar(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS subtitle varchar(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS must_include_text text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS layout_instruction text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS typography_instruction text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS visual_strategy text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS reference_rule jsonb`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS prompt_draft text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS prompt_final text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS primary_color varchar(200) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS secondary_color varchar(200) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS a_plus_template_id bigint`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS include_logo boolean NOT NULL DEFAULT false`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS cta_direction text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS style text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS status varchar(30) NOT NULL DEFAULT 'pending'`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS image_url varchar(1000) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS review_source_image_url varchar(1000) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS error_message text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS candidate_image_url varchar(1000) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS candidate_status varchar(30) NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS candidate_prompt text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS candidate_error_message text NOT NULL DEFAULT ''`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now()`,
		`ALTER TABLE ecom_suite_images ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now()`,
		`CREATE INDEX IF NOT EXISTS idx_ecom_suite_images_task_id ON ecom_suite_images(task_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ecom_suite_images_task_type ON ecom_suite_images(task_id, image_type)`,
	}
	for _, ddl := range imageDDL {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Fatalf("Failed to update ecom_suite_images schema: %v", err)
		}
	}

	conversationDDL := []string{
		`CREATE TABLE IF NOT EXISTS smart_suite_conversations (
			id bigserial PRIMARY KEY,
			session_id varchar(64) NOT NULL,
			user_id bigint NOT NULL,
			task_id varchar(64) NOT NULL DEFAULT '',
			context jsonb NOT NULL DEFAULT '{}'::jsonb,
			messages jsonb NOT NULL DEFAULT '[]'::jsonb,
			created_at timestamptz NOT NULL DEFAULT now(),
			updated_at timestamptz NOT NULL DEFAULT now()
		)`,
		`ALTER TABLE smart_suite_conversations ADD COLUMN IF NOT EXISTS task_ids jsonb NOT NULL DEFAULT '[]'::jsonb`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_smart_suite_conversations_session_id ON smart_suite_conversations(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_smart_suite_conversations_user_id ON smart_suite_conversations(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_smart_suite_conversations_updated_at ON smart_suite_conversations(updated_at)`,
	}
	for _, ddl := range conversationDDL {
		if err := DB.Exec(ddl).Error; err != nil {
			log.Fatalf("Failed to update smart_suite_conversations schema: %v", err)
		}
	}
	if err := DB.Exec(`ALTER TABLE ecom_suite_images ALTER COLUMN role TYPE text`).Error; err != nil {
		log.Fatalf("Failed to alter ecom_suite_images.role type: %v", err)
	}

	if err := DB.Exec(`INSERT INTO menus (parent_id, title, path, icon, sort, created_at, updated_at)
		SELECT 0, '产品套图', '/product-image', 'icon-image', 66, NOW(), NOW()
		WHERE NOT EXISTS (SELECT 1 FROM menus WHERE path = '/product-image')`).Error; err != nil {
		log.Fatalf("Failed to seed product-image menu: %v", err)
	}

	if err := DB.Exec(`DELETE FROM menus WHERE path = '/brand-settings'`).Error; err != nil {
		log.Fatalf("Failed to cleanup brand-settings menu: %v", err)
	}

	log.Println("GORM ecom suite auto migration completed")
}

// migrateBuyerReviewFingerprints recomputes review fingerprints with the
// normalized review link so rows imported with different link spellings
// (query parameters, http vs https, host case, trailing slash) collapse into
// one row. Rows sharing a new fingerprint are merged into the oldest one,
// keeping non-empty fields from later duplicates, and the migration is
// idempotent.
func migrateBuyerReviewFingerprints() error {
	var reviews []model.BuyerReview
	if err := DB.Order("id ASC").Find(&reviews).Error; err != nil {
		return err
	}
	type update struct {
		id          uint64
		fingerprint string
	}
	updates := []update{}
	merged := map[uint64]bool{}
	groups := map[string][]model.BuyerReview{}
	for _, review := range reviews {
		fingerprint := model.BuyerReviewFingerprint(review.Link, review.ChildASIN, review.ReviewDate, review.Name, review.Title, review.Content)
		if fingerprint != review.Fingerprint {
			updates = append(updates, update{id: review.ID, fingerprint: fingerprint})
		}
		groups[fingerprint] = append(groups[fingerprint], review)
	}
	fills := map[uint64]map[string]any{}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		// Oldest row survives; fill its empty fields from later duplicates.
		keeper, fill := group[0], map[string]any{}
		for _, duplicate := range group[1:] {
			merged[duplicate.ID] = true
			if keeper.Rating == 0 && duplicate.Rating != 0 {
				if _, exists := fill["rating"]; !exists {
					fill["rating"] = duplicate.Rating
				}
			}
			if !keeper.Video && duplicate.Video {
				fill["video"] = true
			}
			for _, field := range []struct {
				column string
				value  string
			}{
				{"brand", duplicate.Brand},
				{"country", duplicate.Country},
				{"parent_asin", duplicate.ParentASIN},
				{"child_asin", duplicate.ChildASIN},
				{"name", duplicate.Name},
				{"review_date", duplicate.ReviewDate},
				{"title", duplicate.Title},
				{"content", duplicate.Content},
				{"link", duplicate.Link},
				{"image", duplicate.Image},
				{"review_images", duplicate.ReviewImages},
				{"video_upload", duplicate.VideoUpload},
				{"video_link", duplicate.VideoLink},
			} {
				if field.value != "" {
					if _, exists := fill[field.column]; !exists {
						fill[field.column] = field.value
					}
				}
			}
		}
		if len(fill) > 0 {
			fills[keeper.ID] = fill
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		for id, fill := range fills {
			if err := tx.Model(&model.BuyerReview{}).Where("id = ?", id).Updates(fill).Error; err != nil {
				return err
			}
		}
		if len(updates) > 0 {
			// Detach changed fingerprints first so rewritten values never
			// collide with each other or with untouched rows while the unique
			// index is in flux.
			for _, item := range updates {
				if err := tx.Model(&model.BuyerReview{}).Where("id = ?", item.id).Update("fingerprint", fmt.Sprintf("pending:%d", item.id)).Error; err != nil {
					return err
				}
			}
		}
		// Remove merged duplicates before writing new fingerprints so a
		// survivor can reclaim a fingerprint still held by a duplicate.
		for id := range merged {
			if err := tx.Where("id = ?", id).Delete(&model.BuyerReview{}).Error; err != nil {
				return err
			}
		}
		for _, item := range updates {
			if merged[item.id] {
				continue
			}
			if err := tx.Model(&model.BuyerReview{}).Where("id = ?", item.id).Update("fingerprint", item.fingerprint).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func runReferenceGalleryAutoMigrate() {
	if err := DB.AutoMigrate(&model.ReferenceGallery{}); err != nil {
		log.Fatalf("Failed to auto migrate reference gallery schema: %v", err)
	}
	if err := DB.Exec(`
		INSERT INTO menus (parent_id, title, path, icon, sort, created_at, updated_at)
		SELECT parent.id, '参考图库', '/reference-galleries', 'icon-image', 3, NOW(), NOW()
		FROM menus parent
		WHERE parent.title = '设计素材'
			AND parent.parent_id = 0
			AND NOT EXISTS (SELECT 1 FROM menus WHERE path = '/reference-galleries')
	`).Error; err != nil {
		log.Fatalf("Failed to seed reference gallery menu: %v", err)
	}
	log.Println("GORM reference gallery auto migration completed")
}

func runSceneStyleCategoryAutoMigrate() {
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS scene_style_categories (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			icon VARCHAR(50) NOT NULL DEFAULT '',
			sort INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		log.Fatalf("Failed to create scene_style_categories table: %v", err)
	}
	if err := DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_scene_style_categories_name ON scene_style_categories(name)`).Error; err != nil {
		log.Fatalf("Failed to create idx_scene_style_categories_name: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scene_style_categories_sort ON scene_style_categories(sort)`).Error; err != nil {
		log.Fatalf("Failed to create idx_scene_style_categories_sort: %v", err)
	}
	if err := DB.Exec(`ALTER TABLE scene_style_tags ADD COLUMN IF NOT EXISTS scene_id INT NOT NULL DEFAULT 0`).Error; err != nil {
		log.Fatalf("Failed to add scene_style_tags.scene_id column: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scene_style_tags_scene_id ON scene_style_tags(scene_id)`).Error; err != nil {
		log.Fatalf("Failed to create idx_scene_style_tags_scene_id: %v", err)
	}

	// 迁移：回填 scene_style_references.scene_id（确保所有参考图有 scene_id）
	refsUpdated := int64(0)
	DB.Model(&model.SceneStyleReference{}).Where("scene_id = 0").Count(&refsUpdated)
	log.Printf("Found %d scene_style_references with scene_id=0 (these will use default)", refsUpdated)
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scene_style_references_scene_tag_id ON scene_style_references(scene_id, tag_id, id)`).Error; err != nil {
		log.Fatalf("Failed to create idx_scene_style_references_scene_tag_id: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_scene_style_references_tag_scene_id ON scene_style_references(tag_id, scene_id, id)`).Error; err != nil {
		log.Fatalf("Failed to create idx_scene_style_references_tag_scene_id: %v", err)
	}

	log.Println("Scene style category auto migration completed")
}

func runProductColorFamilyAutoMigrate() {
	if err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS product_color_families (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			image_url VARCHAR(500) NOT NULL DEFAULT '',
			sort INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error; err != nil {
		log.Fatalf("Failed to create product_color_families table: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_product_color_families_sort ON product_color_families(sort)`).Error; err != nil {
		log.Fatalf("Failed to create idx_product_color_families_sort: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_product_color_families_created_at ON product_color_families(created_at)`).Error; err != nil {
		log.Fatalf("Failed to create idx_product_color_families_created_at: %v", err)
	}
	log.Println("Product color family auto migration completed")
}

func runDepartmentAutoMigrate() {
	if err := DB.AutoMigrate(&model.Department{}); err != nil {
		log.Fatalf("Failed to auto migrate Department schema: %v", err)
	}
	if err := DB.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_platform_name_sub_name ON departments(platform, name, sub_name)`).Error; err != nil {
		log.Fatalf("Failed to create idx_departments_platform_name_sub_name: %v", err)
	}
	if err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_departments_sort ON departments(sort, id)`).Error; err != nil {
		log.Fatalf("Failed to create idx_departments_sort: %v", err)
	}
	if err := DB.Exec(`INSERT INTO menus (parent_id, title, path, icon, sort, created_at, updated_at)
		SELECT 0, '部门管理', '/departments', 'icon-user', 26, NOW(), NOW()
		WHERE NOT EXISTS (SELECT 1 FROM menus WHERE path = '/departments')`).Error; err != nil {
		log.Fatalf("Failed to seed department menu: %v", err)
	}
	log.Println("Department auto migration completed")
}

func runWhiteBgAutoMigrate() {
	if err := DB.AutoMigrate(&model.WhiteBgTask{}); err != nil {
		log.Fatalf("Failed to auto migrate WhiteBgTask: %v", err)
	}
	log.Println("WhiteBgTask auto migration completed")
}

func runMultiAngleAutoMigrate() {
	if err := DB.AutoMigrate(&model.MultiAngleTask{}, &model.MultiAngleResult{}); err != nil {
		log.Fatalf("Failed to auto migrate MultiAngle schema: %v", err)
	}
	log.Println("MultiAngle auto migration completed")
}

func runDepthAutoMigrate() {
	if err := DB.AutoMigrate(&model.DepthTask{}, &model.DepthImage{}); err != nil {
		log.Fatalf("Failed to auto migrate Depth schema: %v", err)
	}
	log.Println("Depth auto migration completed")
}

func runLightShadowAutoMigrate() {
	if err := DB.AutoMigrate(&model.LightShadowTask{}, &model.LightShadowImage{}); err != nil {
		log.Fatalf("Failed to auto migrate LightShadow schema: %v", err)
	}
	log.Println("LightShadow auto migration completed")
}

func runImageBatchAutoMigrate() {
	if err := DB.AutoMigrate(&model.ImageBatch{}, &model.ImageBatchItem{}); err != nil {
		log.Fatalf("Failed to auto migrate ImageBatch schema: %v", err)
	}

	// Widen image_batches.title/subtitle/cover from varchar(500) to text so
	// long user prompts (used as batch title) don't trigger SQLSTATE 22001.
	// Idempotent: ALTER COLUMN ... TYPE text is a no-op when already text.
	alterStmts := []string{
		`ALTER TABLE image_batches ALTER COLUMN title TYPE text`,
		`ALTER TABLE image_batches ALTER COLUMN subtitle TYPE text`,
		`ALTER TABLE image_batches ALTER COLUMN cover TYPE text`,
	}
	for _, stmt := range alterStmts {
		if err := DB.Exec(stmt).Error; err != nil {
			log.Fatalf("Failed to widen image_batches column: %v (stmt: %s)", err, stmt)
		}
	}

	log.Println("ImageBatch auto migration completed")
}

func runWorkflowProjectAutoMigrate() {
	if err := DB.AutoMigrate(&model.WorkflowProject{}, &model.WorkflowTemplate{}); err != nil {
		log.Fatalf("Failed to auto migrate WorkflowProject schema: %v", err)
	}
	log.Println("WorkflowProject auto migration completed")
}

func runProductCatalogAutoMigrate() {
	if err := DB.AutoMigrate(&model.ProductSPU{}, &model.ProductSPUAttachment{}, &model.ProductSKU{}, &model.TikTokSKUPrice{}, &model.ProductPriceHistory{}); err != nil {
		log.Fatalf("Failed to auto migrate product catalog schema: %v", err)
	}

	ddl := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_product_spus_spu ON product_spus(spu)`,
		`CREATE INDEX IF NOT EXISTS idx_product_spus_created_at ON product_spus(created_at DESC)`,
		`ALTER TABLE product_spus ADD COLUMN IF NOT EXISTS name varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_spus ADD COLUMN IF NOT EXISTS brand varchar(120) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_spus ADD COLUMN IF NOT EXISTS model varchar(120) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_spus ADD COLUMN IF NOT EXISTS material varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_spus ADD COLUMN IF NOT EXISTS category1 varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_spus ADD COLUMN IF NOT EXISTS category2 varchar(255) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_spu_attachments ADD COLUMN IF NOT EXISTS spu_id bigint NOT NULL DEFAULT 0`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS spu_id bigint NOT NULL DEFAULT 0`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS asin varchar(20) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS stp_model_url varchar(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS auth_doc_url varchar(500) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS listed_at varchar(32) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS package_length varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS package_width varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS package_height varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS package_weight varchar(64) NOT NULL DEFAULT ''`,
		`ALTER TABLE product_skus ADD COLUMN IF NOT EXISTS purchase_price varchar(64) NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_product_spu_attachments_spu_id_sort ON product_spu_attachments(spu_id, sort, id)`,
		`CREATE INDEX IF NOT EXISTS idx_product_skus_spu_id ON product_skus(spu_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_product_skus_sku ON product_skus(sku)`,
		`CREATE INDEX IF NOT EXISTS idx_product_skus_asin ON product_skus(asin)`,
		`CREATE INDEX IF NOT EXISTS idx_product_skus_spu ON product_skus(spu)`,
		`CREATE INDEX IF NOT EXISTS idx_product_skus_style_name ON product_skus(style_name)`,
	}
	for _, stmt := range ddl {
		if err := DB.Exec(stmt).Error; err != nil {
			log.Fatalf("Failed to update product catalog schema: %v (stmt=%s)", err, stmt)
		}
	}

	backfillProductSPUs()
	log.Println("Product catalog auto migration completed")
}

func runAssetAutoMigrate() {
	if err := DB.AutoMigrate(&model.Asset{}, &model.AssetFavorite{}, &model.AssetCategory{}); err != nil {
		log.Fatalf("Failed to auto migrate asset schema: %v", err)
	}

	ddl := []string{
		`CREATE INDEX IF NOT EXISTS idx_assets_owner_type ON assets(owner_type)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_owner_id ON assets(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_user_id ON assets(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_department ON assets(department)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_visibility ON assets(visibility)`,
		`CREATE INDEX IF NOT EXISTS idx_assets_updated_at ON assets(updated_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_favorites_unique ON asset_favorites(user_id, source_type, source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_asset_favorites_user_id ON asset_favorites(user_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_asset_categories_unique ON asset_categories(owner_type, owner_id, department, type, name)`,
		`CREATE INDEX IF NOT EXISTS idx_asset_categories_owner_type ON asset_categories(owner_type)`,
		`CREATE INDEX IF NOT EXISTS idx_asset_categories_user_id ON asset_categories(user_id)`,
	}
	for _, stmt := range ddl {
		if err := DB.Exec(stmt).Error; err != nil {
			log.Fatalf("Failed to update asset schema: %v (stmt=%s)", err, stmt)
		}
	}
	log.Println("Asset auto migration completed")
}

func backfillProductSPUs() {
	var rows []struct {
		SPU string `gorm:"column:spu"`
	}
	if err := DB.Model(&model.ProductSKU{}).Distinct("spu").Where("spu <> ''").Order("spu asc").Find(&rows).Error; err != nil {
		log.Printf("Warning: failed to load distinct SKU SPUs: %v", err)
		return
	}

	for _, item := range rows {
		spuName := strings.TrimSpace(item.SPU)
		if spuName == "" {
			continue
		}

		var spu model.ProductSPU
		err := DB.Where("spu = ?", spuName).First(&spu).Error
		if err != nil {
			spu = model.ProductSPU{SPU: spuName}
			if err := DB.Create(&spu).Error; err != nil {
				log.Printf("Warning: failed to create product spu %q: %v", spuName, err)
				continue
			}
		}

		if err := DB.Model(&model.ProductSKU{}).Where("spu = ? AND spu_id = 0", spuName).Update("spu_id", spu.ID).Error; err != nil {
			log.Printf("Warning: failed to backfill product sku spu_id for %q: %v", spuName, err)
		}

		var legacySKU model.ProductSKU
		if err := DB.Where("spu = ?", spuName).Order("id asc").First(&legacySKU).Error; err == nil {
			updates := map[string]interface{}{}
			if spu.Name == "" && legacySKU.Name != "" {
				updates["name"] = legacySKU.Name
			}
			if spu.Material == "" && legacySKU.Material != "" {
				updates["material"] = legacySKU.Material
			}
			if spu.Category1 == "" && legacySKU.Category1 != "" {
				updates["category1"] = legacySKU.Category1
			}
			if spu.Category2 == "" && legacySKU.Category2 != "" {
				updates["category2"] = legacySKU.Category2
			}
			if len(updates) > 0 {
				if err := DB.Model(&spu).Updates(updates).Error; err != nil {
					log.Printf("Warning: failed to backfill product spu fields for %q: %v", spuName, err)
				}
			}
		}

		if err := DB.Where("spu = ? AND stp_model_url <> ''", spuName).Order("id asc").First(&legacySKU).Error; err == nil && spu.ModelURL == "" {
			spu.ModelURL = legacySKU.StpModelURL
			_ = DB.Save(&spu).Error
		}

		var legacyAuth model.ProductSKU
		if err := DB.Where("spu = ? AND auth_doc_url <> ''", spuName).Order("id asc").First(&legacyAuth).Error; err == nil {
			var exists int64
			DB.Model(&model.ProductSPUAttachment{}).Where("spu_id = ?", spu.ID).Count(&exists)
			if exists == 0 {
				attachment := model.ProductSPUAttachment{
					SPUID:    spu.ID,
					Label:    "认证资料",
					FileURL:  legacyAuth.AuthDocURL,
					FileName: filepath.Base(legacyAuth.AuthDocURL),
					FileType: detectAttachmentFileTypeFromURL(legacyAuth.AuthDocURL),
				}
				if err := DB.Create(&attachment).Error; err != nil {
					log.Printf("Warning: failed to backfill attachment for %q: %v", spuName, err)
				}
			}
		}
	}
}

func detectAttachmentFileTypeFromURL(url string) string {
	ext := strings.ToLower(filepath.Ext(url))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return "image"
	case ".pdf":
		return "pdf"
	default:
		return "file"
	}
}
