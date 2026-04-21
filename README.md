# EventHub - NoSQL Database Project

[![EventHub](https://github.com/ruskaof/nosql-labs/actions/workflows/eventhub.yml/badge.svg)](https://github.com/ruskaof/nosql-labs/actions/workflows/eventhub.yml)

## Настройка проекта

Настройка выполняется через переменные окружения (например, в `.env.local`).

### Конфигурация приложения

| Переменная | Обязательная | Описание | По умолчанию |
|------------|---------------|----------|--------------|
| `APP_HOST` | да | Адрес, на котором слушает сервер | — |
| `APP_PORT` | да | Порт HTTP-сервера | — |
| `APP_USER_SESSION_TTL` | да | Время жизни сессии в секундах (целое число > 0) | — |

### Redis

| Переменная | Обязательная | Описание | По умолчанию |
|------------|---------------|----------|--------------|
| `REDIS_HOST` | нет | Хост Redis | `localhost` |
| `REDIS_PORT` | нет | Порт Redis | `6379` |
| `REDIS_PASSWORD` | нет | Пароль Redis (пусто — без пароля) | — |
| `REDIS_DB` | нет | Номер базы Redis | `0` |

### MongoDB

| Переменная | Обязательная | Описание | По умолчанию |
|------------|---------------|----------|--------------|
| `MONGODB_DATABSE` | да | Имя базы данных MongoDB | — |
| `MONGODB_USER` | да | Имя пользователя MongoDB | — |
| `MONGODB_PASSWORD` | да | Пароль пользователя MongoDB | — |
| `MONGODB_HOST` | да | Хост MongoDB | `mongo` |
| `MONGODB_PORT` | да | Порт MongoDB | `27017` |


## Запуск проекта

Для запуска проекта достаточно выполнить

```
make run
```
