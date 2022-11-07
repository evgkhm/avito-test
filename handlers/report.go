package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/jmoiron/sqlx"
	"net/http"
	"os"
	"strconv"
	"time"
)

type UserReport struct {
	UserData UserReservationRevenue
	Year     int `json:"year"`
	Month    int `json:"month"`
}

// getReport метод для получения данных из БД для добавления в мап файл
func getReport(db *sqlx.DB, year int, month int, w http.ResponseWriter) (map[int]float64, error) {
	dataDB := UserReport{}
	res := make(map[int]float64) //мапа с данными из БД
	rows, err := db.Query("select * from revenue where extract(year from curr_date) = $1 and extract(month from curr_date) = $2", year, month)
	if err != nil {
		description := fmt.Sprint("attempt reading from database")
		err = sendJsonAnswer(false, description, w)
		return res, err
	}
	defer func(rows *sql.Rows) {
		err = rows.Close()
		if err != nil {

		}
	}(rows)

	//Добавление в хэш-таблицу данных с БД
	for rows.Next() {
		var stamp time.Time
		err = rows.Scan(&dataDB.UserData.Id, &dataDB.UserData.IdService, &dataDB.UserData.IdOrder, &dataDB.UserData.Cost, &stamp)
		dataDB.Year = stamp.Year()
		dataDB.Month = int(stamp.Month())
		res[dataDB.UserData.IdService] += dataDB.UserData.Cost
	}
	return res, err
}

// createReportCSV функция создает csv файл отсчета из мап файла
func createReportCSV(data map[int]float64, w http.ResponseWriter) error {
	csvfile, err := os.Create("report.csv")
	if err != nil {
		description := fmt.Sprint("attempt to create csv file")
		err = sendJsonAnswer(false, description, w)
		return err
	}
	cswWriter := csv.NewWriter(csvfile)

	for key, value := range data {
		str1 := "название услуги"
		str2 := strconv.Itoa(key)
		str3 := "общая сумма выручки за отчетный период"
		str4 := strconv.FormatFloat(value, 'f', 2, 64)

		var res []string
		res = append(res, str1)
		res = append(res, str2)
		res = append(res, str3)
		res = append(res, str4)
		err = cswWriter.Write(res)
		if err != nil {
			description := fmt.Sprint("attempt to write to csv file")
			err = sendJsonAnswer(false, description, w)
			return err
		}
	}
	cswWriter.Flush()

	err = csvfile.Close()
	if err != nil {
		description := fmt.Sprint("attempt to close csv file")
		err = sendJsonAnswer(false, description, w)
		return err
	}

	description := fmt.Sprint("csv file created")
	err = sendJsonAnswer(true, description, w)
	return err
}

// ListenRequestReport метод для получения месячного отсчета
func ListenRequestReport(db *sqlx.DB) {
	//Хэндл для отсчета
	http.HandleFunc("/getReport", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			//Получение года с запроса
			year, err := strconv.Atoi(r.URL.Query().Get("year"))
			if err != nil {
				description := fmt.Sprint("attempt to parse data")
				err = sendJsonAnswer(false, description, w)
				return
			}
			if year < 1975 || year > 2030 {
				description := fmt.Sprint("wrong year")
				err = sendJsonAnswer(false, description, w)
				return
			}
			//Получение месяца
			month, err := strconv.Atoi(r.URL.Query().Get("month"))
			if err != nil {
				description := fmt.Sprint("attempt to parse data")
				err = sendJsonAnswer(false, description, w)
				return
			}
			if month < 0 || month > 12 {
				description := fmt.Sprint("wrong month")
				err = sendJsonAnswer(false, description, w)
				return
			}
			//Создание хэш-таблицы с данными для отсчета
			reportMap := make(map[int]float64)
			reportMap, err = getReport(db, year, month, w)
			if err != nil {
				description := fmt.Sprint("attempt to get data from database")
				err = sendJsonAnswer(false, description, w)
				return
			}
			err = createReportCSV(reportMap, w) //создание csv файла
		}
	})
}
