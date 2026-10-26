package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/cache"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/utils"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	ClientType string `json:"client_type"` // "web" | "admin", defaults to "web"
}

type LoginResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user model.User
	if err := database.DB.Preload("Roles").Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}
	if !user.Enabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
		return
	}
	loadUserOrderPlatforms(&user)

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// Normalise client type
	clientType := req.ClientType
	if clientType != "admin" {
		clientType = "web"
	}
	longTerm := clientType == "web" && user.LongLogin
	token, err := utils.GenerateToken(user.ID, user.Username, longTerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Device limit only applies to web clients
	if clientType == "web" {
		if err := checkDeviceLimit(user.ID); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
	}

	// Record this session with an initial heartbeat timestamp
	now := time.Now()
	tokenHash := utils.HashToken(token)
	session := model.UserSession{
		UserID:       user.ID,
		TokenHash:    tokenHash,
		ClientType:   clientType,
		UserAgent:    c.GetHeader("User-Agent"),
		IP:           c.ClientIP(),
		ExpiresAt:    now.Add(sessionDuration(longTerm)),
		LastActiveAt: &now,
	}
	database.DB.Create(&session)

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

func sessionDuration(longTerm bool) time.Duration {
	if longTerm {
		return 10 * 365 * 24 * time.Hour
	}
	return 24 * time.Hour
}

// onlineThreshold returns the cutoff time for "actively online" sessions.
// A session is considered online if its last heartbeat is within this window.
const heartbeatWindow = 90 * time.Second

func onlineThreshold() time.Time {
	return time.Now().Add(-heartbeatWindow)
}

// checkDeviceLimit rejects the login if the user already has
// maxConcurrentDevices sessions that sent a heartbeat recently.
func checkDeviceLimit(userID uint) error {
	// Read setting; 0 or missing means unlimited
	var setting model.Setting
	maxDevices := 0
	if err := database.DB.Where("key = ?", "max_concurrent_devices").First(&setting).Error; err == nil {
		fmt.Sscanf(setting.Value, "%d", &maxDevices)
	}

	// Clean up fully expired JWT sessions first
	database.DB.Where("user_id = ? AND expires_at < ?", userID, time.Now()).Delete(&model.UserSession{})

	if maxDevices <= 0 {
		return nil // unlimited
	}

	// Only count web sessions that have sent a heartbeat recently (truly online)
	var count int64
	database.DB.Model(&model.UserSession{}).
		Where("user_id = ? AND client_type = 'web' AND last_active_at > ?", userID, onlineThreshold()).
		Count(&count)

	if int(count) >= maxDevices {
		return fmt.Errorf("已达到最大同时在线设备数（%d 台），请先在其他设备上退出登录后重试", maxDevices)
	}
	return nil
}

// Heartbeat updates the last_active_at timestamp for the caller's session,
// keeping it counted as "online" by the device-limit check.
// Auth is handled by AuthMiddleware (validates JWT + session existence).
func Heartbeat(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	parts := strings.Split(authHeader, " ")
	tokenHash := utils.HashToken(parts[1])
	now := time.Now()
	// Recheck the database even if authentication used a cached session.
	// Heartbeats update presence only; they never extend the login expiry.
	result := database.DB.Model(&model.UserSession{}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).
		Where("user_id IN (?)", database.DB.Model(&model.User{}).Select("id").Where("enabled = ?", true)).
		Update("last_active_at", now)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "登录状态检查失败，请稍后重试"})
		return
	}
	if result.RowsAffected == 0 {
		cache.InvalidateSession(tokenHash)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "登录已失效，请重新登录"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenHash := utils.HashToken(parts[1])
			if err := database.DB.Where("token_hash = ?", tokenHash).Delete(&model.UserSession{}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "退出登录失败，请重试"})
				return
			}
			cache.InvalidateSession(tokenHash)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	loadUserOrderPlatforms(&user)

	c.JSON(http.StatusOK, user)
}

type SendVerificationCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

