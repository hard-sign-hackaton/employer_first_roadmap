### Формат .env файла
```
BOT_TOKEN="token"
```

### Запуск через Docker Compose
```bash
docker compose up [--build]
```

### Демонстрационная БД

Команда ниже поднимает отдельную PostgreSQL с демонстрационными данными; она не использует основной volume.

1. Создайте `.env` рядом с `docker-compose.yml` и укажите токен бота:

```env
BOT_TOKEN="ваш_токен_MAX"
```

2. Запустите бот и demo-БД:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo.yml up --build
```

При старте бот автоматически применит миграции и заполнит demo-БД. Для запуска в фоне добавьте `-d`:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo.yml up --build -d
```

Проверить состояние контейнеров:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo.yml ps
```

Остановить demo-окружение, сохранив данные:

```bash
docker compose -f docker-compose.yml -f docker-compose.demo.yml down
```

Данные предназначены только для MVP-демонстрации: названия организаций и вузов реальны, но связи, баллы,
возможности работодателей и правила приёма являются тестовыми.
