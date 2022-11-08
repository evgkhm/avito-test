package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"net/http"
	"newNew/repository"
)

// ListenRequestDereserving метод для получения HTTP запроса для разрезервирования средств
func ListenRequestDereserving(db *sqlx.DB) {
	http.HandleFunc("/dereservation", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			//var userFromRequest UserReservationRevenue
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
			//err = repository.UserReservationRevenue.Dereservation(repository.UserReservationRevenue(userFromRequest), db, w)
			err = userFromRequest.Dereservation(db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make a reservation")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}
