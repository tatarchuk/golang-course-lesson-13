# Домашнє завдання 13 — Мікросервісна архітектура: gRPC замість REST (варіант `albums`)

Курс «Go with GenAI», Модуль 3 · Урок 13. Джерела: презентація
`Go_with_GenAI_Lesson13_gRPC_Microservices_UA.pdf` (50 слайдів) і курсовий демо-проєкт `libratrack-grpc-demo`
(`InventoryService`: чотири типи викликів, grpc-gateway, mTLS, JWT, тести, бенчмарк, smoke). Умова нижче — це
офіційне оцінювання «[Tasks 13]», адаптоване під варіант `albums`. Домашнє завдання зі слайда 48 (сервіс із 3+
методами, бенчмарк проти REST, `.proto` від ШІ) виконується лише в частині «три методи, один із них стрімінговий».

## Оцінювання

| Завдання | Бали |
|---|---|
| 1. Запустити приклади з презентації на власному ПК | 70 |
| 2. Реалізувати сервер і клієнт на Go: 1 унарний RPC (створення або читання об'єкта) і 1 стрімінговий RPC на вибір (серверний, клієнтський або двонапрямлений); запустити й продемонструвати коректну взаємодію клієнта й сервера локально | +30 |
| 3. Перевірити код застосунку в SonarQube Cloud і показати результати | обов'язково, бали не вказано |

Здати: скриншоти або відеозапис роботи застосунку та посилання на GitHub-репозиторій із застосунками й
конфігураційними файлами.

## Завдання 1. Приклади з презентації на власному ПК (70)

Приклад із презентації — курсовий демо-проєкт `libratrack-grpc-demo`, скопійований без змін у теку
`libratrack-grpc-demo/` цього репозиторію (окремий Go-модуль `libratrack`). Його README описує сценарії запуску;
кожен відповідає слайдам презентації. Усе виконується на власному ПК (macOS, Apple Silicon), без Docker і VM.

| Крок | Сценарій README демо | Слайди | Що має бути видно |
|---|---|---|---|
| 1.1 | Інструменти: `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `grpcurl` | 15–16, 41 | версії інструментів |
| 1.2 | `go mod tidy`, `go vet`, `go test ./... -race` | 44 | зелені тести демо |
| 1.3 | Швидкий старт (розділ 3): сервер, `client get copy-1`, gateway і `curl /v1/copies/copy-1` | 19, 28, 43 | три термінали |
| 1.4 | 4.1 Unary і коди помилок: `get`, невідомий id → `NotFound`, `register`, повтор → `AlreadyExists`; те саме через `grpcurl` | 19, 41, 44 | коди статусу, а не загальна помилка |
| 1.5 | 4.2 Серверний стрімінг: `watch` з `-max`, кілька оновлень, чистий `io.EOF`; `watch` без `-max` і Ctrl+C → сервер бачить скасування | 20, 42, 46 | `io.EOF` і звільнення підписки |
| 1.6 | 4.3 Клієнтський стрімінг (`bulk`), 4.4 двонапрямлений (`sync`) | 21, 22 | підсумок `bulk`, `applied` у `sync` |
| 1.7 | 4.5 REST через grpc-gateway: 200, 404, POST, 409, PATCH, 400, NDJSON-стрім | 28, 43, 46 | ті самі дані, що й у нативному gRPC |
| 1.8 | 4.6 Токен-автентифікація: без токена `Unauthenticated`, `reader` → `PermissionDenied`, `librarian` → успіх; `grpcurl -H`; REST 401/200/403 | 32–33 | коди за ролями, також для стрімінгу |
| 1.9 | 4.7 mTLS: `gencerts`, `-tls -auth`, успіх, негативні сценарії з причинами відмови | 30–31 | відмова без сертифіката |
| 1.10 | Розділ 5: `go test ./internal/server -run Watch -v`, `TestPayloadSize`, `-bench=.` | 44, 47 | розмір payload, ns/op |
| 1.11 | `make smoke` | 44, 46 | `ВСЕ ГАРАЗД` |
| 1.12 | Розділ 6: `make tools`, `make proto`, `git diff --stat gen/` | 15–16 | згенерований код не змінився |

Докази: текстовий журнал кожного кроку в `docs/demo/`, версії інструментів у `docs/environment.txt`, скриншоти
запущених програм у `docs/screenshots/`. Усе, що на macOS не спрацювало так, як написано в README демо, описується
у звіті в таблиці «що не спрацювало як написано» з виправленням.

## Завдання 2. Власний сервер і клієнт на Go (+30)

Предметна область варіанта — музичні альбоми. Контракт `proto/albums.proto` (пакет `albums.v1`, три методи,
один стрімінговий — як вимагає слайд 48):

| RPC | Тип | Аналог у REST API попередніх тем |
|---|---|---|
| `GetAlbum(GetAlbumRequest) returns (Album)` | унарний | `GET /albums/{id}` |
| `CreateAlbum(CreateAlbumRequest) returns (Album)` | унарний | `POST /albums` |
| `WatchAlbums(WatchAlbumsRequest) returns (stream AlbumEvent)` | серверний стрімінг | немає (живі події `CREATED`) |

Вимоги:

- сховище в пам'яті з демо-даними попередніх тем (3 жанри, 6 виконавців, 10 альбомів із зв'язками з виконавцями)
  і тими самими правилами валідації (`title`, `label`, `genre_id`, `artist_ids`, `release_year`);
- коди статусу gRPC замість звичайних помилок (слайди 45–46): `InvalidArgument` (валідація), `NotFound`,
  `FailedPrecondition` (невідомий жанр або виконавець), `AlreadyExists` (дубль `(title, label)`), `Internal`;
  в обробниках немає `errors.New` і `fmt.Errorf`;
- стрімінговий обробник працює зі `stream.Context()`: скасування з боку клієнта завершує цикл і звільняє підписку
  (горутина не витікає); `max_events > 0` завершує потік чистим `io.EOF`;
- server reflection для `grpcurl`; відкритий текст (`insecure.NewCredentials()`) лише для локальної розробки,
  з попередженням у журналі сервера (слайди 33, 44);
- клієнт на Go: `get`, `create`, `watch`; тести над справжнім `grpc.Server` (написані першими: червоний і зелений
  прогони зберігаються); smoke-скрипт.

Докази: `docs/albums/` (генерація коду, тести до і після, `grpcurl`, клієнт, smoke), скриншоти S13–S16.

## Завдання 3. SonarQube Cloud

- Проєкт у SonarQube Cloud (приватний акаунт GitHub `tatarchuk`, безкоштовний план), аналіз із GitHub Actions
  із покриттям із `go test -coverprofile`. Згенерований код (`*.pb.go`), документація й курсовий демо-проєкт
  виключені з аналізу: це не наш код.
- Показати quality gate, покриття, issues і security hotspots; розібрати знахідки — виправити або пояснити.

Докази: `docs/sonar/` (журнал сканування, список issues, результати й розбір), скриншоти S17–S21, бейдж у README.

## Чеклісти з презентації

Слайд 44, чекліст тестування збірки:

| Сценарій | Очікуваний результат |
|---|---|
| Виклик `grpcurl` до `GetCopy` із коректним ID | коректно повертає message `Copy` |
| Виклик `grpcurl` до `GetCopy` з невідомим ID | `codes.NotFound`, а не загальна помилка |
| Стрімінговий виклик Go-клієнта до `WatchCopyStatus` | кілька оновлень, а потім чистий `io.EOF` |
| REST-виклик через grpc-gateway (`curl GET /v1/copies/:id`) | ті самі дані, що й у нативному gRPC-виклику, у JSON |
| Сервер без TLS лише в локальній розробці | `insecure.NewCredentials()` ніколи не потрапляє в спільну чи продакшен-конфігурацію |

Слайд 46, чекліст перед завершенням:

- кожен RPC повертає коректні gRPC-коди статусу в разі збою (grep на `errors.New` і `fmt.Errorf` в обробниках
  порожній, замість них `status.Errorf`);
- стрімінговий виклик звільняє ресурси при відключенні клієнта (скасувати контекст, переконатися, що серверний
  цикл завершується і горутина не витікає);
- REST-місток повертає ті самі дані, що й нативний gRPC-виклик (`grpcurl` і `curl` для одного ресурсу поруч);
- перевірено невідомий і некоректний ID, а не лише «щасливий» сценарій.

Обидва чекліста відмічаються у звіті з посиланнями на докази: для демо (завдання 1) і для власного сервісу
(завдання 2, крім REST-містка, якого у власному сервісі немає).

## Що здати

- Публічний репозиторій GitHub: демо, власний сервіс, конфігурація CI і SonarQube, докази в `docs/`.
- Скриншоти S1–S21 у `docs/screenshots/` (перелік у звіті README).
- Звіт у README: середовище, кроки, що не спрацювало як написано, результати SonarQube Cloud.

---

# Homework 13 — Microservice architecture: gRPC instead of REST (variant `albums`)

Course "Go with GenAI", Module 3, Lesson 13. Sources: the deck `Go_with_GenAI_Lesson13_gRPC_Microservices_UA.pdf`
(50 slides) and the course demo `libratrack-grpc-demo` (`InventoryService`: the four call types, grpc-gateway, mTLS,
JWT, tests, benchmark, smoke). The task below is the official "[Tasks 13]" grading adapted to the `albums` variant.
The deck's own homework (slide 48: a service with 3+ methods, a benchmark against REST, an AI-generated `.proto`)
is in scope only as "three methods, one of them streaming".

## Grading

| Task | Points |
|---|---|
| 1. Run the examples from the presentation on your own PC | 70 |
| 2. Implement a server and a client in Go: 1 unary RPC (creating or retrieving an object) and 1 streaming RPC of your choice (server, client or bidirectional streaming); run the application and demonstrate the correct interaction between client and server on your local machine | +30 |
| 3. Test the application's code in SonarQube Cloud and show the results | required, no points stated |

Submit screenshots or a video recording of the application in operation and a link to the GitHub repository with
the applications and the configuration files.

## Task 1. The examples from the presentation on your own PC (70)

The example is the course demo `libratrack-grpc-demo`, copied verbatim into `libratrack-grpc-demo/` of this
repository (a separate Go module `libratrack`). Its README describes the run scenarios; each maps to deck slides.
Everything runs on the own PC (macOS, Apple Silicon), without Docker or a VM.

| Step | Demo README scenario | Slides | What must be visible |
|---|---|---|---|
| 1.1 | Tools: `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway`, `grpcurl` | 15-16, 41 | tool versions |
| 1.2 | `go mod tidy`, `go vet`, `go test ./... -race` | 44 | the demo's tests are green |
| 1.3 | Quick start (section 3): server, `client get copy-1`, gateway and `curl /v1/copies/copy-1` | 19, 28, 43 | three terminals |
| 1.4 | 4.1 Unary and error codes: `get`, unknown id -> `NotFound`, `register`, again -> `AlreadyExists`; the same with `grpcurl` | 19, 41, 44 | status codes, not a generic error |
| 1.5 | 4.2 Server streaming: `watch` with `-max`, several updates, a clean `io.EOF`; `watch` without `-max` and Ctrl+C -> the server sees the cancellation | 20, 42, 46 | `io.EOF` and the released subscription |
| 1.6 | 4.3 Client streaming (`bulk`), 4.4 bidirectional (`sync`) | 21, 22 | the `bulk` summary, `applied` in `sync` |
| 1.7 | 4.5 REST through grpc-gateway: 200, 404, POST, 409, PATCH, 400, NDJSON stream | 28, 43, 46 | the same data as native gRPC |
| 1.8 | 4.6 Token authentication: no token `Unauthenticated`, `reader` -> `PermissionDenied`, `librarian` -> success; `grpcurl -H`; REST 401/200/403 | 32-33 | codes by role, streaming included |
| 1.9 | 4.7 mTLS: `gencerts`, `-tls -auth`, success, the negative scenarios with their reasons | 30-31 | refusal without a certificate |
| 1.10 | Section 5: `go test ./internal/server -run Watch -v`, `TestPayloadSize`, `-bench=.` | 44, 47 | payload size, ns/op |
| 1.11 | `make smoke` | 44, 46 | `ВСЕ ГАРАЗД` |
| 1.12 | Section 6: `make tools`, `make proto`, `git diff --stat gen/` | 15-16 | the generated code did not change |

Evidence: a text log of every step in `docs/demo/`, tool versions in `docs/environment.txt`, screenshots of running
programs in `docs/screenshots/`. Whatever did not work on macOS as the demo README says goes into the report's
"what did not work as written" table with its fix.

## Task 2. Own server and client in Go (+30)

The variant's domain is music albums. The contract `proto/albums.proto` (package `albums.v1`, three methods, one
streaming, as slide 48 asks):

| RPC | Kind | Counterpart in the REST API of the earlier lessons |
|---|---|---|
| `GetAlbum(GetAlbumRequest) returns (Album)` | unary | `GET /albums/{id}` |
| `CreateAlbum(CreateAlbumRequest) returns (Album)` | unary | `POST /albums` |
| `WatchAlbums(WatchAlbumsRequest) returns (stream AlbumEvent)` | server streaming | none (live `CREATED` events) |

Requirements:

- an in-memory store with the demo data of the earlier lessons (3 genres, 6 artists, 10 albums with their artist
  links) and the same validation rules (`title`, `label`, `genre_id`, `artist_ids`, `release_year`);
- gRPC status codes instead of plain errors (slides 45-46): `InvalidArgument` (validation), `NotFound`,
  `FailedPrecondition` (unknown genre or artist), `AlreadyExists` (duplicate `(title, label)`), `Internal`;
  no `errors.New` or `fmt.Errorf` in the handlers;
- the streaming handler works from `stream.Context()`: a client cancellation ends the loop and releases the
  subscription (no goroutine leak); `max_events > 0` ends the stream with a clean `io.EOF`;
- server reflection for `grpcurl`; plaintext (`insecure.NewCredentials()`) for local development only, with a
  warning in the server log (slides 33, 44);
- a Go client: `get`, `create`, `watch`; tests over a real `grpc.Server` (written first: the red and the green
  runs are kept); a smoke script.

Evidence: `docs/albums/` (code generation, tests before and after, `grpcurl`, client, smoke), screenshots S13-S16.

## Task 3. SonarQube Cloud

- A SonarQube Cloud project (private GitHub account `tatarchuk`, Free plan), analysis from GitHub Actions with
  coverage from `go test -coverprofile`. Generated code (`*.pb.go`), the documentation and the course demo are
  excluded from the analysis: not our code.
- Show the quality gate, coverage, issues and security hotspots; triage the findings: fix or explain.

Evidence: `docs/sonar/` (scan log, issue list, results and triage), screenshots S17-S21, a badge in the README.

## Checklists from the deck

Slide 44, build-test checklist:

| Scenario | Expected result |
|---|---|
| `grpcurl` call to `GetCopy` with a valid ID | returns the `Copy` message correctly |
| `grpcurl` call to `GetCopy` with an unknown ID | `codes.NotFound`, not a generic error |
| Go client streaming call to `WatchCopyStatus` | several updates, then a clean `io.EOF` |
| REST call through grpc-gateway (`curl GET /v1/copies/:id`) | the same data as the native gRPC call, as JSON |
| Server without TLS only in local development | `insecure.NewCredentials()` never reaches a shared or production configuration |

Slide 46, closing checklist:

- every RPC returns correct gRPC status codes on failure (a grep for `errors.New` and `fmt.Errorf` in the handlers
  is empty; `status.Errorf` instead);
- a streaming call releases its resources when the client disconnects (cancel the context, confirm the server loop
  ends and the goroutine does not leak);
- the REST bridge returns the same data as the native gRPC call (`grpcurl` and `curl` for one resource side by side);
- tested with an unknown and an invalid ID, not only the happy path.

Both checklists are ticked in the report with links to the evidence: for the demo (task 1) and for the own service
(task 2, except the REST bridge, which the own service does not have).

## Deliverables

- A public GitHub repository: the demo, the own service, the CI and SonarQube configuration, evidence under `docs/`.
- Screenshots S1-S21 under `docs/screenshots/` (listed in the README report).
- The report in the README: environment, steps, what did not work as written, the SonarQube Cloud results.
