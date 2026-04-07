package service

import (
	"auth/internal/config"
	"auth/internal/repository"
	"auth/internal/utils"
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("wrong credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

type UserService struct {
	db  *repository.UserRepository
	cfg *config.Config
}

func NewUserService(db *repository.UserRepository, cfg *config.Config) *UserService {
	return &UserService{db: db, cfg: cfg}
}

func (service *UserService) GetToken(ctx context.Context, username, password string) (string, string, *uuid.UUID, error) {
	user, err := service.db.GetUser(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", "", nil, ErrInvalidCredentials
		}
		return "", "", nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}
	accessToken, err := utils.CreateAccessToken(
		service.cfg.AuthPrivateKey, service.cfg.JWTAccsessExpires, user.ID, user.Role,
	)
	if err != nil {
		return "", "", nil, err
	}
	refreshToken, err := utils.CreateRefreshToken(
		service.cfg.AuthPrivateKey, service.cfg.JWTRefreshExpires, user.ID,
	)
	return accessToken, refreshToken, &user.ID, err
}

func (service *UserService) RefreshToken(ctx context.Context, refreshToken string) (string, *uuid.UUID, error) {
	claims, err := utils.VerifyRefreshToken(service.cfg.AuthPublicKey, refreshToken)
	if err != nil {
		return "", nil, err
	}
	userID, ok := claims["sub"].(string)
	if !ok {
		return "", nil, utils.ErrJWTDecode
	}
	user, err := service.db.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", nil, utils.ErrJWTDecode
		}
		return "", nil, err
	}
	token, err := utils.CreateAccessToken(
		service.cfg.AuthPrivateKey, service.cfg.JWTAccsessExpires, user.ID, user.Role,
	)
	return token, nil, err
}

func (service *UserService) CreateUser(ctx context.Context, username, password string) (string, string, *uuid.UUID, error) {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return "", "", nil, err
	}
	user, err := service.db.CreateUser(ctx, username, passwordHash)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return "", "", nil, ErrUserAlreadyExists
		}
		return "", "", nil, err
	}
	accessToken, err := utils.CreateAccessToken(
		service.cfg.AuthPrivateKey, service.cfg.JWTAccsessExpires, user.ID, user.Role,
	)
	if err != nil {
		return "", "", nil, err
	}
	refreshToken, err := utils.CreateRefreshToken(
		service.cfg.AuthPrivateKey, service.cfg.JWTRefreshExpires, user.ID,
	)
	return accessToken, refreshToken, &user.ID, err
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}
