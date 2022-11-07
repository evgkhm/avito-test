package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"net/http"
	"newNew/repository"
)

type UserReservationRevenue struct {
	Id        int     `json:"id"`
	IdService int     `json:"id_service"`
	IdOrder   int     `json:"id_order"`
	Cost      float64 `json:"cost"`
}

// ListenRequestReservation метод для получения HTTP запроса для резервирования средств
func ListenRequestReservation(db *sqlx.DB) {
	http.HandleFunc("/reservation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var userFromRequest UserReservationRevenue
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
			err = repository.UserReservationRevenue.Reservation(repository.UserReservationRevenue(userFromRequest), db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make a reservation")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}
