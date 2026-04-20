package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"ssh-port-forwarder/internal/config"
	"ssh-port-forwarder/internal/model"
	"ssh-port-forwarder/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token has expired")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

type AuthService struct {
	userRepo  repository.UserRepository
	jwtConfig config.JWTConfig
}

type TokenPair struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

type Claims struct {
	UserID   uint64 `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// NewAuthService 创建新的认证服务实例
func NewAuthService(userRepo repository.UserRepository, jwtConfig config.JWTConfig) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtConfig: jwtConfig,
	}
}

// HashPassword 使用 bcrypt 对密码进行哈希，cost=12
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 校验密码是否匹配哈希值
func (s *AuthService) CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateTokenPair 生成 JWT Token 和 RefreshToken
func (s *AuthService) GenerateTokenPair(user *model.User) (*TokenPair, error) {
	now := time.Now()

	// 设置默认过期时间
	tokenExpire := s.jwtConfig.TokenExpire
	if tokenExpire == 0 {
		tokenExpire = 86400 // 默认 24 小时
	}
	refreshExpire := s.jwtConfig.RefreshExpire
	if refreshExpire == 0 {
		refreshExpire = 604800 // 默认 7 天
	}

	// 创建 Access Token
	accessClaims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(tokenExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.jwtConfig.SecretCurrent))
	if err != nil {
		return nil, err
	}

	// 创建 Refresh Token
	refreshClaims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(refreshExpire) * time.Second)),
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		Subject:   user.Username,
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.jwtConfig.SecretCurrent))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		Token:        accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(time.Duration(tokenExpire) * time.Second).Unix(),
	}, nil
}

// ValidateToken 验证 JWT Token，支持 Secret 轮转
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	// 首先尝试用当前 Secret 验证
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtConfig.SecretCurrent), nil
	})

	if err == nil && token.Valid {
		if claims, ok := token.Claims.(*Claims); ok {
			return claims, nil
		}
	}

	// 如果失败且有配置旧 Secret，尝试用旧 Secret 验证
	if s.jwtConfig.SecretPrevious != "" {
		token, err = jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(s.jwtConfig.SecretPrevious), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(*Claims); ok {
				return claims, nil
			}
		}
	}

	// 检查是否是过期错误
	if errors.Is(err, jwt.ErrTokenExpired) {
		return nil, ErrTokenExpired
	}

	return nil, ErrInvalidToken
}

// Login 用户登录
func (s *AuthService) Login(username, password string) (*TokenPair, error) {
	// 查找用户
	user, err := s.userRepo.FindByUsername(username)
	if err != nil || user == nil {
		return nil, ErrInvalidCredentials
	}

	// 校验密码
	if !s.CheckPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// 生成 Token
	return s.GenerateTokenPair(user)
}

// RefreshToken 刷新 Token
func (s *AuthService) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	// 验证 Refresh Token
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.jwtConfig.SecretCurrent), nil
	})

	if err != nil || !token.Valid {
		// 尝试用旧 Secret 验证
		if s.jwtConfig.SecretPrevious != "" {
			token, err = jwt.Parse(refreshTokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(s.jwtConfig.SecretPrevious), nil
			})
		}
		if err != nil || !token.Valid {
			return nil, ErrInvalidToken
		}
	}

	// 获取用户名
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	subject, err := claims.GetSubject()
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 查找用户
	user, err := s.userRepo.FindByUsername(subject)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	// 生成新的 TokenPair
	return s.GenerateTokenPair(user)
}

// CreateDefaultAdmin 创建默认管理员账户
func (s *AuthService) CreateDefaultAdmin(username, password string) error {
	// 检查用户是否已存在
	existing, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return err
	}
	if existing != nil {
		// 用户已存在，跳过
		return nil
	}

	// 哈希密码
	passwordHash, err := s.HashPassword(password)
	if err != nil {
		return err
	}

	// 创建管理员用户
	user := &model.User{
		Username:     username,
		PasswordHash: passwordHash,
		Role:         "admin",
	}

	return s.userRepo.Create(user)
}
