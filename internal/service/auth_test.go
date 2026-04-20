package service

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"ssh-port-forwarder/internal/config"
	"ssh-port-forwarder/internal/model"
)

// MockUserRepository 模拟用户仓库
type MockUserRepository struct {
	users map[string]*model.User
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		users: make(map[string]*model.User),
	}
}

func (m *MockUserRepository) Create(user *model.User) error {
	if _, exists := m.users[user.Username]; exists {
		return errors.New("user already exists")
	}
	m.users[user.Username] = user
	return nil
}

func (m *MockUserRepository) FindByID(id uint64) (*model.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *MockUserRepository) FindByUsername(username string) (*model.User, error) {
	user, exists := m.users[username]
	if !exists {
		return nil, nil // 找不到用户时返回 (nil, nil)
	}
	return user, nil
}

func (m *MockUserRepository) Update(user *model.User) error {
	m.users[user.Username] = user
	return nil
}

func (m *MockUserRepository) Delete(id uint64) error {
	return nil
}

func (m *MockUserRepository) List(page, pageSize int) ([]model.User, int64, error) {
	return nil, 0, nil
}

// TestHashAndCheckPassword 测试密码哈希和校验
func TestHashAndCheckPassword(t *testing.T) {
	service := NewAuthService(nil, config.JWTConfig{})

	password := "mySecretPassword123"

	// 测试哈希
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("HashPassword returned empty string")
	}

	if hash == password {
		t.Error("HashPassword returned plaintext")
	}

	// 测试正确密码校验
	if !service.CheckPassword(password, hash) {
		t.Error("CheckPassword returned false for correct password")
	}

	// 测试错误密码校验
	if service.CheckPassword("wrongPassword", hash) {
		t.Error("CheckPassword returned true for wrong password")
	}

	// 测试不同密码生成不同哈希
	hash2, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword second call failed: %v", err)
	}
	if hash == hash2 {
		t.Error("Same password produced same hash (bcrypt should add salt)")
	}
}

// TestGenerateAndValidateToken 测试生成和验证 Token
func TestGenerateAndValidateToken(t *testing.T) {
	jwtConfig := config.JWTConfig{
		SecretCurrent: "my-test-secret-key-for-jwt-signing-32b",
		TokenExpire:   3600,
		RefreshExpire: 86400,
	}

	service := NewAuthService(nil, jwtConfig)

	user := &model.User{
		ID:       1,
		Username: "testuser",
		Role:     "admin",
	}

	// 生成 TokenPair
	tokenPair, err := service.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	if tokenPair.Token == "" {
		t.Error("GenerateTokenPair returned empty access token")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("GenerateTokenPair returned empty refresh token")
	}

	if tokenPair.ExpiresAt == 0 {
		t.Error("GenerateTokenPair returned zero ExpiresAt")
	}

	// 验证 Token
	claims, err := service.ValidateToken(tokenPair.Token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %d, want %d", claims.UserID, user.ID)
	}

	if claims.Username != user.Username {
		t.Errorf("Username mismatch: got %s, want %s", claims.Username, user.Username)
	}

	if claims.Role != user.Role {
		t.Errorf("Role mismatch: got %s, want %s", claims.Role, user.Role)
	}
}

// TestTokenRotationFallback 测试 Token Secret 轮转回退
func TestTokenRotationFallback(t *testing.T) {
	oldSecret := "old-secret-key-for-testing-jwt-tokens-"
	newSecret := "new-secret-key-for-testing-jwt-tokens-"

	// 使用旧 Secret 创建服务并生成 Token
	oldService := NewAuthService(nil, config.JWTConfig{
		SecretCurrent: oldSecret,
	})

	user := &model.User{
		ID:       1,
		Username: "testuser",
		Role:     "operator",
	}

	tokenPair, err := oldService.GenerateTokenPair(user)
	if err != nil {
		t.Fatalf("GenerateTokenPair with old secret failed: %v", err)
	}

	// 使用新 Secret + 旧 Secret 作为 Previous 创建新服务
	newService := NewAuthService(nil, config.JWTConfig{
		SecretCurrent:  newSecret,
		SecretPrevious: oldSecret,
	})

	// 验证旧 Secret 签发的 Token
	claims, err := newService.ValidateToken(tokenPair.Token)
	if err != nil {
		t.Fatalf("ValidateToken with fallback failed: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: got %d, want %d", claims.UserID, user.ID)
	}

	// 验证只用新 Secret 的服务无法验证旧 Token
	newOnlyService := NewAuthService(nil, config.JWTConfig{
		SecretCurrent: newSecret,
	})

	_, err = newOnlyService.ValidateToken(tokenPair.Token)
	if err == nil {
		t.Error("ValidateToken should fail without fallback secret")
	}
}

