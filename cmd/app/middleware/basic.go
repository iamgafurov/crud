package middleware

import (
	"log"
	"net/http"
)

func Basic(auth func(login, pass string) bool) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			username, password, ok := request.BasicAuth()
			if !ok {
				log.Print("Cant parse username and password")
				http.Error(writer, http.StatusText(401), 401)
				return
			}
			if !auth(username, password) {
				http.Error(writer, http.StatusText(401), 401)
				return
			}
			handler.ServeHTTP(writer, request)
		})
	}
}
