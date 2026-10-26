package seeder

import (
	"encoding/json"
	"log"

	"advertising-image/backend/internal/model"

	"gorm.io/gorm"
)

func MigrateSettingsToModels(db *gorm.DB) {
	// Check if migration has already been done to avoid re-inserting deleted models
	var flag model.Setting
	if err := db.Where("key = ?", "models_migrated").First(&flag).Error; err == nil {
		// Already migrated, skip
		return
	}

	migrateType(db, "chat_models", "chat")
	migrateType(db, "drawing_models", "drawing")
	migrateType(db, "video_models", "video")

	// Mark as migrated so it won't run again
	db.Create(&model.Setting{Key: "models_migrated", Value: "1"})
}

func migrateType(db *gorm.DB, settingKey, modelType string) {
	var setting model.Setting
	if err := db.Where("key = ?", settingKey).First(&setting).Error; err == nil {
		var modelNames []string
		if err := json.Unmarshal([]byte(setting.Value), &modelNames); err == nil {
			for _, name := range modelNames {
				if name == "" {
					continue
				}

				var existingModel model.AiModel
				if err := db.Where("name = ?", name).First(&existingModel).Error; err == nil {
					// Model exists, append type if not present
					hasType := false
					for _, t := range existingModel.Types {
						if t == modelType {
							hasType = true
							break
						}
					}
					if !hasType {
						existingModel.Types = append(existingModel.Types, modelType)
						if err := db.Save(&existingModel).Error; err != nil {
							log.Printf("Failed to update model %s types: %v", name, err)
						} else {
							log.Printf("Updated model %s types: %v", name, existingModel.Types)
						}
					}
				} else {
					// Create new model
					aiModel := model.AiModel{
						Name:     name,
						Types:    model.StringArray{modelType},
						IsActive: true,
					}
					if err := db.Create(&aiModel).Error; err != nil {
						log.Printf("Failed to migrate model %s: %v", name, err)
					} else {
						log.Printf("Migrated model %s (%s)", name, modelType)
					}
				}
			}
		}
	}
}