func SendVerificationCode(c *gin.Context) {
	var req SendVerificationCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效的邮箱地址"})
		return
	}

	email := strings.ToLower(req.Email)

	if utils.IsEmailExists(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该邮箱已被注册"})
		return
	}

	if cache.RDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务暂不可用，请稍后重试"})
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("email_code:%s", email)

	coolingKey := fmt.Sprintf("email_cooling:%s", email)
	exists, _ := cache.RDB.Exists(ctx, coolingKey).Result()
	if exists > 0 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "验证码发送过于频繁，请稍后再试"})
		return
	}

	code := utils.GenerateVerificationCode()

	if err := cache.RDB.Set(ctx, key, code, 5*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证码生成失败"})
		return
	}

	cache.RDB.Set(ctx, coolingKey, "1", 60*time.Second)

	if err := utils.SendVerificationCode(email, code); err != nil {
		log.Printf("发送验证码失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证码发送失败，请检查邮箱配置"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送"})
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required,min=6"`
	Code     string `json:"code"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写完整的注册信息"})
		return
	}

	var setting model.Setting
	emailRegEnabled := database.DB.Where(&model.Setting{Key: "email_register_enabled"}).First(&setting).Error == nil && setting.Value == "true"

	var email string
	if emailRegEnabled {
		if req.Email == "" || req.Code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请填写邮箱和验证码"})
			return
		}
		email = strings.ToLower(req.Email)

		if cache.RDB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "服务暂不可用，请稍后重试"})
			return
		}

		ctx := context.Background()
		key := fmt.Sprintf("email_code:%s", email)

		savedCode, err := cache.RDB.Get(ctx, key).Result()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已过期，请重新获取"})
			return
		}

		if savedCode != req.Code {
			c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
			return
		}

		if utils.IsEmailExists(email) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该邮箱已被注册"})
			return
		}

		cache.RDB.Del(ctx, key)
	} else {
		email = fmt.Sprintf("%s@local.user", req.Username)
	}

	var existingUser model.User
	if err := database.DB.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名已被使用"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}

	user := model.User{
		Username: req.Username,
		Email:    email,
		Password: string(hashedPassword),
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，请稍后重试"})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册成功，请登录"})
		return
	}

	// Create a web session for the newly registered user
	now := time.Now()
	session := model.UserSession{
		UserID:       user.ID,
		TokenHash:    utils.HashToken(token),
		ClientType:   "web",
		UserAgent:    c.GetHeader("User-Agent"),
		IP:           c.ClientIP(),
		ExpiresAt:    now.Add(sessionDuration(false)),
		LastActiveAt: &now,
	}
	database.DB.Create(&session)

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

func CheckEmailRegisterEnabled(c *gin.Context) {
	var setting model.Setting
	err := database.DB.Where(&model.Setting{Key: "email_register_enabled"}).First(&setting).Error
	enabled := err == nil && setting.Value == "true"
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

// ── Change Password (logged-in user) ────────────────────────────────────────

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

func ChangePassword(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少 6 位"})
		return
	}

	var user model.User
	if err := database.DB.First(&user, userIDVal).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前密码错误"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	if err := database.DB.Model(&user).Update("password", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	// Invalidate all OTHER sessions (keep current one valid)
	tokenHash := ""
	if auth := c.GetHeader("Authorization"); len(auth) > 7 {
		tokenHash = utils.HashToken(auth[7:])
	}
	database.DB.Where("user_id = ? AND token_hash != ?", user.ID, tokenHash).Delete(&model.UserSession{})

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

// ── Forgot / Reset Password ──────────────────────────────────────────────────

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword sends a password-reset verification code to the given email.
func ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效的邮箱地址"})
		return
	}

	email := strings.ToLower(req.Email)

	// Verify email exists
	var user model.User
	if err := database.DB.Where("LOWER(email) = ?", email).First(&user).Error; err != nil {
		// Return generic success to avoid email enumeration
		c.JSON(http.StatusOK, gin.H{"message": "如果该邮箱已注册，验证码已发送"})
		return
	}

	if cache.RDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务暂不可用，请稍后重试"})
		return
	}

	ctx := context.Background()
	coolingKey := fmt.Sprintf("reset_cooling:%s", email)
	if exists, _ := cache.RDB.Exists(ctx, coolingKey).Result(); exists > 0 {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "发送过于频繁，请稍后再试"})
		return
	}

	code := utils.GenerateVerificationCode()
	codeKey := fmt.Sprintf("reset_code:%s", email)

	if err := cache.RDB.Set(ctx, codeKey, code, 10*time.Minute).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "验证码生成失败"})
		return
	}
	cache.RDB.Set(ctx, coolingKey, "1", 60*time.Second)

	if err := utils.SendPasswordResetCode(email, code); err != nil {
		log.Printf("发送密码重置邮件失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "邮件发送失败，请检查邮箱配置"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "如果该邮箱已注册，验证码已发送"})
}

type ResetPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Code     string `json:"code" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// ResetPassword validates the reset code and updates the user's password.
func ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写完整信息，密码至少 6 位"})
		return
	}

	email := strings.ToLower(req.Email)

	if cache.RDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "服务暂不可用，请稍后重试"})
		return
	}

	ctx := context.Background()
	codeKey := fmt.Sprintf("reset_code:%s", email)

	savedCode, err := cache.RDB.Get(ctx, codeKey).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已过期，请重新获取"})
		return
	}
	if savedCode != req.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	var user model.User
	if err := database.DB.Where("LOWER(email) = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户不存在"})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码重置失败"})
		return
	}

	if err := database.DB.Model(&user).Update("password", string(hashed)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	// Invalidate all existing sessions for this user so old tokens can't be reused
	database.DB.Where("user_id = ?", user.ID).Delete(&model.UserSession{})

	// Remove the used reset code
	cache.RDB.Del(ctx, codeKey)

	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功，请重新登录"})
}

