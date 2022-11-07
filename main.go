package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type User struct {
	Id      int     `json:"id"`
	Balance float64 `json:"balance"`
}

type UserReservationRevenue struct {
	Id        int     `json:"id"`
	IdService int     `json:"id_service"`
	IdOrder   int     `json:"id_order"`
	Cost      float64 `json:"cost"`
}

type UserReport struct {
	UserData UserReservationRevenue
	Year     int `json:"year"`
	Month    int `json:"month"`
}

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

// revenue метод признания выручки – списывает из резерва
func (dataRequest UserReservationRevenue) revenue(db *sql.DB, w http.ResponseWriter) error {
	dataDB := UserReservationRevenue{0, 0, 0, 0}
	var err error
	if err = db.QueryRow("select * from reservation where id = $1 and id_service = $2 and id_order = $3 and cost = $4",
		dataRequest.Id, &dataRequest.IdService, dataRequest.IdOrder, &dataRequest.Cost).Scan(&err); err != nil {
		//проверка того,что пользователя нет в БД
		if err == sql.ErrNoRows {
			description := fmt.Sprintf("no data for id = %d", dataRequest.Id)
			err = sendJsonAnswer(false, description, w)
			return err
		}
		//получение данных из БД
		row := db.QueryRow("select * from reservation where id = $1 and id_service = $2 and id_order = $3 and cost = $4",
			dataRequest.Id, &dataRequest.IdService, dataRequest.IdOrder, &dataRequest.Cost)
		err = row.Scan(&dataDB.Id, &dataDB.IdService, &dataDB.IdOrder, &dataDB.Cost)
		//Проверка, что есть такой пользователь в таблице зарезервированных
		if dataRequest.Id != dataDB.Id {
			description := fmt.Sprintf("no data for id = %d", dataRequest.Id)
			err = sendJsonAnswer(false, description, w)
			return err
		} else if dataRequest.IdOrder != dataDB.IdOrder {
			description := fmt.Sprintf("no data for id_order = %d", dataRequest.IdOrder)
			err = sendJsonAnswer(false, description, w)
			return err
		} else if dataRequest.IdService != dataDB.IdService {
			description := fmt.Sprintf("no data for id_service = %d", dataRequest.IdService)
			err = sendJsonAnswer(false, description, w)
			return err
		} else if dataRequest.Cost != dataDB.Cost {
			description := fmt.Sprintf("no data for cost = %f", dataRequest.Cost)
			err = sendJsonAnswer(false, description, w)
			return err
		}
		sqlStr := `insert into "revenue" values ($1,$2,$3,$4)`
		_, err = db.Exec(sqlStr, dataRequest.Id, dataRequest.IdService, dataRequest.IdOrder, dataRequest.Cost)
		if err != nil {
			panic(err)
		}

		//Удаление строки резервации из таблицы reservation
		sqlStr = `delete from "reservation" where id = $1 and id_service = $2 and id_order = $3 and cost = $4`
		_, err = db.Exec(sqlStr, dataRequest.Id, dataRequest.IdService, dataRequest.IdOrder, dataRequest.Cost)
		if err != nil {
			panic(err)
		}

		//Метод для получения обновленных данных из DB
		row = db.QueryRow("select * from revenue where id = $1", dataDB.Id)
		if err != nil {
			panic(err)
		}
		//Считывание данных
		err = row.Scan(&dataDB.Id, &dataDB.IdOrder, &dataDB.IdService, &dataDB.Cost)
		description := fmt.Sprintf("user id = %d was debited %.2f from reservation", dataDB.Id, dataRequest.Cost)
		err = sendJsonAnswer(true, description, w)
	}
	return err
}

