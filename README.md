### Формат .env файла
```
BOT_TOKEN="token"
EMPLOYER_API_PORT=8080
EMPLOYER_API_KEY="replace-with-a-secret"
EMPLOYER_ALLOWED_ORIGIN="http://localhost:3000"
```

### API работодателя

API запускается вместе с ботом на `EMPLOYER_API_PORT` и сохраняет данные в ту
же PostgreSQL, из которой пользовательские сценарии получают компании,
направления, ЕГЭ, вузы и возможности работодателя. `SEED_DEMO_DATA=false`
оставляет единственным источником каталога данные, введённые работодателем.

Ручки:

- `POST /api/v1/employer/catalog` — создать полный каталог работодателя;
- `GET /api/v1/employer/catalog/{companyID}` — получить сохранённую форму;
- `PUT /api/v1/employer/catalog/{companyID}` — атомарно обновить каталог;
- `PATCH /api/v1/employer/opportunities/{opportunityID}/status` — быстро
  включить или выключить стажировку, практику, проект либо целевое обучение;
- `GET /api/v1/employer/companies/{companyID}/applications` — получить заявки
  пользователей и всю историю ответов;
- `PATCH /api/v1/employer/applications/{applicationID}` — отправить пользователю
  статус, сообщение и контакт работодателя;
- `GET /api/v1/employer/reference-data` — справочники и допустимые enum;
- `GET /healthz` — проверка процесса без авторизации.

Все ручки `/api/v1/employer/*` принимают ключ как
`Authorization: Bearer <EMPLOYER_API_KEY>` либо `X-API-Key`. Если переменная не
задана, авторизация отключена — это удобно только для локальной разработки.
Для браузерной формы укажите её точный origin в `EMPLOYER_ALLOWED_ORIGIN`.

Пример создания связного каталога находится в
[`docs/employer-catalog.example.json`](docs/employer-catalog.example.json).
Запрос создаётся транзакционно: при ошибке ни одна часть формы не сохраняется.
Для нового направления без блока `roadmap` API создаёт обязательные девять
шагов бота автоматически. В пользовательском окружении не включайте
`APP_ENV=demo` и не задавайте `SEED_DEMO_DATA=true`.

Ответ на заявку отправляется так:

```http
PATCH /api/v1/employer/applications/123
Authorization: Bearer <EMPLOYER_API_KEY>
Content-Type: application/json

{
  "status": "interview",
  "message": "Приглашаем на интервью во вторник в 15:00",
  "contact": "hr@example.com"
}
```

Допустимые статусы: `under_review`, `interview`, `accepted`, `rejected`.
Каждое изменение добавляется в историю, а не перезаписывает предыдущее.
Пользователь проверяет актуальный ответ и историю в боте командой `/feedback`.

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
