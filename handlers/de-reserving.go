package handlers

import (
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"io"
	"net/http"
	"newNew/repository"
)

// ListenRequestDereserving метод для получения HTTP запроса для разрезервирования средств
func ListenRequestDereserving(db *sqlx.DB) {
	http.HandleFunc("/dereservation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			userFromRequest := repository.NewUserReservRev()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal(body, &userFromRequest)
			if err != nil {
				description := "attempt to parse data"
				err = repository.SendJsonAnswer(false, description, w)
				if err != nil {
					return
				}
				return
			}
			err = userFromRequest.Dereservation(db, w)
			if err != nil {
				description := "attempt to make a reservation"
				err = repository.SendJsonAnswer(false, description, w)
				if err != nil {
					return
				}
				return
			}
		}
	})
}
