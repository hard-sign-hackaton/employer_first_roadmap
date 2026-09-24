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

Сейчас демо-каталог включает 7 работодателей из IT, машиностроения, нефтехимии, атомной отрасли,
медицины и транспорта; 23 карьерных направления, 18 образовательных программ, 11 предметов ЕГЭ и
17 тегов интересов. У каждого карьерного направления есть связь хотя бы с одной ОП, набором ЕГЭ и
демонстрационной возможностью работодателя. Это позволяет проверить ранжирование по интересам,
фильтрацию направлений 11-классника по выбранным ЕГЭ и формирование roadmap в разных отраслях.

### Интеграционный сценарный тест

Тест не подключается к MAX. Он проверяет связку сервисов, репозиториев GORM и PostgreSQL для пути
«работодатель → профиль → направление → ЕГЭ → цель → roadmap».

Для него используется отдельная БД `employer_first_roadmap_test` и отдельный Docker volume:

```bash
docker compose -p efr_test -f docker-compose.yml -f docker-compose.test.yml up -d --wait postgres
docker compose -p efr_test -f docker-compose.yml -f docker-compose.test.yml run --rm --no-deps tests
```

Тест сам применяет миграции, заполняет каталог демо-данными и выполняет пользовательскую часть сценария
в откатываемой транзакции. После окончания остановить тестовую БД можно так:

```bash
docker compose -p efr_test -f docker-compose.yml -f docker-compose.test.yml down
```
