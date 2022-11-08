package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"net/http"
	"newNew/repository"
)

// ListenRequestSum метод для получения HTTP запроса начисления средств
func ListenRequestSum(db *sqlx.DB) {
	http.HandleFunc("/sum", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			UserFromRequest := repository.NewUser()
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal(body, &UserFromRequest)
			if err != nil {
				description := fmt.Sprint("attempt to parse data")
				err = sendJsonAnswer(false, description, w)
				return
			}
			err = UserFromRequest.Sum(db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make accrual")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}
