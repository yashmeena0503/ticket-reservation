package auth

import (
	"context"
	"errors"
	"reservation/internal/db"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email taken")
)

type Service struct {
	q      *db.Queries
	secret []byte
	ttl    time.Duration
}

func NewService(q *db.Queries, secret string) *Service {
	return &Service{
		q:      q,
		secret: []byte(secret),
		ttl:    15 * time.Minute,
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (db.NewUserRow, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return db.NewUserRow{}, err
	}
	user, err := s.q.NewUser(ctx, db.NewUserParams{
		Email:        email,
		PasswordHash: hash,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
		return db.NewUserRow{}, ErrEmailTaken
	}
	return user, err
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	ok, err := argon2id.ComparePasswordAndHash(password, user.PasswordHash)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrInvalidCredentials
	}

	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   user.ID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}).SignedString(s.secret)
}

func (s *Service) Verify(token string) (uuid.UUID, error) {
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(claims.Subject)
}
