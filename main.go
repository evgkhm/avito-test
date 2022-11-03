package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/matperez/go-cbr-client"
)

// User структура для записи/чтения данных с БД
type User struct {
	Id      int     `json:"id"`
	Balance float64 `json:"balance"`
	Name    string  `json:"name"`
}

// UserToUser структура для перевода средств от пользователя к пользователю
/*type UserToUser struct {
	IdWhoTransfer int     `json:"idWhoTransfer"`
	IdWhoReceive  int     `json:"idWhoReceive"`
	Balance       float64 `json:"balance"`
}*/

type UserReservationRevenue struct {
	Id        int     `json:"id"`
	IdService int     `json:"id_service"`
	IdOrder   int     `json:"id_order"`
	Cost      float64 `json:"cost"`
}

// revenue метод признания выручки – списывает из резерва
func (userDataFromRequest UserReservationRevenue) revenue(db *sql.DB, w http.ResponseWriter) error {
	dataFromDB := UserReservationRevenue{0, 0, 0, 0}
	//Метод для получения всех данных из DB
	rows, err := db.Query("select * from reservation") //начинаем парсить таблицу c резервациями
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	//Считывание данных
	for rows.Next() {
		err := rows.Scan(&dataFromDB.Id, &dataFromDB.IdService, &dataFromDB.IdOrder, &dataFromDB.Cost) //
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			//Вывод текущих данных
			//fmt.Fprintf(w, "у пользователя с id %d было %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)

			//Проверка, что есть такой пользователь в таблице зарезервированных
			if userDataFromRequest.Id != dataFromDB.Id {
				fmt.Fprintln(w, "нет такого пользователя, запрос отклонен")
			} else if userDataFromRequest.IdOrder != dataFromDB.IdOrder {
				fmt.Fprintln(w, "ID заказа не совпадает, запрос отклонен")
			} else if userDataFromRequest.IdService != dataFromDB.IdService {
				fmt.Fprintln(w, "ID услуги не совпадает, запрос отклонен")
			} else if userDataFromRequest.Cost != dataFromDB.Cost {
				fmt.Fprintln(w, "сумма не совпадает, запрос отклонен")
			} else {
				//Строка с sql запросом на обновление данных в основной таблице usr
				//sqlStr := `update "usr" set "balance"=$1 where "id"=$2`
				//Выполнение sql запроса
				//_, err := db.Exec(sqlStr, newBalance, dataFromDB.Id)
				//if err != nil {
				//	panic(err)
				//}

				//Строка с sql запросом на добавление данных в таблицу revenue
				/*sqlStr = `insert into "reservation" (id, id_service, id_order, cost) values (
				$1,$2,$3,$4)`*/
				sqlStr := `insert into "revenue" values ($1,$2,$3,$4)`
				_, err = db.Exec(sqlStr, userDataFromRequest.Id, userDataFromRequest.IdService, userDataFromRequest.IdOrder, userDataFromRequest.Cost)
				if err != nil {
					panic(err)
				}

				//Удаление строки резервации из таблицы reservation
				sqlStr = `delete from "reservation" where id = $1`
				_, err = db.Exec(sqlStr, userDataFromRequest.Id)
				if err != nil {
					panic(err)
				}

				//Вывод обновленных данных
				//Метод для получения обновленных данных из DB
				row := db.QueryRow("select * from revenue where id = $1", dataFromDB.Id)
				if err != nil {
					panic(err)
				}
				//Считывание данных
				err = row.Scan(&dataFromDB.Id, &dataFromDB.IdOrder, &dataFromDB.IdService, &dataFromDB.Cost)
				fmt.Fprintf(w, "признание выручки успешно, у пользователя с id %d списали %.2f руб с резерва\n", userDataFromRequest.Id, userDataFromRequest.Cost)

			}
		}
	}
	return err
}

// reservation метод резервирования средств с основного баланса на отдельном счете
func (userDataFromRequest UserReservationRevenue) reservation(db *sql.DB, w http.ResponseWriter) error {
	dataFromDB := User{0, 0, ""}
	//Метод для получения всех данных из DB
	rows, err := db.Query("select * from usr")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	//Считывание данных
	for rows.Next() {
		err := rows.Scan(&dataFromDB.Id, &dataFromDB.Balance, &dataFromDB.Name) //
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			//Проверяем что id который ввели совпадает с тем, что есть в DB
			if userDataFromRequest.Id == dataFromDB.Id { //TODO: перемесить в другое место!!!
				//Вывод текущих данных
				//fmt.Fprintf(w, "у пользователя с id %d было %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)
				//Определение нового баланса
				newBalance := dataFromDB.Balance - userDataFromRequest.Cost

				//Проверка того, что нельзя уйти в минус
				if newBalance < 0 {
					newBalance = dataFromDB.Balance
					fmt.Fprintln(w, "попытка уйти в минус, запрос на списания отклонен")
				} else {
					//Строка с sql запросом на обновление данных в основной таблице usr
					sqlStr := `update "usr" set "balance"=$1 where "id"=$2`
					//Выполнение sql запроса
					_, err := db.Exec(sqlStr, newBalance, dataFromDB.Id)
					if err != nil {
						panic(err)
					}

					//Строка с sql запросом на добавление данных в таблице reservation
					/*sqlStr = `insert into "reservation" (id, id_service, id_order, cost) values (
					$1,$2,$3,$4)`*/
					sqlStr = `insert into "reservation" values ($1,$2,$3,$4)`
					_, err = db.Exec(sqlStr, userDataFromRequest.Id, userDataFromRequest.IdService, userDataFromRequest.IdOrder, userDataFromRequest.Cost)
					if err != nil {
						panic(err)
					}

					//Вывод обновленных данных
					//Метод для получения обновленных данных из DB
					row := db.QueryRow("select * from usr where id = $1", dataFromDB.Id)
					if err != nil {
						panic(err)
					}
					//Считывание данных
					err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance, &dataFromDB.Name)
					fmt.Fprintf(w, "резервирование успешено, у пользователя с id %d стало %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)

				}
			}

		}
	}
	return err
}

