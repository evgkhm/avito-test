# Микросервис для работы с балансом пользователей
## Реализовано:
1. Метод начисления средств на баланс
2. Метод резервирования средств
3. Метод признания выручки
4. Метод получения баланса пользователя
5. Доп. задание 1 (реализовать метод для получения месячного отчета)
6. Сценарий разрезервирования денег

## Как запустить?
1. `git clone https://github.com/evgkhm/avito-test`
2. cd avito-test
3. `docker-compose up --build`  
4. При первом запуске может потребуется прописать миграцию БД
`migrate -path ./db/migrations -database 'postgres://admin:admin@127.0.0.1:5436/users?sslmode=disable' up`

## Тестирование
Postman запросы
https://www.getpostman.com/collections/6c76713950f887050d0b

### Примеры запросов
Метод получения баланса пользователя при первом подключении сообщит о том, что пользователь отсутствует  
![image](https://user-images.githubusercontent.com/110117813/200485739-49d09784-19c7-4b3a-a6e0-546b69446bdd.png)

Метод начисления средств пользователю
![image](https://user-images.githubusercontent.com/110117813/200485869-265a0a38-c134-4be5-b804-23c980c3ad9a.png)

После начисления средств создается пользователь
![image](https://user-images.githubusercontent.com/110117813/200486013-d32bc55b-a06a-4a75-902a-cc545d91dd43.png)

Метод резервирования средств
![image](https://user-images.githubusercontent.com/110117813/200486301-a5518a68-f5fd-4fca-91f1-d254a0bcb095.png)

Метод признания выручки
![image](https://user-images.githubusercontent.com/110117813/200486427-dc44a251-d3e6-4631-942d-0d9ea5928d15.png)

Метод получения месячного отсчета 
![image](https://user-images.githubusercontent.com/110117813/200486587-d9018696-7f8a-4d5d-8a03-c7757563b69e.png)

Создается report.csv с данными
![image](https://user-images.githubusercontent.com/110117813/200488259-66a79c3d-a8f9-4b45-bb84-6cf2d99e103e.png)

Сценарий разрезервирования денег
![image](https://user-images.githubusercontent.com/110117813/200497194-c0ff6a0a-b889-49ed-8086-abfdb4520335.png)


### Для подключения к БД
1. Из командной строки выполнить `docker ps`
2. Найти CONTAINER ID
3. Выполнить `docker exec -it CONTAINER ID /bin/bash`
4. Подключение к БД `psql -U admin -d users`
