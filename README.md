### Формат .env файла
```
BOT_TOKEN="token"
EMPLOYER_API_PORT=8080
ADMIN_API_TOKEN="replace-with-a-secret"
EMPLOYER_ALLOWED_ORIGIN="http://localhost:3000"
```

### API работодателя

API запускается вместе с ботом на `EMPLOYER_API_PORT` и сохраняет данные в ту
же PostgreSQL, из которой пользовательские сценарии получают компании,
направления, ЕГЭ, вузы и возможности работодателя. Токены индивидуальны:
в БД хранится только SHA-256-хэш токена.

Ручки:

- `GET /api/v1/me` и `GET /api/v1/reference-data` — профиль API-аккаунта и
  неизменяемые справочники;
- `GET /api/v1/education-programs` — готовый список ОП, из которого
  работодатель выбирает программы для направления;
- `/api/v1/employer/company`, `/directions`, `/opportunities` — кабинет
  работодателя. Компания определяется токеном, а не параметром URL;
- `PUT /api/v1/employer/directions/{id}/education-programs` — связать своё
  направление с уже существующими ОП;
- `GET /api/v1/employer/applications` — получить заявки только своей компании;
- `PATCH /api/v1/employer/applications/{applicationID}` — отправить пользователю
  статус, сообщение и контакт работодателя;
- `/api/v1/admin/*` — backoffice администратора: аккаунты, компании, вузы, ОП,
  наборы ЕГЭ, проходные баллы, правила приёма и шаблоны roadmap;
- `GET /healthz` — проверка процесса без авторизации.

Все ручки, кроме `/healthz`, требуют `Authorization: Bearer <token>`.
При старте `ADMIN_API_TOKEN` создаёт или обновляет аккаунт `bootstrap-admin`.
Администратор создаёт аккаунт работодателя через `POST /api/v1/admin/accounts`;
сгенерированный токен возвращается только в ответе на создание или перевыпуск.
Администратор получает списки всех динамических сущностей через `GET`-ручки
`/api/v1/admin/*`. Вместо опасного удаления у компаний, направлений,
возможностей, вузов и ОП есть `PATCH .../archive` с телом
`{"is_active": false}`; архивные записи не попадают в новые рекомендации бота,
но сохраняются для истории roadmap.
Для браузерной формы укажите её точный origin в `EMPLOYER_ALLOWED_ORIGIN`.

Ответ на заявку отправляется так:

```http
PATCH /api/v1/employer/applications/123
Authorization: Bearer <employer-token>
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

### Запуск: выберите один каталог

Создайте `.env` рядом с `docker-compose.yml` и укажите как минимум токен MAX:

```env
BOT_TOKEN="ваш_токен_MAX"
```

Дальше выберите один из двух изолированных контуров. Не нужно вручную менять
`SEED_DEMO_DATA` или `SEED_MOSCOW_CATALOG` в `.env`: нужный Compose-override
задаёт их сам. Для одновременной работы двух контуров используйте разные имена
проектов, как в командах ниже.

| Контур | Данные | Команда запуска |
| --- | --- | --- |
| Demo | Тестовые связи, баллы и возможности. Подходит для разработки и проверки всех сценариев. | `docker compose -p efr_demo -f docker-compose.yml -f docker-compose.demo.yml up --build -d` |
| Москва / real catalog | Проверенные московские вузы, ОП и возможности работодателей с официальными ссылками. Подходит для демонстрации реальных траекторий. | `docker compose -p efr_moscow -f docker-compose.yml -f docker-compose.moscow.yml up --build -d` |

Обычный `docker compose up` без override применяет миграции и обязательные
справочники, но **не** загружает ни demo-, ни московский каталог.

### Демонстрационный контур

Запуск:

```bash
docker compose -p efr_demo -f docker-compose.yml -f docker-compose.demo.yml up --build -d
```

Проверить состояние и логи:

```bash
docker compose -p efr_demo -f docker-compose.yml -f docker-compose.demo.yml ps
docker compose -p efr_demo -f docker-compose.yml -f docker-compose.demo.yml logs -f efr_bot
```

Остановить, сохранив demo-данные:

```bash
docker compose -p efr_demo -f docker-compose.yml -f docker-compose.demo.yml down
```

Чтобы начать demo-контур с пустой БД, удалите **только его** volume:

```bash
docker compose -p efr_demo -f docker-compose.yml -f docker-compose.demo.yml down -v
```

Данные предназначены только для MVP-демонстрации: названия организаций и вузов реальны, но связи, баллы,
возможности работодателей и правила приёма являются тестовыми.

Сейчас демо-каталог включает 7 работодателей из IT, машиностроения, нефтехимии, атомной отрасли,
медицины и транспорта; 23 карьерных направления, 18 образовательных программ, 11 предметов ЕГЭ и
17 тегов интересов. У каждого карьерного направления есть связь хотя бы с одной ОП, набором ЕГЭ и
демонстрационной возможностью работодателя. Это позволяет проверить ранжирование по интересам,
фильтрацию направлений 11-классника по выбранным ЕГЭ и формирование roadmap в разных отраслях.

### Московский контур с реальными данными

Контур использует свою PostgreSQL (`employer_first_roadmap_moscow`) и свой volume,
поэтому не смешивается с demo-БД. При первом старте бот применит миграции и
идемпотентно импортирует реальный каталог.

```bash
docker compose -p efr_moscow -f docker-compose.yml -f docker-compose.moscow.yml up --build -d
```

Проверить состояние и логи:

```bash
docker compose -p efr_moscow -f docker-compose.yml -f docker-compose.moscow.yml ps
docker compose -p efr_moscow -f docker-compose.yml -f docker-compose.moscow.yml logs -f efr_bot
```

Остановить, сохранив каталог:

```bash
docker compose -p efr_moscow -f docker-compose.yml -f docker-compose.moscow.yml down
```

Полностью удалить только московскую БД и при следующем запуске импортировать
каталог заново:

```bash
docker compose -p efr_moscow -f docker-compose.yml -f docker-compose.moscow.yml down -v
```

Сейчас real-каталог содержит 10 работодателей, 8 московских вузов, 26 ОП и 19
возможностей. Для возможностей хранятся официальные карьерные/стажировочные
ссылки и дата проверки; удалённые варианты не привязаны к региону, а очные — к
Москве. Demo-записи в этом контуре не создаются.

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
