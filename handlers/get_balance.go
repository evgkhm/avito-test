package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"net/http"
	"newNew/repository"
	"strconv"
)

type jsonResponse struct {
	Result      bool
	Description string
}

// sendJsonAnswer получает результат работы, описание и отправляет сообщение в json формате
func sendJsonAnswer(result bool, description string, w http.ResponseWriter) error {
	var data jsonResponse
	data.Result, data.Description = result, description

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(w, string(jsonData))
	return err
}

// ListenRequestGetBalance метод для получения HTTP запроса получения текущего баланса пользователя
func ListenRequestGetBalance(db *sqlx.DB) {
	//Хэндл для начисления и списания
	http.HandleFunc("/gb", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			//Получение id с запроса
			id, err := strconv.Atoi(r.URL.Query().Get("id"))
			if err != nil {
				description := fmt.Sprint("attempt to parse data")
				err = sendJsonAnswer(false, description, w)
				return
			}
			err = repository.GetBalance(db, id, w)
			if err != nil {
				description := fmt.Sprint("attempt to get balance")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}
