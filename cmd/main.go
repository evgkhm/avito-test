package main

import (
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"newNew/handlers"
	"newNew/repository"
	"os"
	"sync"
)

func main() {
	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	startHttpServer(httpServerExitDone)

	if err := godotenv.Load("./.env"); err != nil { //"../.env" for local using
		log.Fatalf("error loading env variables :%s", err.Error())
	}
	db, err := repository.NewPostgresDB(repository.Config{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     os.Getenv("POSTGRES_PORT"),
		Username: os.Getenv("POSTGRES_USERNAME"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DBName:   os.Getenv("POSTGRES_DB"),
		SSLMode:  os.Getenv("POSTGRES_SSLMODE"),
	})

	repository.NewRepository(db)

	if err != nil {
		panic(err)
	}
	defer func(db *sqlx.DB) {
		err = db.Close()
		if err != nil {
			panic(err)
		}
	}(db)

	//Создание хэндла для начисления средств
	handlers.ListenRequestSum(db)

	//Создание хэндла для резервирования средств
	handlers.ListenRequestReservation(db)

	//Создание хэндла для получения баланса пользователя
	handlers.ListenRequestGetBalance(db)

	//Создание хэндла для признания выручки пользователя
	handlers.ListenRequestRevenue(db)

	//Создание хэндла для месячного отсчета
	handlers.ListenRequestReport(db)

	//Создание хэндла для разрезервирования
	handlers.ListenRequestDereserving(db)

	select {}
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
