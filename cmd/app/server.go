package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/iamgafurov/crud/pkg/customers"
	"github.com/iamgafurov/crud/pkg/security"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	GET    = "GET"
	POST   = "POST"
	DELETE = "DELETE"
)

type Server struct {
	mux          *mux.Router
	customersSvc *customers.Service
	securitySvc  *security.Service
}
type Token struct {
	Token string `json:"token"`
}

type Responce struct {
	CustomerID int64  `json:"customerId"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
}

type ResponceOk struct {
	Status     string `json:"status"`
	CustomerID int64  `json:"customerId"`
}

type ResponceFail struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

var ErrNotFound = errors.New("item not found")
var ErrExpired = errors.New("token is expired")
var ErrInternal = errors.New("internal error")
var ErrNoSuchUser = errors.New("no such user")
var ErrInvalidPassword = errors.New("invalid password")

func NewServer(mux *mux.Router, customersSvc *customers.Service, securitySvc *security.Service) *Server {
	return &Server{mux: mux, customersSvc: customersSvc, securitySvc: securitySvc}
}

func (s *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.mux.ServeHTTP(writer, request)
}

func (s *Server) Init() {
	s.mux.HandleFunc("/customers/{id:[0-9]+}", s.handleGetCustomersByID).Methods(GET)
	s.mux.HandleFunc("/customers/active", s.handleGetCustomersAllActive).Methods(GET)
	s.mux.HandleFunc("/customers", s.handleGetCustomersAll).Methods(GET)

	s.mux.HandleFunc("/api/customers/token", s.handleGetToken).Methods(POST)
	s.mux.HandleFunc("/api/customers/token/validate", s.handleValidateToken).Methods(POST)
	s.mux.HandleFunc("/api/customers", s.handleGetCustomersSave).Methods(POST)
	s.mux.HandleFunc("/customers/{id}/block", s.handleCustomersBlockByID).Methods(POST)

	s.mux.HandleFunc("/customers/{id}/block", s.handleCustomersUnBlockByID).Methods(DELETE)
	s.mux.HandleFunc("/customers", s.handleCustomersRemoveByID).Methods(DELETE)
	//s.mux.Use(middleware.Basic(s.securitySvc.Auth))
}

func (s *Server) handleGetCustomersByID(writer http.ResponseWriter, request *http.Request) {
	idParam, ok := mux.Vars(request)["id"]
	if !ok {
		log.Print("Cant parse id")
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		log.Print(err)
		http.Error(writer, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	item, err := s.customersSvc.ByID(request.Context(), id)
	if errors.Is(err, customers.ErrNotFound) {
		http.Error(writer, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(item)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	_, err = writer.Write(data)
	if err != nil {
		http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGetCustomersAll(w http.ResponseWriter, r *http.Request) {
	items, err := s.customersSvc.All(r.Context())
	if errors.Is(err, customers.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(items)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGetCustomersAllActive(w http.ResponseWriter, r *http.Request) {
	items, err := s.customersSvc.AllActive(r.Context())
	if errors.Is(err, customers.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(items)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleGetCustomersSave(w http.ResponseWriter, r *http.Request) {
	var customer *customers.Customer
	var item *customers.CustomerAuth
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		log.Print("Can't Decode customer")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	fmt.Print(item)
	customer, err = s.customersSvc.Create(r.Context(), item)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(customer)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleCustomersRemoveByID(w http.ResponseWriter, r *http.Request) {
	idParam, ok := mux.Vars(r)["id"]
	if !ok {
		log.Print("Missing id")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	customer, err := s.customersSvc.RemoveByID(r.Context(), id)
	if errors.Is(err, customers.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(customer)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleCustomersBlockByID(w http.ResponseWriter, r *http.Request) {
	idParam, ok := mux.Vars(r)["id"]
	if !ok {
		log.Print("Cant parse id")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	customer, err := s.customersSvc.BlockByID(r.Context(), id)
	if errors.Is(err, customers.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(customer)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

}

func (s *Server) handleCustomersUnBlockByID(w http.ResponseWriter, r *http.Request) {
	idParam, ok := mux.Vars(r)["id"]
	if !ok {
		log.Print("Cant parse id")
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	customer, err := s.customersSvc.UnBlockByID(r.Context(), id)
	if errors.Is(err, customers.ErrNotFound) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(customer)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

}

func (s *Server) handleGetToken(w http.ResponseWriter, r *http.Request) {
	var auth *security.Auth
	var tok Token
	err := json.NewDecoder(r.Body).Decode(&auth)
	fmt.Print(auth)
	if err != nil {
		log.Print("Can't Decode login and password")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	log.Print("Login: ", auth.Login, "Password: ", auth.Password)

	token, err := s.customersSvc.TokenForCustomer(r.Context(), auth.Login, auth.Password)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	tok.Token = token
	data, err := json.Marshal(tok)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
func (s *Server) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	var fail ResponceFail
	var ok ResponceOk
	var token Token
	var data []byte
	code := 200

	err := json.NewDecoder(r.Body).Decode(&token)
	if err != nil {
		log.Print("Can't Decode token")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	id, er := s.securitySvc.AuthenticateCusomer(r.Context(), token.Token)

	if er == security.ErrNoSuchUser {
		code = 404
		fail.Status = "fail"
		fail.Reason = "not found"
	} else if er == security.ErrExpired {
		code = 400
		fail.Status = "fail"
		fail.Reason = "expired"
	} else if er == nil {
		log.Print(id)
		ok.Status = "ok"
		ok.CustomerID = id
	} else {
		log.Print("err", er)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if code != 200 {
		w.WriteHeader(code)

		data, err = json.Marshal(fail)
		if err != nil {
			log.Print(err)
		}
	} else {
		data, err = json.Marshal(ok)
		if err != nil {
			log.Print(err)
		}
	}
	_, err = w.Write(data)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	return
}
