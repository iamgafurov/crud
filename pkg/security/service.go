package security

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

var ErrNotFound = errors.New("item not found")
var ErrExpired = errors.New("token is expired")
var ErrInternal = errors.New("internal error")
var ErrNoSuchUser = errors.New("no such user")
var ErrInvalidPassword = errors.New("invalid password")

type Service struct {
	pool *pgxpool.Pool
}

type Auth struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}
func (s *Service) Auth(login, password string) bool {
	bdPass := ""
	ctx := context.Background()
	log.Print(login, password)

	err := s.pool.QueryRow(ctx, `
	SELECT password FROM managers WHERE login = $1
	`, login).Scan(&bdPass)
	if err != nil {

		log.Print("Not ok", err)
		return false
	}
	if password != bdPass {
		log.Print("Not ok", err)
		return false
	}
	log.Print("ok")
	return true
}

func (s *Service) AuthenticateCusomer(
	ctx context.Context,
	token string,
) (id int64, err error) {

	expiredTime := time.Now()
	nowTimeInSec := expiredTime.UnixNano()
	err = s.pool.QueryRow(ctx, `SELECT customer_id, expire FROM customers_tokens WHERE token = $1`, token).Scan(&id, &expiredTime)
	if err != nil {
		log.Print(err)
		return 0, ErrNoSuchUser
	}

	if nowTimeInSec > expiredTime.UnixNano() {
		return -1, ErrExpired
	}
	return id, nil
}