// sum метод для списания/начисления средств в БД и вывод обратной связи
func (userDataFromRequest User) sum(db *sql.DB, w http.ResponseWriter) error {
	dataFromDB := User{0, 0, ""}
	//Метод для получения всех данных из DB
	rows, err := db.Query("select * from usr")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	//Считывание данных
	for rows.Next() {
		err := rows.Scan(&dataFromDB.Id, &dataFromDB.Balance, &dataFromDB.Name) //
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			//Проверяем что id который ввели совпадает с тем, что есть в DB
			if userDataFromRequest.Id == dataFromDB.Id {
				//Вывод текущих данных
				//fmt.Fprintf(w, "у пользователя с id %d было %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)

				//Проверка того, что нельзя уйти в минус
				if userDataFromRequest.Balance < 0 {
					//newBalance = dataFromDB.Balance
					fmt.Fprintln(w, "попытка начислить отрицательное число, запрос отклонен")
				} else if userDataFromRequest.Balance < 0.1 {
					fmt.Fprintln(w, "попытка начислить число меньше копейки, запрос отклонен")
				} else {
					//Определение нового баланса
					newBalance := dataFromDB.Balance + userDataFromRequest.Balance
					//Строка с sql запросом на обновление данных
					sqlStr := `update "usr" set "balance"=$1 where "id"=$2`
					//Выполнение sql запроса
					_, err := db.Exec(sqlStr, newBalance, dataFromDB.Id)
					if err != nil {
						panic(err)
					}

					//Вывод обновленных данных
					//Метод для получения обновленных данных из DB
					row := db.QueryRow("select * from usr where id = $1", dataFromDB.Id)
					if err != nil {
						panic(err)
					}
					//Считывание данных
					err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance, &dataFromDB.Name)
					fmt.Fprintf(w, "успешно, у пользователя с id %d стало %.2f руб\n", dataFromDB.Id, dataFromDB.Balance)
				}
			}
		}

	}
	return err
}

// getBalance метод для получения текущего баланса пользователя в БД и вывод обратной связи
func (userDataFromRequest User) getBalance(db *sql.DB, id int, w http.ResponseWriter, currency string) error {
	dataFromDB := User{0, 0, ""}
	//Получение всех данных из DB
	rows, err := db.Query("select * from usr")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	//Считывание данных
	for rows.Next() {
		err := rows.Scan(&dataFromDB.Id, &dataFromDB.Balance, &dataFromDB.Name) //
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			//Проверяем что id который ввели совпадает с тем, что есть в DB
			if id == dataFromDB.Id {
				//Метод для получения обновленных данных из DB
				row := db.QueryRow("select * from usr where id = $1", dataFromDB.Id)
				if err != nil {
					panic(err)
				}
				//Считывание данных
				err = row.Scan(&dataFromDB.Id, &dataFromDB.Balance, &dataFromDB.Name)

				//Доп. задание 1: конвертация баланса в доллары
				if currency == "USD" {
					client := cbr.NewClient()
					rate, err := client.GetRate("USD", time.Now())
					if err != nil {
						panic(err)
					}
					balanceUSD := dataFromDB.Balance / rate
					fmt.Fprintf(w, "у пользователя с id %d  %.2f $", dataFromDB.Id, balanceUSD)
				} else {
					fmt.Fprintf(w, "у пользователя с id %d  %.2f руб", dataFromDB.Id, dataFromDB.Balance)
				}
			}
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

	/*result, err := db.Exec("insert into usr (id, balance, name) values (2, 10, 'Ivan')")
	if err != nil {
		panic(err)
	}
	fmt.Println(result.LastInsertId()) // не поддерживается
	fmt.Println(result.RowsAffected()) // количество добавленных строк*/

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
			var userFromRequest User
			//Получение id с запроса
			id := r.URL.Query().Get("id")
			currency := r.URL.Query().Get("currency")
			iD, err := strconv.Atoi(id)
			if err != nil {
				io.WriteString(w, "ошибка, при парсинге данных, отмена запроса")
				return
			}
			err = userFromRequest.getBalance(db, iD, w, currency)
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