// reservation метод резервирования средств с основного баланса на отдельном счете
func (dataRequest UserReservationRevenue) reservation(db *sql.DB, w http.ResponseWriter) error {
	dataDB := User{0, 0}
	var err error
	if err = db.QueryRow("select * from usr where id = $1", dataRequest.Id).Scan(&err); err != nil {
		if err == sql.ErrNoRows { //пользователя нет в БД
			description := fmt.Sprintf("no data for id = %d", dataRequest.Id)
			err = sendJsonAnswer(false, description, w)
			return err
		}
		//получение данных из БД
		row := db.QueryRow("select * from usr where id = $1", dataRequest.Id)
		err = row.Scan(&dataDB.Id, &dataDB.Balance)

		newBalance := dataDB.Balance - dataRequest.Cost
		//Проверка того, что нельзя уйти в минус
		if newBalance < 0 {
			description := fmt.Sprint("attempt to go into the negative")
			err = sendJsonAnswer(false, description, w)
			return err
		}
		//Добавление в таблицу reservation данных
		sqlStr := `insert into "reservation" values ($1,$2,$3,$4) on conflict do nothing`
		var res sql.Result
		res, err = db.Exec(sqlStr, dataRequest.Id, dataRequest.IdService, dataRequest.IdOrder, dataRequest.Cost)
		if err != nil {
			panic(err)
		} else if n, _ := res.RowsAffected(); n != 1 { //нет добавления в таблицу коллизия данных
			description := fmt.Sprint("data collision")
			err = sendJsonAnswer(false, description, w)
			return err
		}

		//Строка с sql запросом на обновление данных в основной таблице usr
		sqlStr = `update "usr" set "balance"=$1 where "id"=$2`
		_, err = db.Exec(sqlStr, newBalance, dataDB.Id)
		if err != nil {
			panic(err)
		}
		//вывод обновленных данных из таблицы usr
		row = db.QueryRow("select * from usr where id = $1", dataDB.Id)
		if err != nil {
			panic(err)
		}
		//Считывание данных
		err = row.Scan(&dataDB.Id, &dataDB.Balance)
		description := fmt.Sprintf("user id = %d now has %.2f", dataDB.Id, dataDB.Balance)
		err = sendJsonAnswer(true, description, w)
	}
	return err
}

// sum метод для начисления средств в БД
func (userDataFromRequest User) sum(db *sql.DB, w http.ResponseWriter) error {
	dataDB := User{0, 0}
	var err error
	//ищем нужный ID из БД
	if err = db.QueryRow("select * from usr where id = $1", userDataFromRequest.Id).Scan(&err); err != nil {
		if err == sql.ErrNoRows { //пользователя нет в БД, нужно добавить
			//Строка с sql запросом на добавление данных в таблицу usr
			sqlStr := `insert into "usr" values ($1,$2)`
			_, err = db.Exec(sqlStr, userDataFromRequest.Id, userDataFromRequest.Balance)
			if err != nil {
				panic(err)
			}
			//Получение обновленных данных из DB
			row := db.QueryRow("select * from usr where id = $1", userDataFromRequest.Id)

			//Считывание данных
			err = row.Scan(&dataDB.Id, &dataDB.Balance)
			description := fmt.Sprintf("user id = %d now has %.2f", dataDB.Id, dataDB.Balance)
			err = sendJsonAnswer(true, description, w)
		} else { //пользователь есть в БД, нужно обновить баланс
			row := db.QueryRow("select * from usr where id = $1", userDataFromRequest.Id)
			err = row.Scan(&dataDB.Id, &dataDB.Balance)
			if userDataFromRequest.Balance < 0 {
				description := fmt.Sprint("attempt to add a negative number")
				err = sendJsonAnswer(false, description, w)
			} else if userDataFromRequest.Balance < 0.1 {
				description := fmt.Sprint("attempt to add a number less than a penny")
				err = sendJsonAnswer(false, description, w)
			} else {
				//Определение нового баланса
				newBalance := dataDB.Balance + userDataFromRequest.Balance
				sqlStr := `update "usr" set "balance"=$1 where "id"=$2`
				//Выполнение sql запроса
				_, err = db.Exec(sqlStr, newBalance, dataDB.Id)
				if err != nil {
					panic(err)
				}
				//Получение обновленных данных из DB
				row = db.QueryRow("select * from usr where id = $1", userDataFromRequest.Id)
				//Считывание данных
				err = row.Scan(&dataDB.Id, &dataDB.Balance)

				//Вывод обновленных данных
				description := fmt.Sprintf("user id = %d now has %.2f", dataDB.Id, dataDB.Balance)
				err = sendJsonAnswer(true, description, w)
			}
		}
	}
	return err
}

