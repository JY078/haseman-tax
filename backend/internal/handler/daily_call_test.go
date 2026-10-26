package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDailyCallDepartmentPermissions(t *testing.T) {
	for _, tt := range []struct {
		name, department, subDepartment string
		admin, found, register, review  bool
	}{
		{"overseas", "海外客服", "", false, true, true, true},
		{"reviewer", "客服", "", false, true, true, true},
		{"department suffix", "海外客服部", "", false, true, true, true},
		{"overseas child", "客服", "海外客服", false, true, true, true},
		{"review child", "运营", "客服部", false, true, true, true},
		{"ordinary child inherits", "海外客服", "一组", false, true, true, true},
		{"no substring match", "客服运营", "", false, true, false, false},
		{"removed department", "海外客服", "", false, false, false, false},
		{"unrelated department", "运营", "", false, true, false, false},
		{"admin without department", "", "", true, false, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			user := model.User{}
			if tt.admin {
				user.Roles = []model.Role{{Name: "admin"}}
			}
			got := dailyCallDepartmentPermissions(user, model.Department{Name: tt.department, SubName: tt.subDepartment}, tt.found)
			if got.CanRegister != tt.register || got.CanReview != tt.review {
				t.Fatalf("permissions = %+v", got)
			}
		})
	}
}

func TestDailyCallRejectsForgedFields(t *testing.T) {
	for _, field := range []string{"quality_result", "reason", "created_at", "created_by_id", "created_by_name", "reviewed_by_id", "registration_updated_at"} {
		t.Run(field, func(t *testing.T) {
			var input dailyCallRegistrationInput
			if err := decodeDailyCallInput(strings.NewReader(fmt.Sprintf(`{"name":"Customer","%s":"forged"}`, field)), &input); err == nil {
				t.Fatal("forged field accepted")
			}
		})
	}
	for _, body := range []string{`{"name":"Customer"}`, `{"created_at":"2020-01-01"}`, `{} {}`} {
		var input dailyCallReviewInput
		if err := decodeDailyCallInput(strings.NewReader(body), &input); err == nil {
			t.Fatalf("invalid review accepted: %s", body)
		}
	}
}

func TestDailyCallLanguage(t *testing.T) {
	for _, tt := range []struct{ department, subDepartment, want string }{
		{"海外客服", "", "en"},
		{"客服", "海外客服部", "en"},
		{"海外客服", "客服", "zh"},
		{"海外客服部", "一组", "en"},
		{"客服", "", "zh"},
		{"运营", "", "zh"},
		{"", "", "zh"},
	} {
		user := model.User{Department: tt.department, SubDepartment: tt.subDepartment, Roles: []model.Role{{Name: "admin"}}}
		if got := dailyCallLanguage(user); got != tt.want {
			t.Fatalf("department=%q subdepartment=%q: got %s want %s", tt.department, tt.subDepartment, got, tt.want)
		}
	}
}

