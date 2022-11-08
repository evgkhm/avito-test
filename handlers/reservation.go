package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"net/http"
	"newNew/repository"
)

// ListenRequestReservation метод для получения HTTP запроса для резервирования средств
func ListenRequestReservation(db *sqlx.DB) {
	http.HandleFunc("/reservation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			userFromRequest := repository.NewUserReservRev()
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal(body, &userFromRequest)
			if err != nil {
				description := fmt.Sprint("attempt to parse data")
				err = sendJsonAnswer(false, description, w)
				return
			}
			err = userFromRequest.Reservation(db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make a reservation")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}
