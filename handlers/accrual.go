package handlers

import (
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"io"
	"net/http"
	"newNew/repository"
)

// ListenRequestSum метод для получения HTTP запроса начисления средств
func ListenRequestSum(db *sqlx.DB) {
	http.HandleFunc("/sum", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			UserFromRequest := repository.NewUser()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal(body, &UserFromRequest)
			if err != nil {
				description := "attempt to parse data"
				err = repository.SendJsonAnswer(false, description, w)
				if err != nil {
					return
				}
				return
			}
			err = UserFromRequest.Sum(db, w)
			if err != nil {
				description := "attempt to make accrual"
				err = repository.SendJsonAnswer(false, description, w)
				if err != nil {
					return
				}
				return
			}
		}
	})
}
