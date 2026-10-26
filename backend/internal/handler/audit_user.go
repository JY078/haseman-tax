package handler

import (
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
)

func loadUserDisplayNames(userIDs []uint) map[uint]string {
	uniqueIDs := make([]uint, 0, len(userIDs))
	seen := make(map[uint]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}

	names := make(map[uint]string, len(uniqueIDs))
	if len(uniqueIDs) == 0 {
		return names
	}

	var users []model.User
	if err := database.DB.Select("id", "name", "username").Where("id IN ?", uniqueIDs).Find(&users).Error; err != nil {
		return names
	}
	for _, user := range users {
		name := strings.TrimSpace(user.Name)
		if name == "" {
			name = strings.TrimSpace(user.Username)
		}
		names[user.ID] = name
	}
	return names
}
