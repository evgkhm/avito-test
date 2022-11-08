package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx"
	"net/http"
	"os"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	DBName   string
	SSLMode  string
}

type Postgres struct {
}

type User struct {
	Id      int     `json:"id"`
	Balance float64 `json:"balance"`
}

type jsonResponse struct {
	Result      bool
	Description string
}

type UserReservationRevenue struct {
	Id        int     `json:"id"`
	IdService int     `json:"id_service"`
	IdOrder   int     `json:"id_order"`
	Cost      float64 `json:"cost"`
}

func NewRepository(db *sqlx.DB) *Postgres {
	return &Postgres{}
}

func NewUser() *User {
	return &User{}
}

func NewUserReservRev() *UserReservationRevenue {
	return &UserReservationRevenue{}
}

type Repository interface {
	Sum(db *sqlx.DB, w http.ResponseWriter) error
	Reservation(db *sqlx.DB, w http.ResponseWriter) error
	Revenue(db *sqlx.DB, w http.ResponseWriter) error
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

// NewPostgresDB открытие ДБ, данные для входа из .env файла
func NewPostgresDB(cfg Config) (db *sqlx.DB, err error) {
	db, err = sqlx.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USERNAME"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_SSLMODE")))
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return db, err
}

// GetBalance метод для получения текущего баланса пользователя в БД и вывод обратной связи
func GetBalance(db *sqlx.DB, id int, w http.ResponseWriter) error {
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

// Revenue метод для признания средств из резерва
func (dataRequest *UserReservationRevenue) Revenue(db *sqlx.DB, w http.ResponseWriter) error {
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

// Reservation метод резервирования средств с основного баланса на отдельном счете
func (dataRequest *UserReservationRevenue) Reservation(db *sqlx.DB, w http.ResponseWriter) error {
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

// Sum метод для начисления средств в БД
func (dataRequest *User) Sum(db *sqlx.DB, w http.ResponseWriter) error {
	dataDB := User{0, 0}
	var err error
	//ищем нужный ID из БД
	if err = db.QueryRow("select * from usr where id = $1", dataRequest.Id).Scan(&err); err != nil {
		if err == sql.ErrNoRows { //пользователя нет в БД, нужно добавить
			//Строка с sql запросом на добавление данных в таблицу usr
			sqlStr := `insert into "usr" values ($1,$2)`
			_, err = db.Exec(sqlStr, dataRequest.Id, dataRequest.Balance)
			if err != nil {
				panic(err)
			}
			//Получение обновленных данных из DB
			row := db.QueryRow("select * from usr where id = $1", dataRequest.Id)

			//Считывание данных
			err = row.Scan(&dataDB.Id, &dataDB.Balance)
			description := fmt.Sprintf("user id = %d now has %.2f", dataDB.Id, dataDB.Balance)
			err = sendJsonAnswer(true, description, w)
		} else { //пользователь есть в БД, нужно обновить баланс
			row := db.QueryRow("select * from usr where id = $1", dataRequest.Id)
			err = row.Scan(&dataDB.Id, &dataDB.Balance)
			if dataRequest.Balance < 0 {
				description := fmt.Sprint("attempt to add a negative number")
				err = sendJsonAnswer(false, description, w)
			} else if dataRequest.Balance < 0.1 {
				description := fmt.Sprint("attempt to add a number less than a penny")
				err = sendJsonAnswer(false, description, w)
			} else {
				//Определение нового баланса
				newBalance := dataDB.Balance + dataRequest.Balance
				sqlStr := `update "usr" set "balance"=$1 where "id"=$2`
				//Выполнение sql запроса
				_, err = db.Exec(sqlStr, newBalance, dataDB.Id)
				if err != nil {
					panic(err)
				}
				//Получение обновленных данных из DB
				row = db.QueryRow("select * from usr where id = $1", dataRequest.Id)
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

// Dereservation метод резервирования средств с основного баланса на отдельном счете
func (dataRequest *UserReservationRevenue) Dereservation(db *sqlx.DB, w http.ResponseWriter) error {
	dataDB := UserReservationRevenue{0, 0, 0, 0}
	var err error
	if err = db.QueryRow("select * from reservation where id = $1 and id_service = $2 and id_order = $3 and cost = $4",
		dataRequest.Id, &dataRequest.IdService, dataRequest.IdOrder, &dataRequest.Cost).Scan(&err); err != nil {
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
		//Удаление строки резервации из таблицы reservation
		sqlStr := `delete from "reservation" where id = $1 and id_service = $2 and id_order = $3 and cost = $4`
		_, err = db.Exec(sqlStr, dataRequest.Id, dataRequest.IdService, dataRequest.IdOrder, dataRequest.Cost)
		if err != nil {
			panic(err)
		}

		//получение данных из БД usr
		dataDereserv := User{0, 0}
		row = db.QueryRow("select * from usr where id = $1", dataRequest.Id)
		err = row.Scan(&dataDereserv.Id, &dataDereserv.Balance)
		if err != nil {
			panic(err)
		}

		newBalance := dataDereserv.Balance + dataDB.Cost

		//Строка с sql запросом на обновление данных в основной таблице usr
		sqlStr = `update "usr" set "balance"=$1 where "id"=$2`
		_, err = db.Exec(sqlStr, newBalance, dataDB.Id)
		if err != nil {
			panic(err)
		}

		//Метод для получения обновленных данных из DB
		row = db.QueryRow("select * from usr where id = $1", dataDereserv.Id)
		if err != nil {
			panic(err)
		}
		//Считывание данных
		err = row.Scan(&dataDereserv.Id, &dataDereserv.Balance)
		description := fmt.Sprintf("user id = %d now has %.2f", dataDereserv.Id, dataDereserv.Balance)
		err = sendJsonAnswer(true, description, w)
	}
	return err
}