// TestValidateExpiredToken 测试过期 Token 验证失败
func TestValidateExpiredToken(t *testing.T) {
	secret := "test-secret-key-for-expired-token-test"

	// 创建一个已经过期的 Token
	expiredClaims := Claims{
		UserID:   1,
		Username: "testuser",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // 1 小时前过期
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign expired token: %v", err)
	}

	service := NewAuthService(nil, config.JWTConfig{
		SecretCurrent: secret,
	})

	_, err = service.ValidateToken(tokenString)
	if err == nil {
		t.Error("ValidateToken should fail for expired token")
	}

	if !errors.Is(err, ErrTokenExpired) && err != ErrInvalidToken {
		t.Errorf("Expected ErrTokenExpired or ErrInvalidToken, got: %v", err)
	}
}

// TestLogin 测试登录流程
func TestLogin(t *testing.T) {
	mockRepo := NewMockUserRepository()
	jwtConfig := config.JWTConfig{
		SecretCurrent: "test-secret-key-for-login-tests-32bytes",
		TokenExpire:   3600,
		RefreshExpire: 86400,
	}

	service := NewAuthService(mockRepo, jwtConfig)

	// 创建测试用户
	password := "testPassword123"
	passwordHash, _ := service.HashPassword(password)
	testUser := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: passwordHash,
		Role:         "admin",
	}
	mockRepo.Create(testUser)

	// 测试正确登录
	tokenPair, err := service.Login("testuser", password)
	if err != nil {
		t.Fatalf("Login with correct credentials failed: %v", err)
	}

	if tokenPair.Token == "" {
		t.Error("Login returned empty access token")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Login returned empty refresh token")
	}

	// 验证 Token 有效性
	claims, err := service.ValidateToken(tokenPair.Token)
	if err != nil {
		t.Fatalf("ValidateToken for login token failed: %v", err)
	}

	if claims.Username != "testuser" {
		t.Errorf("Token username mismatch: got %s, want testuser", claims.Username)
	}

	// 测试错误密码
	_, err = service.Login("testuser", "wrongpassword")
	if err != ErrInvalidCredentials {
		t.Errorf("Login with wrong password should return ErrInvalidCredentials, got: %v", err)
	}

	// 测试不存在的用户
	_, err = service.Login("nonexistent", password)
	if err != ErrInvalidCredentials {
		t.Errorf("Login with nonexistent user should return ErrInvalidCredentials, got: %v", err)
	}
}

// TestRefreshToken 测试刷新 Token
func TestRefreshToken(t *testing.T) {
	mockRepo := NewMockUserRepository()
	jwtConfig := config.JWTConfig{
		SecretCurrent: "test-secret-key-for-refresh-token-test",
		TokenExpire:   3600,
		RefreshExpire: 86400,
	}

	service := NewAuthService(mockRepo, jwtConfig)

	// 创建测试用户
	password := "testPassword123"
	passwordHash, _ := service.HashPassword(password)
	testUser := &model.User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: passwordHash,
		Role:         "operator",
	}
	mockRepo.Create(testUser)

	// 先生成 TokenPair
	tokenPair, err := service.GenerateTokenPair(testUser)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	// 使用 RefreshToken 刷新
	newTokenPair, err := service.RefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if newTokenPair.Token == "" {
		t.Error("RefreshToken returned empty access token")
	}

	if newTokenPair.RefreshToken == "" {
		t.Error("RefreshToken returned empty refresh token")
	}

	// 验证新 Token
	claims, err := service.ValidateToken(newTokenPair.Token)
	if err != nil {
		t.Fatalf("ValidateToken for refreshed token failed: %v", err)
	}

	if claims.Username != "testuser" {
		t.Errorf("Refreshed token username mismatch: got %s, want testuser", claims.Username)
	}

	// 测试无效 RefreshToken
	_, err = service.RefreshToken("invalid.token.string")
	if err != ErrInvalidToken {
		t.Errorf("RefreshToken with invalid token should return ErrInvalidToken, got: %v", err)
	}
}

// TestCreateDefaultAdmin 测试创建默认管理员
func TestCreateDefaultAdmin(t *testing.T) {
	mockRepo := NewMockUserRepository()
	service := NewAuthService(mockRepo, config.JWTConfig{})

	// 测试创建默认管理员
	err := service.CreateDefaultAdmin("admin", "adminPassword123")
	if err != nil {
		t.Fatalf("CreateDefaultAdmin failed: %v", err)
	}

	// 验证用户已创建
	user, err := mockRepo.FindByUsername("admin")
	if err != nil {
		t.Fatalf("Failed to find created admin: %v", err)
	}

	if user.Username != "admin" {
		t.Errorf("Username mismatch: got %s, want admin", user.Username)
	}

	if user.Role != "admin" {
		t.Errorf("Role mismatch: got %s, want admin", user.Role)
	}

	// 验证密码已正确哈希
	if !service.CheckPassword("adminPassword123", user.PasswordHash) {
		t.Error("Admin password hash check failed")
	}

	// 再次创建应该跳过（不报错）
	err = service.CreateDefaultAdmin("admin", "differentPassword")
	if err != nil {
		t.Errorf("CreateDefaultAdmin for existing user should not error, got: %v", err)
	}

	// 验证密码未被修改
	user2, _ := mockRepo.FindByUsername("admin")
	if !service.CheckPassword("adminPassword123", user2.PasswordHash) {
		t.Error("Password should not be changed when user already exists")
	}
}
