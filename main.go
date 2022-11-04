package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
)

// User структура для записи/чтения данных с БД
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

// revenue метод признания выручки – списывает из резерва
func (dataRequest UserReservationRevenue) revenue(db *sql.DB, w http.ResponseWriter) error {
	dataDB := UserReservationRevenue{0, 0, 0, 0}
	var err error
	if err = db.QueryRow("select * from reservation where id = $1 and id_service = $2 and id_order = $3 and cost = $4",
		dataRequest.Id, &dataRequest.IdService, dataRequest.IdOrder, &dataRequest.Cost).Scan(&err); err != nil {
		//проверка того,что пользователя нет в БД
		if err == sql.ErrNoRows {
			fmt.Fprintf(w, "нет данных о id %d ", dataRequest.Id)
			return err
		}
		//получение данных из БД
		row := db.QueryRow("select * from reservation where id = $1 and id_service = $2 and id_order = $3 and cost = $4",
			dataRequest.Id, &dataRequest.IdService, dataRequest.IdOrder, &dataRequest.Cost)
		err = row.Scan(&dataDB.Id, &dataDB.IdService, &dataDB.IdOrder, &dataDB.Cost)
		//Проверка, что есть такой пользователь в таблице зарезервированных
		if dataRequest.Id != dataDB.Id {
			fmt.Fprintln(w, "нет такого пользователя, запрос отклонен")
			return err
		} else if dataRequest.IdOrder != dataDB.IdOrder {
			fmt.Fprintln(w, "ID заказа не совпадает, запрос отклонен")
			return err
		} else if dataRequest.IdService != dataDB.IdService {
			fmt.Fprintln(w, "ID услуги не совпадает, запрос отклонен")
			return err
		} else if dataRequest.Cost != dataDB.Cost {
			fmt.Fprintln(w, "сумма не совпадает, запрос отклонен")
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
		fmt.Fprintf(w, "признание выручки успешно, у пользователя с id %d списали %.2f руб с резерва\n", dataRequest.Id, dataRequest.Cost)
	}
	return err
}

// reservation метод резервирования средств с основного баланса на отдельном счете
func (dataRequest UserReservationRevenue) reservation(db *sql.DB, w http.ResponseWriter) error {
	dataFromDB := User{0, 0}
	var err error
	if err = db.QueryRow("select * from usr where id = $1", dataRequest.Id).Scan(&err); err != nil {
		if err == sql.ErrNoRows { //пользователя нет в БД
			fmt.Fprintf(w, "нет данных о id %d ", dataRequest.Id)
			return err
		}
		//получение данных из БД
		row := db.QueryRow("select * from usr where id = $1", dataRequest.Id)
		err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance)

		newBalance := dataFromDB.Balance - dataRequest.Cost
		//Проверка того, что нельзя уйти в минус
		if newBalance < 0 {
			fmt.Fprintln(w, "попытка уйти в минус, запрос на списания отклонен")
			return err
		}
		//Добавление в таблицу reservation данных
		sqlStr := `insert into "reservation" values ($1,$2,$3,$4) on conflict do nothing`
		var res sql.Result
		res, err = db.Exec(sqlStr, dataRequest.Id, dataRequest.IdService, dataRequest.IdOrder, dataRequest.Cost)
		if err != nil {
			panic(err)
		} else if n, _ := res.RowsAffected(); n != 1 { //нет добавления в таблицу коллизия данных
			io.WriteString(w, "ошибка, при выполнении резервировании средств, коллизия данных")
			return err
		}

		//Строка с sql запросом на обновление данных в основной таблице usr
		sqlStr = `update "usr" set "balance"=$1 where "id"=$2`
		_, err = db.Exec(sqlStr, newBalance, dataFromDB.Id)
		if err != nil {
			panic(err)
		}
		//вывод обновленных данных из таблицы usr
		row = db.QueryRow("select * from usr where id = $1", dataFromDB.Id)
		if err != nil {
			panic(err)
		}
		//Считывание данных
		err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance)
		fmt.Fprintf(w, "резервирование успешно, у пользователя с id %d стало %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)
	}
	return err
}

// sum метод для начисления средств в БД
func (userDataFromRequest User) sum(db *sql.DB, w http.ResponseWriter) error {
	dataFromDB := User{0, 0}
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
			err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance)
			fmt.Fprintf(w, "успешно, у пользователя с id %d стало %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)
		} else { //пользователь есть в БД, нужно обновить баланс
			row := db.QueryRow("select * from usr where id = $1", userDataFromRequest.Id)
			err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance)
			if userDataFromRequest.Balance < 0 {
				fmt.Fprintln(w, "попытка начислить отрицательное число, запрос отклонен")
			} else if userDataFromRequest.Balance < 0.1 {
				fmt.Fprintln(w, "попытка начислить число меньше копейки, запрос отклонен")
			} else {
				//Определение нового баланса
				newBalance := dataFromDB.Balance + userDataFromRequest.Balance
				sqlStr := `update "usr" set "balance"=$1 where "id"=$2`
				//Выполнение sql запроса
				_, err = db.Exec(sqlStr, newBalance, dataFromDB.Id)
				if err != nil {
					panic(err)
				}
				//Получение обновленных данных из DB
				row = db.QueryRow("select * from usr where id = $1", userDataFromRequest.Id)
				//Считывание данных
				err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance)
				fmt.Fprintf(w, "успешно, у пользователя с id %d стало %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)
			}
		}
	}
	return err
}

// getBalance метод для получения текущего баланса пользователя в БД и вывод обратной связи
func getBalance(db *sql.DB, id int, w http.ResponseWriter) error {
	dataFromDB := User{0, 0}

	var err error
	if err = db.QueryRow("select * from usr where id = $1", id).Scan(&err); err != nil {
		if err == sql.ErrNoRows { //пользователя нет в БД
			fmt.Fprintf(w, "нет данных о id %d ", id)
		} else {
			row := db.QueryRow("select * from usr where id = $1", id)
			err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance)
			fmt.Fprintf(w, "у пользователя с id %d  %.2f руб", dataFromDB.Id, dataFromDB.Balance)
		}
	}

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
	defer db.Close()

	//Создание хэндла для начисления средств
	listenRequestSum(db)

	//Создание хэндла для резервирования средств
	listenRequestReservation(db)

	//Создание хэндла для получения баланса пользователя
	listenRequestGetBalance(db)

	//Создание хэндла для признания выручки пользователя
	listenRequestRevenue(db)
	select {}
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
				io.WriteString(w, "ошибка, при парсинге данных, отмена запроса\n")
				return
			}
			err = userFromRequest.revenue(db, w)
			if err != nil {
				io.WriteString(w, "ошибка, при выполнении резервировании средств")
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
			body, err := ioutil.ReadAll(r.Body) //можно создать отдельную функцию для обоих хэндлов
			if err != nil {
				panic(err)
			}
			err = json.Unmarshal(body, &userFromRequest)
			if err != nil {
				io.WriteString(w, "ошибка, при парсинге данных, отмена запроса\n")
				return
			}
			err = userFromRequest.reservation(db, w)
			if err != nil {
				io.WriteString(w, "ошибка, при выполнении резервировании средств")
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
				io.WriteString(w, "ошибка, при парсинге данных, отмена запроса")
				return
			}

			err = userFromRequest.sum(db, w)
			if err != nil {
				io.WriteString(w, "ошибка, при выполнении зачислении/списания средств")
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
				io.WriteString(w, "ошибка, при парсинге данных, отмена запроса")
				return
			}
			err = getBalance(db, id, w)
			if err != nil {
				io.WriteString(w, "ошибка, при получении баланса пользователя")
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
