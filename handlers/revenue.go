package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"net/http"
	"newNew/repository"
)

// ListenRequestRevenue метод для получения HTTP запроса для принятия средств из резерва
func ListenRequestRevenue(db *sqlx.DB) {
	//Хэндл для начисления и списания
	http.HandleFunc("/revenue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var userFromRequest UserReservationRevenue
			body, err := ioutil.ReadAll(r.Body) //можно создать отдельную функцию для обоих хэндлов
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal(body, &userFromRequest)
			if err != nil {
				description := fmt.Sprint("attempt to parse data")
				err = sendJsonAnswer(false, description, w)
				return
			}
			err = repository.UserReservationRevenue.Revenue(repository.UserReservationRevenue(userFromRequest), db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make a revenue")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}
