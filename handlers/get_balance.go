package handlers

import (
	"github.com/jmoiron/sqlx"
	"net/http"
	"newNew/repository"
	"strconv"
)

// ListenRequestGetBalance метод для получения HTTP запроса получения текущего баланса пользователя
func ListenRequestGetBalance(db *sqlx.DB) {
	//Хэндл для начисления и списания
	http.HandleFunc("/gb", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			//Получение id с запроса
			id, err := strconv.Atoi(r.URL.Query().Get("id"))
			if err != nil {
				description := "attempt to parse data"
				err = repository.SendJsonAnswer(false, description, w)
				if err != nil {
					return
				}
				return
			}
			err = repository.GetBalance(db, id, w)
			if err != nil {
				description := "attempt to get balance"
				err = repository.SendJsonAnswer(false, description, w)
				if err != nil {
					return
				}
				return
			}
		}
	})
}
