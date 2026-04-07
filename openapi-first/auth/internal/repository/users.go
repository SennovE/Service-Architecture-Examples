package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
)

type User struct {
	ID           uuid.UUID
	Username     string
	Role         string
	PasswordHash string `db:"password_hash"`
}

type UserRepository struct {
	conn *sqlx.DB
}

func NewUserRepository(conn *sqlx.DB) *UserRepository {
	return &UserRepository{conn: conn}
}

func (db *UserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	const query = `
		SELECT id, username, role, password_hash
		FROM users
		WHERE users.id = $1
		LIMIT 1;
	`
	user := new(User)
	err := db.conn.GetContext(ctx, user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (db *UserRepository) GetUser(ctx context.Context, username string) (*User, error) {
	const query = `
		SELECT id, username, role, password_hash
		FROM users
		WHERE users.username = $1
		LIMIT 1;
	`
	user := new(User)
	err := db.conn.GetContext(ctx, user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (db *UserRepository) CreateUser(ctx context.Context, username, passwordHash string) (*User, error) {
	const query = `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, username, role;
	`
	user := new(User)
	err := db.conn.QueryRowxContext(ctx, query, username, passwordHash).StructScan(user)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}
	return user, nil
}

func isUniqueViolation(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code == "23505"
	}
	return false
}
