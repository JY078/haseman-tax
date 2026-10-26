package handler

import (
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/cache"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUserOnlineLogoutIntegration(t *testing.T) {
	dsn := os.Getenv("AUTH_TEST_DSN")
	if dsn == "" {
		t.Skip("set AUTH_TEST_DSN for isolated PostgreSQL integration")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("integration database must be local")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	oldDB, oldRedis := database.DB, cache.RDB
	database.DB, cache.RDB = tx, nil
	t.Cleanup(func() { tx.Rollback(); database.DB, cache.RDB = oldDB, oldRedis })
	schema := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Exec("CREATE TABLE users (id BIGINT PRIMARY KEY, enabled BOOLEAN NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("INSERT INTO users VALUES (1, TRUE), (2, TRUE), (3, FALSE)").Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.UserSession{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	staleHeartbeat := now.Add(-24 * time.Hour)
	for _, session := range []model.UserSession{
		{UserID: 1, TokenHash: utils.HashToken("web-token"), ClientType: "web", ExpiresAt: now.Add(time.Hour), LastActiveAt: &staleHeartbeat},
		{UserID: 1, TokenHash: utils.HashToken("admin-token"), ClientType: "admin", ExpiresAt: now.Add(time.Hour), LastActiveAt: &now},
		{UserID: 1, TokenHash: utils.HashToken("expired-token"), ExpiresAt: now.Add(-time.Hour), LastActiveAt: &now},
		{UserID: 2, TokenHash: utils.HashToken("no-heartbeat"), ExpiresAt: now.Add(time.Hour)},
		{UserID: 3, TokenHash: utils.HashToken("disabled-token"), ExpiresAt: now.Add(time.Hour), LastActiveAt: &now},
	} {
		if err := tx.Create(&session).Error; err != nil {
			t.Fatal(err)
		}
	}
	users := []model.User{{ID: 1, Enabled: true}, {ID: 2, Enabled: true}, {ID: 3}}
	checkOnline := func(want bool) {
		t.Helper()
		if err := loadUserOnlineStatus(users); err != nil {
			t.Fatal(err)
		}
		if users[0].Online != want || users[1].Online || users[2].Online {
			t.Fatalf("online states = %v, %v; want %v, false", users[0].Online, users[1].Online, want)
		}
	}
	r := gin.New()
	r.POST("/logout", Logout)
	r.POST("/heartbeat", Heartbeat)
	heartbeat := func(token string, status int) {
		t.Helper()
		req := httptest.NewRequest("POST", "/heartbeat", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("heartbeat status %d, want %d: %s", w.Code, status, w.Body.String())
		}
	}
	logout := func(token string) {
		t.Helper()
		req := httptest.NewRequest("POST", "/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("logout status %d: %s", w.Code, w.Body.String())
		}
	}
	checkOnline(true)
	logout("admin-token")
	checkOnline(false) // A stale heartbeat cannot keep an unexpired login online.
	heartbeat("web-token", 200)
	checkOnline(true)
	var refreshed model.UserSession
	if err := tx.Where("token_hash = ?", utils.HashToken("web-token")).First(&refreshed).Error; err != nil {
		t.Fatal(err)
	}
	if refreshed.ExpiresAt.Sub(now.Add(time.Hour)).Abs() > time.Millisecond {
		t.Fatal("heartbeat extended the login expiry")
	}
	logout("web-token")
	checkOnline(false) // The expired session must not keep the account online.
	logout("web-token")
	checkOnline(false)
	for _, token := range []string{"web-token", "expired-token", "disabled-token"} {
		heartbeat(token, 401)
	}
}
