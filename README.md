# Микросервис для работы с балансом пользователей
Реализовано:
1. Метод начисления средств на баланс,
2. Метод резервирования средств,
3. Метод признания выручки,
4. Метод получения баланса пользователя,
5. Доп. задание 1 (реализовать метод для получения месячного отчета).

## Как запустить?
1. git clone https://github.com/evgkhm/golang_http_postgresql-main
2. cd golang_http_postgresql-main
3. docker-compose up --build  
4. При первом запуске может потребуется прописать миграцию БД
migrate -path ./db/migrations -database 'postgres://admin:admin@127.0.0.1:5436/users?sslmode=disable' up

Выполнения PUT-запроса для начисления/списания средств  
![image](https://user-images.githubusercontent.com/110117813/181467920-032ee6e3-64ac-4a12-8dd4-8da03b70347d.png)

Изменился баланс пользователя

![image](https://user-images.githubusercontent.com/110117813/181468028-9cc63eb6-d83c-4cb5-ab60-87b1b0908d29.png)

## Метод перевода средств от пользователя к пользователю

![image](https://user-images.githubusercontent.com/110117813/181468664-fdda0c99-2acc-433e-b3ed-1bbd7312892c.png)

![image](https://user-images.githubusercontent.com/110117813/181468717-21c28a4f-2e19-4cdf-983e-8bc5dddb3b05.png)

## Метод получения текущего баланса пользователя

![image](https://user-images.githubusercontent.com/110117813/181469640-11cf2975-74af-4851-840d-ec11201986bc.png)

Можно в $ по текущему курсу ЦБ

![image](https://user-images.githubusercontent.com/110117813/181469819-398b97d5-0f65-4401-a55a-70fc70fa0dd8.png)
