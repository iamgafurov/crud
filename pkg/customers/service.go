package customers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("item not found")
var ErrInternal = errors.New("internal error")
var ErrNoSuchUser = errors.New("no such user")
var ErrInvalidPassword = errors.New("invalid password")

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type Customer struct {
	ID      int64     `json:"id"`
	Name    string    `json:"name"`
	Phone   string    `json:"phone"`
	Active  bool      `json:"active"`
	Created time.Time `json:"created"`
}

type CustomerAuth struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (s *Service) ByID(ctx context.Context, id int64) (*Customer, error) {
	item := &Customer{}

	err := s.pool.QueryRow(ctx, `
	SELECT id,name, phone, active, created FROM customers WHERE id = $1
	`, id).Scan(&item.ID, &item.Name, &item.Phone, &item.Active, &item.Created)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	return item, nil
}

func (s *Service) All(ctx context.Context) ([]*Customer, error) {
	items := make([]*Customer, 0)
	rows, err := s.pool.Query(ctx, `
	SELECT id,name, phone, active, created FROM customers ORDER BY id
	`)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	for rows.Next() {
		item := &Customer{}
		rows.Scan(&item.ID, &item.Name, &item.Phone, &item.Active, &item.Created)
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) AllActive(ctx context.Context) ([]*Customer, error) {
	items := make([]*Customer, 0)
	rows, err := s.pool.Query(ctx, `
	SELECT id,name, phone, active, created FROM customers WHERE active= true ORDER BY id;
	`)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}

	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}

	for rows.Next() {
		item := &Customer{}
		rows.Scan(&item.ID, &item.Name, &item.Phone, &item.Active, &item.Created)
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, item *CustomerAuth) (*Customer, error) {
	customer := &Customer{
		Name:   item.Name,
		Phone:  item.Phone,
		Active: true,
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(item.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	log.Print(hex.EncodeToString(hash))
	err = s.pool.QueryRow(ctx, `
	INSERT INTO customers(name,phone,password) VALUES ($1,$2,$3)  RETURNING id;
	`, item.Name, item.Phone, hash).Scan(&customer.ID)
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	return customer, nil
}

func (s *Service) Update(ctx context.Context, item *Customer) (*Customer, error) {
	customer := &Customer{
		ID:    item.ID,
		Name:  item.Name,
		Phone: item.Phone,
	}

	err := s.pool.QueryRow(ctx, `
	UPDATE customers SET name =$1,phone=$2 WHERE id =$3 RETURNING active,created
	`, item.Name, item.Phone, item.ID).Scan(&customer.Active, &customer.Created)
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	return customer, nil
}

func (s *Service) RemoveByID(ctx context.Context, id int64) (*Customer, error) {
	customer := &Customer{}
	err := s.pool.QueryRow(ctx, `
	DELETE FROM customers WHERE id= $1 RETURNING id,name,phone,active,created
	`, id).Scan(&customer.ID, &customer.Name, &customer.Phone, &customer.Active, &customer.Created)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	return customer, nil
}

func (s *Service) BlockByID(ctx context.Context, id int64) (*Customer, error) {
	customer := &Customer{}
	err := s.pool.QueryRow(ctx, `
	UPDATE customers SET active= false WHERE id= $1 RETURNING id,name,phone,active,created
	`, id).Scan(&customer.ID, &customer.Name, &customer.Phone, &customer.Active, &customer.Created)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	return customer, nil
}

func (s *Service) UnBlockByID(ctx context.Context, id int64) (*Customer, error) {
	customer := &Customer{}
	err := s.pool.QueryRow(ctx, `
	UPDATE customers SET active= true WHERE id= $1 RETURNING id,name,phone,active,created
	`, id).Scan(&customer.ID, &customer.Name, &customer.Phone, &customer.Active, &customer.Created)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		log.Print(err)
		return nil, ErrInternal
	}
	return customer, nil
}

func (s *Service) TokenForCustomer(
	ctx context.Context,
	phone string, password string,
) (token string, err error) {
	var hash string
	var id int64
	err = s.pool.QueryRow(ctx, `SELECT id,password From customers WHERE phone = $1`, phone).Scan(&id, &hash)

	if err == pgx.ErrNoRows {
		return "", ErrInvalidPassword
	}
	if err != nil {
		return "", ErrInternal
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return "", ErrInvalidPassword
	}
	buffer := make([]byte, 256)
	n, err := rand.Read(buffer)
	if n != len(buffer) || err != nil {
		return "", ErrInternal
	}

	token = hex.EncodeToString(buffer)
	_, err = s.pool.Exec(ctx, `INSERT INTO customers_tokens(token,customer_id) VALUES($1,$2)`, token, id)
	if err != nil {
		return "", ErrInternal
	}

	return token, nil
}