// ── Session Management ───────────────────────────────────────────────────────

type SessionInfo struct {
	ID           uint       `json:"id"`
	ClientType   string     `json:"client_type"`
	UserAgent    string     `json:"user_agent"`
	IP           string     `json:"ip"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt *time.Time `json:"last_active_at"`
	IsCurrent    bool       `json:"is_current"`
}

func GetMySessions(c *gin.Context) {
	userIDVal, _ := c.Get("userID")

	currentHash := ""
	if auth := c.GetHeader("Authorization"); len(auth) > 7 {
		currentHash = utils.HashToken(auth[7:])
	}

	var sessions []model.UserSession
	database.DB.Where("user_id = ? AND expires_at > ?", userIDVal, time.Now()).
		Order("created_at DESC").
		Find(&sessions)

	result := make([]SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, SessionInfo{
			ID:           s.ID,
			ClientType:   s.ClientType,
			UserAgent:    s.UserAgent,
			IP:           s.IP,
			CreatedAt:    s.CreatedAt,
			LastActiveAt: s.LastActiveAt,
			IsCurrent:    s.TokenHash == currentHash,
		})
	}

	c.JSON(http.StatusOK, result)
}

func RevokeSession(c *gin.Context) {
	userIDVal, _ := c.Get("userID")
	id := c.Param("id")

	result := database.DB.Where("id = ? AND user_id = ?", id, userIDVal).Delete(&model.UserSession{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已下线该设备"})
}

// ── Bind / Update Email ───────────────────────────────────────────────────────

func SendBindEmailCode(c *gin.Context) {
	userIDVal, _ := c.Get("userID")

	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入有效的邮箱地址"})
		return
	}

	ctx := context.Background()
	coolingKey := fmt.Sprintf("bind_email_cooling:%d", userIDVal)
	if ttl, _ := cache.RDB.TTL(ctx, coolingKey).Result(); ttl > 0 {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":        fmt.Sprintf("请 %d 秒后再试", int(ttl.Seconds())),
			"wait_seconds": int(ttl.Seconds()),
		})
		return
	}

	// Check if email is already used by another account
	var existing model.User
	if err := database.DB.Where("email = ? AND id != ?", req.Email, userIDVal).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该邮箱已被其他账号使用"})
		return
	}

	code := utils.GenerateVerificationCode()
	codeKey := fmt.Sprintf("bind_email_code:%d", userIDVal)
	// Store "code:email" together so we verify against the same email later
	cache.RDB.Set(ctx, codeKey, req.Email+":"+code, 10*time.Minute)
	cache.RDB.Set(ctx, coolingKey, "1", 60*time.Second)

	if err := utils.SendVerificationCode(req.Email, code); err != nil {
		log.Printf("发送绑定邮箱验证码失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "邮件发送失败，请检查邮箱配置"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送"})
}

func BindEmail(c *gin.Context) {
	userIDVal, _ := c.Get("userID")

	var req struct {
		Email string `json:"email" binding:"required,email"`
		Code  string `json:"code"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	ctx := context.Background()
	codeKey := fmt.Sprintf("bind_email_code:%d", userIDVal)
	stored, err := cache.RDB.Get(ctx, codeKey).Result()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码已过期，请重新获取"})
		return
	}

	// stored format: "email:code"
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 || parts[0] != req.Email || parts[1] != req.Code {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	// Check again that email is not taken (race condition guard)
	var existing model.User
	if err := database.DB.Where("email = ? AND id != ?", req.Email, userIDVal).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该邮箱已被其他账号使用"})
		return
	}

	if err := database.DB.Model(&model.User{}).Where("id = ?", userIDVal).Update("email", req.Email).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绑定失败，请重试"})
		return
	}

	cache.RDB.Del(ctx, codeKey)
	c.JSON(http.StatusOK, gin.H{"message": "邮箱绑定成功", "email": req.Email})
}
