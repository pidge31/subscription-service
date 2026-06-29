# Subscription Service

REST API для агрегации данных об онлайн-подписках пользователей.

## Стек

- Go 1.25
- PostgreSQL 16
- [chi](https://github.com/go-chi/chi) — HTTP-роутер
- [sqlx](https://github.com/jmoiron/sqlx) — работа с БД
- [golang-migrate](https://github.com/golang-migrate/migrate) — миграции
- [swaggo](https://github.com/swaggo/swag) — Swagger-документация

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

Сервис поднимается на `http://localhost:8080`.  
Swagger UI: `http://localhost:8080/swagger/index.html`

## Конфигурация

Все параметры задаются через `.env` (или переменные окружения):

| Переменная    | Значение по умолчанию | Описание              |
|---------------|-----------------------|-----------------------|
| DB_HOST       | localhost             | Хост PostgreSQL       |
| DB_PORT       | 5432                  | Порт PostgreSQL       |
| DB_USER       | postgres              | Пользователь БД       |
| DB_PASSWORD   | postgres              | Пароль БД             |
| DB_NAME       | subscriptions         | Имя базы данных       |
| SERVER_PORT   | 8080                  | Порт HTTP-сервера     |

## API

### Подписки

| Метод  | Путь                        | Описание                          |
|--------|-----------------------------|-----------------------------------|
| POST   | /subscriptions              | Создать подписку                  |
| GET    | /subscriptions              | Список подписок (с фильтрами)     |
| GET    | /subscriptions/{id}         | Получить подписку по ID           |
| PUT    | /subscriptions/{id}         | Обновить подписку                 |
| DELETE | /subscriptions/{id}         | Удалить подписку                  |
| GET    | /subscriptions/total        | Суммарная стоимость за период     |

### Формат дат

Даты передаются в формате `MM-YYYY` (например, `07-2025`).

### Создание подписки

```
POST /subscriptions
```

```json
{
  "service_name": "Yandex Plus",
  "price": 400,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "start_date": "07-2025",
  "end_date": "12-2025"
}
```

`end_date` — необязательное поле.

### Обновление подписки

```
PUT /subscriptions/{id}
```

Все поля опциональны; передаются только те, что нужно изменить:

```json
{
  "price": 500,
  "end_date": "06-2026"
}
```

### Суммарная стоимость

```
GET /subscriptions/total?from=07-2025&to=12-2025&user_id=...&service_name=...
```

| Параметр     | Обязательный | Описание                        |
|--------------|--------------|---------------------------------|
| from         | да           | Начало периода (MM-YYYY)        |
| to           | да           | Конец периода (MM-YYYY)         |
| user_id      | нет          | Фильтр по пользователю          |
| service_name | нет          | Фильтр по названию сервиса      |

Ответ:

```json
{
  "total": 3000
}
```

Стоимость считается как `price × количество месяцев` за пересечение периода подписки с запрошенным диапазоном.

## Структура проекта

```
.
├── cmd/
│   └── main.go          # точка входа, запуск сервера
├── internal/
│   ├── config.go        # конфигурация и подключение к БД
│   ├── model.go         # модели и типы запросов/ответов
│   ├── repository.go    # слой доступа к данным (SQL)
│   ├── service.go       # бизнес-логика и валидация
│   └── handler.go       # HTTP-хендлеры и роуты
├── migrations/
│   └── 000001_create_subscriptions_table.up.sql
├── docs/                # сгенерированная Swagger-документация
├── docker-compose.yml
├── Dockerfile
└── .env.example
```

## Миграции

Применяются автоматически при старте сервиса.
