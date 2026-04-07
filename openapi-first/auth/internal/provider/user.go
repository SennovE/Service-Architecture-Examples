package provider

import (
	"auth/internal/gen"
	"auth/internal/service"
	"auth/internal/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserProvider struct {
	srvc *service.UserService
}

func NewUserProvider(srvc *service.UserService) *UserProvider {
	return &UserProvider{srvc: srvc}
}

func (provider *UserProvider) Login(ctx *gin.Context) {
	var req gen.LoginJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	accessToken, refreshToken, userID, err := provider.srvc.GetToken(ctx, req.Name, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			makeErrorResponse(
				ctx, http.StatusConflict, gen.WRONGCREDENTIALS, "wrong credetials", "",
			)
			return
		}
		makeErrorResponse(
			ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", err.Error(),
		)
		return
	}

	ctx.Set("userID", *userID)
	ctx.JSON(
		http.StatusOK,
		gen.TokenPairResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	)
}

func (provider *UserProvider) RefreshToken(ctx *gin.Context) {
	var req gen.RefreshTokenJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	accessToken, userID, err := provider.srvc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, utils.ErrJWTDecode) || errors.Is(err, utils.ErrJWTExpired) {
			makeErrorResponse(
				ctx, http.StatusConflict, gen.REFRESHTOKENINVALID, "refresh invalid token", "",
			)
			return
		}
		makeErrorResponse(
			ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", err.Error(),
		)
		return
	}

	ctx.Set("userID", *userID)
	ctx.JSON(
		http.StatusOK,
		gen.TokenPairResponse{
			AccessToken:  accessToken,
			RefreshToken: req.RefreshToken,
		},
	)
}

func (provider *UserProvider) RegisterUser(ctx *gin.Context) {
	var req gen.RegisterUserJSONRequestBody
	if !validateRequestBody(ctx, &req) {
		return
	}

	accessToken, refreshToken, userID, err := provider.srvc.CreateUser(ctx, req.Name, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			makeErrorResponse(
				ctx, http.StatusConflict, gen.USERALREADYEXISTS, "user with same name exists", "",
			)
			return
		}
		makeErrorResponse(
			ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", err.Error(),
		)
		return
	}

	ctx.Set("userID", *userID)
	ctx.JSON(
		http.StatusCreated,
		gen.TokenPairResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	)
}

func validateRequestBody[T any](ctx *gin.Context, req *T) bool {
	if err := ctx.ShouldBindJSON(req); err != nil {
		makeErrorResponse(
			ctx, http.StatusBadRequest, gen.VALIDATIONERROR, "invalid body", err.Error(),
		)
		return false
	}
	return true
}

func makeErrorResponse(ctx *gin.Context, statusCode int, errorCode gen.ErrorCode, msg, err string) {
	errResp := gen.ErrorResponse{
		ErrorCode: errorCode,
		Message:   msg,
	}
	if err != "" {
		errResp.Details = &map[string]any{"error": err}
	}
	ctx.JSON(
		statusCode,
		errResp,
	)
}
