package seeder

import (
	"errors"
	"log"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func SeedRolesAndAdmin() {
	db := database.DB

	// 1. Ensure Roles Exist
	roles := []string{"admin", "user"}
	for _, roleName := range roles {
		var role model.Role
		if err := db.FirstOrCreate(&role, model.Role{Name: roleName}).Error; err != nil {
			log.Printf("Failed to seed role %s: %v", roleName, err)
		} else {
			// Update description if needed, or log success
			// log.Printf("Role %s ensured", roleName)
		}
	}

	// 2. Ensure Admin User Exists
	adminEmail := "admin@example.com"
	var adminUser model.User
	result := db.Where("email = ?", adminEmail).First(&adminUser)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Admin does not exist, create it
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash password: %v", err)
			}

			newAdmin := model.User{
				Username: "admin",
				Email:    adminEmail,
				Password: string(hashedPassword),
			}

			if err := db.Create(&newAdmin).Error; err != nil {
				log.Printf("Failed to create admin user: %v", err)
				return
			}

			// Assign admin role
			var adminRole model.Role
			if err := db.Where("name = ?", "admin").First(&adminRole).Error; err == nil {
				if err := db.Model(&newAdmin).Association("Roles").Append(&adminRole); err != nil {
					log.Printf("Failed to assign admin role: %v", err)
				}
			}

			log.Println("Admin user created successfully")
		} else {
			log.Printf("Error checking for admin user: %v", result.Error)
		}
	} else {
		// Admin exists, ensure role is assigned (optional, but good for consistency)
		var adminRole model.Role
		if err := db.Where("name = ?", "admin").First(&adminRole).Error; err == nil {
			// Check existence
			associationCount := db.Model(&adminUser).Where("id = ?", adminRole.ID).Association("Roles").Count()
			if associationCount == 0 {
				if err := db.Model(&adminUser).Association("Roles").Append(&adminRole); err != nil {
					// Ignore duplicate key error if concurrent runs happen
					log.Printf("Failed to assign admin role to existing admin: %v", err)
				} else {
					log.Println("Admin role assigned to existing admin user")
				}
			}
		}
		log.Println("Admin user already exists, skipping creation")
	}

	// 3. Seed Menus
	seedMenus(db)

	// 4. Assign all menus to admin role
	seedAdminRoleMenus(db)

	// 5. Migrate Settings to Models
	MigrateSettingsToModels(db)
}

func seedMenus(db *gorm.DB) {
	menus := []model.Menu{
		{Title: "仪表盘", Path: "/dashboard", Icon: "icon-dashboard", Sort: 1},
		{Title: "模板库", Path: "/prompts", Icon: "icon-file", Sort: 2},
		{Title: "产品库", Path: "/product-prompts", Icon: "icon-image", Sort: 3},
		{Title: "设计素材", Path: "", Icon: "icon-book", Sort: 4},
		{Title: "使用记录", Path: "/usage", Icon: "icon-history", Sort: 8},
		{Title: "图片统计", Path: "/image-stats", Icon: "icon-bar-chart", Sort: 9},
		{Title: "角色管理", Path: "/roles", Icon: "icon-safe", Sort: 10},
		{Title: "模型管理", Path: "/models", Icon: "icon-robot", Sort: 11},
		{Title: "公告管理", Path: "/announcements", Icon: "icon-notification", Sort: 14},
		{Title: "意见反馈", Path: "/feedbacks", Icon: "icon-message", Sort: 15},
		{Title: "菜单管理", Path: "/menus", Icon: "icon-menu", Sort: 16},
		{Title: "用户管理", Path: "/users", Icon: "icon-user", Sort: 17},
		{Title: "部门管理", Path: "/departments", Icon: "icon-user", Sort: 18},
		{Title: "系统设置", Path: "/settings", Icon: "icon-settings", Sort: 19},
	}

	for _, menu := range menus {
		var existingMenu model.Menu
		var err error
		if menu.Path != "" {
			err = db.Where("path = ?", menu.Path).First(&existingMenu).Error
		} else {
			// For menus with empty path (parent categories), find by title and parent_id = 0
			err = db.Where("title = ? AND parent_id = 0", menu.Title).First(&existingMenu).Error
		}
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Create(&menu).Error; err != nil {
					log.Printf("Failed to seed menu %s: %v", menu.Title, err)
				} else {
					log.Printf("Menu %s created", menu.Title)
				}
			}
		} else {
			/** 若菜单已存在，同步更新 title 和 icon，保证数据与代码一致 */
			updates := map[string]interface{}{
				"title": menu.Title,
				"icon":  menu.Icon,
				"sort":  menu.Sort,
			}
			/** 如果是父菜单（空路径），也要同步清空路径 */
			if menu.Path == "" {
				updates["path"] = ""
			}
			if err := db.Model(&existingMenu).Updates(updates).Error; err != nil {
				log.Printf("Failed to update menu %s: %v", menu.Title, err)
			}
		}
	}

	// Seed child menus under "风格库"
	seedChildMenus(db)
}

func seedChildMenus(db *gorm.DB) {
	// Find parent menu ID for "设计素材"
	var styleMenu model.Menu
	if err := db.Where("title = ? AND parent_id = 0", "设计素材").First(&styleMenu).Error; err != nil {
		log.Printf("Parent menu '设计素材' not found, skipping child menu creation")
		return
	}

	// Add "场景风格" as child of "风格库"
	childMenu1 := model.Menu{
		Title:    "场景风格",
		Path:     "/scene-styles",
		Icon:     "icon-book",
		Sort:     1,
		ParentID: styleMenu.ID,
	}
	var existing1 model.Menu
	if err := db.Where("path = ? AND parent_id = ?", childMenu1.Path, childMenu1.ParentID).First(&existing1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&childMenu1).Error; err != nil {
				log.Printf("Failed to seed child menu %s: %v", childMenu1.Title, err)
			} else {
				log.Printf("Child menu %s created under 风格库", childMenu1.Title)
			}
		}
	}

	childMenus := []model.Menu{
		{Title: "色系库", Path: "/product-colors", Icon: "icon-palette", Sort: 2, ParentID: styleMenu.ID},
		{Title: "参考图库", Path: "/reference-galleries", Icon: "icon-image", Sort: 3, ParentID: styleMenu.ID},
	}
	for _, childMenu := range childMenus {
		var existing model.Menu
		if err := db.Where("path = ? AND parent_id = ?", childMenu.Path, childMenu.ParentID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Create(&childMenu).Error; err != nil {
					log.Printf("Failed to seed child menu %s: %v", childMenu.Title, err)
				} else {
					log.Printf("Child menu %s created under 设计素材", childMenu.Title)
				}
			}
		}
	}
}

// seedAdminRoleMenus assigns all menus to the admin role (idempotent)
func seedAdminRoleMenus(db *gorm.DB) {
	var adminRole model.Role
	if err := db.Preload("Menus").Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		log.Printf("Admin role not found, skipping menu assignment: %v", err)
		return
	}

	var allMenus []model.Menu
	db.Find(&allMenus)
	if len(allMenus) == 0 {
		return
	}

	// Always sync all menus to admin role to include any newly added child menus
	if err := db.Model(&adminRole).Association("Menus").Replace(allMenus); err != nil {
		log.Printf("Failed to assign menus to admin role: %v", err)
	} else {
		log.Printf("Assigned %d menus to admin role", len(allMenus))
	}
}