// The optional integration test uses an isolated schema inside a rolled-back transaction.
func TestDailyCallIntegration(t *testing.T) {
	envFile := os.Getenv("DAILY_CALL_TEST_ENV_FILE")
	if envFile == "" {
		t.Skip("set DAILY_CALL_TEST_ENV_FILE to a local PostgreSQL .env file")
	}
	settings := viper.New()
	settings.SetConfigFile(envFile)
	if err := settings.ReadInConfig(); err != nil {
		t.Fatal("cannot read integration database configuration")
	}
	host := settings.GetString("DB_HOST")
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatal("integration test requires a local database")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(settings.GetString("DB_USER"), settings.GetString("DB_PASSWORD")), Host: host + ":" + settings.GetString("DB_PORT"), Path: "/" + settings.GetString("DB_NAME")}
	query := url.Values{"sslmode": {settings.GetString("DB_SSLMODE")}}
	dsn.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot connect to local integration database")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	previous := database.DB
	database.DB = tx
	t.Cleanup(func() { tx.Rollback(); database.DB = previous })
	schema := fmt.Sprintf("daily_call_test_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.User{}, &model.Department{}, &model.DailyCall{}); err != nil {
		t.Fatal(err)
	}
	departments := []model.Department{{Platform: "amazon", Name: "海外客服"}, {Platform: "amazon", Name: "客服"}, {Platform: "amazon", Name: "运营"}}
	if err := tx.Create(&departments).Error; err != nil {
		t.Fatal(err)
	}
	users := []model.User{
		{Username: "overseas", Name: "Overseas Agent", Email: "overseas@example.test", Platform: "amazon", Department: "海外客服"},
		{Username: "reviewer", Name: "Review Agent", Email: "reviewer@example.test", Platform: "amazon", Department: "客服"},
		{Username: "admin", Name: "Administrator", Email: "admin@example.test", Roles: []model.Role{{Name: "admin"}}},
		{Username: "other", Email: "other@example.test", Platform: "amazon", Department: "运营"},
	}
	for i := range users {
		if err := tx.Create(&users[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if id, _ := strconv.Atoi(c.GetHeader("X-Test-User")); id > 0 {
			c.Set("userID", uint(id))
		}
		c.Next()
	})
	router.GET("/calls", ListDailyCalls)
	router.POST("/calls", CreateDailyCall)
	router.GET("/calls/:id", GetDailyCall)
	router.PUT("/calls/:id/registration", UpdateDailyCallRegistration)
	router.PUT("/calls/:id/review", UpdateDailyCallReview)
	send := func(user uint, method, path, body string, status int) []byte {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", strconv.Itoa(int(user)))
		router.ServeHTTP(recorder, req)
		if recorder.Code != status {
			t.Fatalf("%s %s user=%d: status=%d want=%d body=%s", method, path, user, recorder.Code, status, recorder.Body.String())
		}
		return recorder.Body.Bytes()
	}
	registration := `{"name":"Customer","order_number":"ORDER-123","email":"customer@example.test","phone_no":"+1 555 0100","bought_in":"Amazon","country":"US","brief_of_issue":"Missing part"}`
	review := `{"quality_result":"需要补发","reason":"缺少安装配件"}`
	send(0, http.MethodGet, "/calls", "", 401)
	send(users[3].ID, http.MethodGet, "/calls", "", 403)
	for _, actor := range users[:2] {
		var outbound model.DailyCall
		if err := json.Unmarshal(send(actor.ID, http.MethodPost, "/calls?kind=outbound", registration, 201), &outbound); err != nil { t.Fatal(err) }
		outboundPath := fmt.Sprintf("/calls/%d", outbound.ID)
		for _, editor := range users[:2] {
			send(editor.ID, http.MethodPut, outboundPath+"/registration?kind=outbound", registration, 200)
			send(editor.ID, http.MethodPut, outboundPath+"/review?kind=outbound", `{"information_requested":"Photo","result":"Called"}`, 200)
			send(editor.ID, http.MethodGet, outboundPath+"?kind=outbound", "", 200)
		}
		send(actor.ID, http.MethodPut, outboundPath+"/review", review, 404)
	}
	send(users[1].ID, http.MethodPost, "/calls?kind=outbound", registration, 201)
	send(users[0].ID, http.MethodPost, "/calls?kind=outbound", `{"name":" ","brief_of_issue":"test"}`, 201)
	send(users[0].ID, http.MethodPost, "/calls", strings.TrimSuffix(registration, "}")+`,"created_by_id":999}`, 400)
	before := time.Now().Truncate(time.Microsecond)
	createdJSON := send(users[0].ID, http.MethodPost, "/calls", registration, 201)
	var created model.DailyCall
	if err := json.Unmarshal(createdJSON, &created); err != nil {
		t.Fatal(err)
	}
	if created.CreatedAt.Before(before) || created.CreatedAt.After(time.Now()) || created.CreatedByID != users[0].ID || created.CreatedByName != users[0].Name {
		t.Fatalf("incorrect creation provenance: %+v", created)
	}
	path := fmt.Sprintf("/calls/%d", created.ID)
	send(users[0].ID, http.MethodPut, path+"/review", review, 200)
	send(users[1].ID, http.MethodPut, path+"/registration", registration, 200)
	send(users[1].ID, http.MethodPut, path+"/review", `{"quality_result":" ","reason":"test"}`, 200)
	send(users[1].ID, http.MethodPut, path+"/review", review, 200)
	send(users[0].ID, http.MethodPut, path+"/registration", strings.Replace(registration, "Missing part", "Missing two parts", 1), 200)
	var detail struct {
		Data model.DailyCall `json:"data"`
	}
	if err := json.Unmarshal(send(users[1].ID, http.MethodGet, path, "", 200), &detail); err != nil {
		t.Fatal(err)
	}
	if !detail.Data.CreatedAt.Equal(created.CreatedAt) || detail.Data.CreatedByID != users[0].ID || detail.Data.ReviewedByID == nil || *detail.Data.ReviewedByID != users[1].ID || detail.Data.ReviewedByName != users[1].Name || detail.Data.QualityResult != "需要补发" || detail.Data.BriefOfIssue != "Missing two parts" {
		t.Fatalf("section update overwrote provenance or other section: %+v", detail.Data)
	}
	beforeNoOp := detail.Data
	unchangedRegistration := strings.Replace(registration, "Missing part", "Missing two parts", 1)
	unchangedRegistration = strings.Replace(unchangedRegistration, `"Customer"`, `" Customer "`, 1)
	for _, request := range []struct{ section, body string }{
		{"registration", unchangedRegistration}, {"review", review},
	} {
		var result struct {
			Changed bool `json:"changed"`
		}
		if err := json.Unmarshal(send(users[2].ID, http.MethodPut, path+"/"+request.section, request.body, 200), &result); err != nil {
			t.Fatal(err)
		}
		if result.Changed {
			t.Fatalf("identical %s save reported a change", request.section)
		}
	}
	if err := json.Unmarshal(send(users[1].ID, http.MethodGet, path, "", 200), &detail); err != nil {
		t.Fatal(err)
	}
	if !detail.Data.UpdatedAt.Equal(beforeNoOp.UpdatedAt) || !detail.Data.RegistrationUpdatedAt.Equal(beforeNoOp.RegistrationUpdatedAt) || !detail.Data.ReviewedAt.Equal(*beforeNoOp.ReviewedAt) || detail.Data.RegistrationUpdatedByID != beforeNoOp.RegistrationUpdatedByID || *detail.Data.ReviewedByID != *beforeNoOp.ReviewedByID {
		t.Fatal("identical save changed operation timestamps or contributors")
	}
	send(users[2].ID, http.MethodPost, "/calls", registration, 201)
	send(users[2].ID, http.MethodPut, path+"/registration", registration, 200)
	if err := json.Unmarshal(send(users[1].ID, http.MethodGet, path, "", 200), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Data.RegistrationUpdatedByID != users[2].ID || detail.Data.BriefOfIssue != "Missing part" {
		t.Fatal("actual edit did not update contributor and content")
	}
	send(users[2].ID, http.MethodPut, path+"/review", review, 200)
	for _, filter := range []string{"status=pending", "status=reviewed", "keyword=ORDER-123&status=reviewed"} {
		var list struct {
			Total int               `json:"total"`
			Data  []model.DailyCall `json:"data"`
		}
		if err := json.Unmarshal(send(users[0].ID, http.MethodGet, "/calls?"+filter, "", 200), &list); err != nil {
			t.Fatal(err)
		}
		if list.Total != 1 || len(list.Data) != 1 {
			t.Fatalf("filter %s: %+v", filter, list)
		}
	}
	send(users[1].ID, http.MethodPut, "/calls/999999/review", review, 404)
	send(users[1].ID, http.MethodGet, "/calls/not-an-id", "", 400)
	if err := tx.Model(&users[0]).Update("department", "运营").Error; err != nil {
		t.Fatal(err)
	}
	send(users[0].ID, http.MethodPut, path+"/registration", registration, 403)
}
