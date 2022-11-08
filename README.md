# Микросервис для работы с балансом пользователей
## Реализовано:
1. Метод начисления средств на баланс,
2. Метод резервирования средств,
3. Метод признания выручки,
4. Метод получения баланса пользователя,
5. Доп. задание 1 (реализовать метод для получения месячного отчета).

## Как запустить?
1. `git clone https://github.com/evgkhm/golang_http_postgresql-main`
2. `cd golang_http_postgresql-main`
3. `docker-compose up --build`  
4. При первом запуске может потребуется прописать миграцию БД
`migrate -path ./db/migrations -database 'postgres://admin:admin@127.0.0.1:5436/users?sslmode=disable' up`

## Тестирование
Postman запросы
https://www.getpostman.com/collections/6c76713950f887050d0b

### Примеры запросов
Выполнения начисления средств  
![image](https://user-images.githubusercontent.com/110117813/181467920-032ee6e3-64ac-4a12-8dd4-8da03b70347d.png)

Изменился баланс пользователя
![image](https://user-images.githubusercontent.com/110117813/181468028-9cc63eb6-d83c-4cb5-ab60-87b1b0908d29.png)

### Для подключения к БД
1. Из командной строки выполнить `docker ps`
2. Найти CONTAINER ID
3. Выполнить `docker exec -it CONTAINER ID /bin/bash`
4. Подключение к БД `psql -U admin -d users`
