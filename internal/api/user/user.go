package user

import (
	"context"
	"net/http"
	"time"

	"richmond-api/internal/api/errors"
	e "richmond-api/internal/api/errors"
	"richmond-api/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type CreateRequest struct {
	Login    string `json:"login"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Login    string `json:"login"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	Login string `json:"login"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type Querier interface {
	GetUserByName(ctx context.Context, userName string) (db.User, error)
	GetUserByID(ctx context.Context, userID int32) (db.User, error)
	CreateUser(ctx context.Context, params db.CreateUserParams) (db.User, error)
	CreateSession(
		ctx context.Context,
		params db.CreateSessionParams,
	) (db.Session, error)
}

type UserHandler struct {
	queries Querier
}

func NewUserHandler(queries Querier) *UserHandler {
	return &UserHandler{queries: queries}
}

// @Summary Create a new user
// @Description Creates a new user with hashed password
// @Tags user
// @Accept json
// @Produce json
// @Param request body CreateRequest true "User credentials"
// @Success 201 {object} UserResponse
// @Failure 409 {object} e.ErrorResponse
// @Router /api/v1/user/new [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.BadRequest(c, "invalid request")
		return
	}
	_, err := h.queries.GetUserByName(c.Request.Context(), req.Login)
	if err == nil {
		errors.Conflict(c, "user already exists")
		return
	}
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		errors.InternalError(c, "failed to hash password")
		return
	}
	user, err := h.queries.CreateUser(c.Request.Context(), db.CreateUserParams{
		UserName: req.Login,
		UserPass: string(hashedPassword),
	})
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to create user"},
		)
		return
	}
	c.JSON(http.StatusCreated, UserResponse{Login: user.UserName})
}

// @Summary Login
// @Description Login with credentials, returns auth token
// @Tags user
// @Accept json
// @Produce json
// @Param request body LoginRequest true "User credentials"
// @Success 200 {object} TokenResponse
// @Failure 401 {object} e.ErrorResponse
// @Router /api/v1/user/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errors.BadRequest(c, "invalid request")
		return
	}
	user, err := h.queries.GetUserByName(c.Request.Context(), req.Login)
	if err != nil {
		errors.Unauthorized(c, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.UserPass), []byte(req.Password)); err != nil {
		errors.Unauthorized(c, "invalid credentials")
		return
	}
	token := uuid.New().String()
	expiresAt := pgtype.Timestamp{
		Time:  time.Now().Add(1 * time.Hour),
		Valid: true,
	}
	_, err = h.queries.CreateSession(
		c.Request.Context(),
		db.CreateSessionParams{
			UserID:    user.UserID,
			Token:     token,
			ExpiresAt: expiresAt,
		},
	)
	if err != nil {
		errors.InternalError(c, "failed to create session")
		return
	}
	c.JSON(http.StatusOK, TokenResponse{Token: token})
}

// @Summary Get current user
// @Description Returns user data for authenticated user
// @Tags user
// @Produce json
// @Success 200 {object} UserResponse
// @Failure 401 {object} e.ErrorResponse
// @Router /api/v1/user [get]
// @Security BearerAuth
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
func (h *UserHandler) Get(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		errors.Unauthorized(c, "unauthorized")
		return
	}
	user, err := h.queries.GetUserByID(c.Request.Context(), userID.(int32))
	if err != nil {
		errors.NotFound(c, "user not found")
		return
	}
	c.JSON(http.StatusOK, UserResponse{Login: user.UserName})
}
