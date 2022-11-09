package handlers

import (
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"io"
	"net/http"
	"newNew/repository"
)

// ListenRequestRevenue метод для получения HTTP запроса для принятия средств из резерва
func ListenRequestRevenue(db *sqlx.DB) {
	//Хэндл для начисления и списания
	http.HandleFunc("/revenue", func(w http.ResponseWriter, r *http.Request) {
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
			err = userFromRequest.Revenue(db, w)
			if err != nil {
				description := "attempt to make a revenue"
				err = repository.SendJsonAnswer(false, description, w)
				if err != nil {
					return
				}
				return
			}
		}
	})
}
