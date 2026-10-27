# Albums gRPC

gRPC-сервіс обліку музичних альбомів на Go: контракт Protocol Buffers, сервер і клієнт з унарними та
серверним стрімінговим викликами, перевірка коду в SonarQube Cloud. Проєкт виконано в межах курсу
«Go with GenAI», варіант `albums`, Тема 13 «Мікросервісна архітектура: gRPC замість REST».

Це новий репозиторій без коду попередніх тем: REST API, вебінтерфейс, PostgreSQL і serverless-частина
лишилися в репозиторіях Тем 8–12 (останній — <https://github.com/tatarchuk/golang-course-lesson-12>).

Умова Теми 13, адаптована під варіант `albums`: [albums_lesson13_homework.md](albums_lesson13_homework.md).

## Структура проєкту

| Шлях | Призначення |
|---|---|
| `albums_lesson13_homework.md` | умова домашнього завдання (UA + EN): оцінювання, кроки, чеклісти зі слайдів 44 і 46 |
| `go.mod` | модуль `homework`, Go 1.27 |
| `libratrack-grpc-demo/` | приклад із презентації (курсовий демо-проєкт `InventoryService`), скопійований без змін як окремий Go-модуль `libratrack`; від `go mod tidy` додано лише `go.sum` і неявні залежності в `go.mod` |
| `docs/environment.txt` | версії інструментів: macOS, Go, protoc, protoc-плагіни, grpcurl |
| `docs/demo/` | журнали запуску демо крок за кроком: `go-test.txt` (збірка і тести), `quickstart.txt` (швидкий старт), `unary.txt` (унарні виклики, коди помилок, grpcurl), `server-stream.txt` (серверний стрімінг і скасування клієнтом), `client-stream.txt` і `bidi.txt` (клієнтський і двонапрямлений стрімінг), `gateway.txt` (REST через grpc-gateway), `auth.txt` (JWT-токени, маски замість значень), `mtls.txt` (mTLS), `tests.txt` і `bench.txt` (тести, бенчмарк і порівняння з цифрами README демо), `smoke.txt` (`make smoke`), `proto-regen.txt` (перегенерація коду з `.proto`) |
| `Makefile` | цілі збірки й перевірок (`make` показує список) |
| `.github/workflows/lesson13-check.yml` | CI: завдання `static` (структура, форматування, збірка, документація) і `demo` (vet, тести з `-race`, smoke, актуальність згенерованого коду демо) |
| `LICENSE` | MIT |

Контракт `proto/`, згенерований код `gen/`, власні сервер, клієнт і тести та їхні докази додаються поетапно;
таблиця оновлюється разом із ними.

## Як запустити

Інструменти з `docs/environment.txt`: `brew install protobuf`, `go install` плагінів `protoc` у версіях із
`libratrack-grpc-demo/Makefile` (`make tools` у теці демо) і `grpcurl`; `$(go env GOPATH)/bin` має бути в `PATH`.
Демо з презентації запускається за його README: `cd libratrack-grpc-demo && go mod tidy && go test ./... -race`,
далі розділ 3 («Швидкий старт»). Журнали нашого запуску кожного розділу демо лежать у `docs/demo/` (таблиця вище).
Власний сервер і клієнт з'являться разом із кодом.

## CI

Робочий процес `lesson13-check.yml` запускається на кожен push і pull request і містить два завдання:

- `static`: обов'язкові файли на місці, `gofmt`, `go vet` і `go build`, `go mod tidy` без змін, лише ASCII у власному
  Go-коді, у документації немає JWT-токенів, `.env` не закомічено, shell-скрипти проходять `bash -n`. Кожна перевірка
  потрапляє в підсумок запуску (Actions → run → Summary) як PASS/FAIL; крок падає наприкінці, якщо хоч одна
  перевірка не пройшла.
- `demo`: демо з презентації в теці `libratrack-grpc-demo/` — `go vet`, `go mod tidy` без змін, тести з `-race`,
  `bash scripts/smoke.sh` (сервер + gateway + сценарії слайда 44) і перегенерація коду з `.proto` тими самими
  версіями плагінів (`make tools proto`; відмінність допускається лише в рядку з версією `protoc`).