// getBalance метод для получения текущего баланса пользователя в БД и вывод обратной связи
func getBalance(db *sql.DB, id int, w http.ResponseWriter) error {
	dataDB := User{0, 0}
	var err error
	if err = db.QueryRow("select * from usr where id = $1", id).Scan(&err); err != nil {
		if err == sql.ErrNoRows { //пользователя нет в БД
			description := fmt.Sprintf("no data for id = %d", id)
			err = sendJsonAnswer(false, description, w)
			return err
		}
		row := db.QueryRow("select * from usr where id = $1", id)
		err = row.Scan(&dataDB.Id, &dataDB.Balance)

		description := fmt.Sprintf("id = %d, balance = %.2f", dataDB.Id, dataDB.Balance)
		err = sendJsonAnswer(true, description, w)
	}
	return err
}

// getBalance метод для получения текущего баланса пользователя в БД и вывод обратной связи
func getReport(db *sql.DB, year int, month int, w http.ResponseWriter) (map[int]float64, error) {
	dataDB := UserReport{}
	res := make(map[int]float64) //мапа с данными из БД
	//var usersCount uint64              //число пользователей из БД для добавления в мапу

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

// createReportCSV функция создает csv файл отсчета
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

func main() {
	//Создание сервера
	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	startHttpServer(httpServerExitDone)

	//Соединение с БД
	db, err := sql.Open("postgres", "postgres://admin:admin@host.docker.internal:5436/users?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer func(db *sql.DB) {
		err = db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	//Создание хэндла для начисления средств
	listenRequestSum(db)

	//Создание хэндла для резервирования средств
	listenRequestReservation(db)

	//Создание хэндла для получения баланса пользователя
	listenRequestGetBalance(db)

	//Создание хэндла для признания выручки пользователя
	listenRequestRevenue(db)

	//Создание хэндла для месячного отсчета
	listenRequestReport(db)
	select {}
}

// listenRequestReport метод для получения месячного отсчета
func listenRequestReport(db *sql.DB) {
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

// listenRequestReservation метод для получения HTTP запроса для резервирования средств
func listenRequestRevenue(db *sql.DB) {
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
			err = userFromRequest.revenue(db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make a revenue")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}

// listenRequestReservation метод для получения HTTP запроса для резервирования средств
func listenRequestReservation(db *sql.DB) {
	//Хэндл для начисления и списания
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
			err = userFromRequest.reservation(db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make a reservation")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}

// listenRequestSum метод для получения HTTP запроса начисления средств
func listenRequestSum(db *sql.DB) {
	//Хэндл для начисления и списания
	http.HandleFunc("/sum", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			var userFromRequest User
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
			err = userFromRequest.sum(db, w)
			if err != nil {
				description := fmt.Sprint("attempt to make accrual")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}

// listenRequestGetBalance метод для получения HTTP запроса получения текущего баланса пользователя
func listenRequestGetBalance(db *sql.DB) {
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
			err = getBalance(db, id, w)
			if err != nil {
				description := fmt.Sprint("attempt to get balance")
				err = sendJsonAnswer(false, description, w)
				return
			}
		}
	})
}

// startHttpServer старт http сервера
func startHttpServer(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{Addr: ":8080"}

	go func() {
		defer wg.Done() // let main know we are done cleaning up

		// always returns error. ErrServerClosed on graceful close
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// unexpected error. port in use?
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
