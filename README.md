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
| `go.mod` | модуль `homework`, Go 1.27; `google.golang.org/grpc` і `google.golang.org/protobuf` у версіях, які `go get` дав у день генерації (`docs/albums/proto-gen.txt`) |
| `proto/albums.proto` | контракт `albums.v1`: `AlbumsService` з `GetAlbum`, `CreateAlbum` (унарні) і `WatchAlbums` (серверний стрімінг), коментар до кожного повідомлення і виклику |
| `gen/albums/` | код, згенерований `protoc` (`make proto`), закомічений і не редагується вручну |
| `cmd/albums-server/` | gRPC-сервер (`:50051`): ланцюжок інтерсепторів, server reflection для grpcurl, м'яка зупинка з лімітом 5 с |
| `cmd/albums-client/` | Go-клієнт: `get` і `create` (унарні), `watch` (серверний стрімінг, `-max N`, Ctrl+C скасовує потік) |
| `internal/store/` | сховище в пам'яті: модель, правила валідації і демо-дані Теми 12 як Go-літерали, підписки на події; тести |
| `internal/server/` | реалізація трьох RPC, відображення помилок сховища на gRPC-коди; тести через справжній `grpc.Server` на `127.0.0.1:0` |
| `internal/middleware/` | інтерсептори recovery і logging (унарні та стрімінгові); тести |
| `scripts/smoke.sh` | наскрізна перевірка власного сервісу: збірка в `bin/`, сервер на `127.0.0.1:50061`, сценарії клієнта |
| `docs/albums/` | журнали власного сервісу: `proto-gen.txt` (генерація коду, версії модулів), `tests-before.txt` (червоний прогін до реалізації), `tests-after.txt` (зелений прогін, покриття, smoke), `grpcurl.txt` (сервіс через server reflection: list, describe, три RPC і коди), `client.txt` (Go-клієнт: get, create, watch з `-max 2` і зі скасуванням Ctrl+C), `smoke.txt` (`make smoke`) |
| `libratrack-grpc-demo/` | приклад із презентації (курсовий демо-проєкт `InventoryService`), скопійований без змін як окремий Go-модуль `libratrack`; від `go mod tidy` додано лише `go.sum` і неявні залежності в `go.mod` |
| `docs/environment.txt` | версії інструментів: macOS, Go, protoc, protoc-плагіни, grpcurl |
| `docs/demo/` | журнали запуску демо крок за кроком: `go-test.txt` (збірка і тести), `quickstart.txt` (швидкий старт), `unary.txt` (унарні виклики, коди помилок, grpcurl), `server-stream.txt` (серверний стрімінг і скасування клієнтом), `client-stream.txt` і `bidi.txt` (клієнтський і двонапрямлений стрімінг), `gateway.txt` (REST через grpc-gateway), `auth.txt` (JWT-токени, маски замість значень), `mtls.txt` (mTLS), `tests.txt` і `bench.txt` (тести, бенчмарк і порівняння з цифрами README демо), `smoke.txt` (`make smoke`), `proto-regen.txt` (перегенерація коду з `.proto`) |
| `docs/screenshots/` | знімки з термінала користувача для кожного кроку (таблиця в розділі «Скриншоти») |
| `Makefile` | цілі збірки й перевірок (`make` показує список) |
| `.github/workflows/lesson13-check.yml` | CI: завдання `static` (структура, форматування, збірка, контракт, документація), `test` (власний модуль: vet, тести з `-race` і покриттям, smoke) і `demo` (те саме для демо з презентації) |
| `LICENSE` | MIT |

## Як запустити

Інструменти з `docs/environment.txt`: `brew install protobuf`, `go install` плагінів `protoc` у версіях із
`libratrack-grpc-demo/Makefile` (`make tools`) і `grpcurl`; `$(go env GOPATH)/bin` має бути в `PATH`.
Код із контракту перегенеровується командою `make proto` (`make` без аргументів показує всі цілі).

Власний сервіс: `make server` (або `go run ./cmd/albums-server`) у першому терміналі, у другому —
`go run ./cmd/albums-client get 1`, `go run ./cmd/albums-client create "Blue Train" "Blue Note" 1 2`,
`go run ./cmd/albums-client -max 2 watch`; `make test` — тести з `-race` і покриттям, `make smoke` — наскрізна перевірка.
Демо з презентації запускається за його README: `cd libratrack-grpc-demo && go mod tidy && go test ./... -race`,
далі розділ 3 («Швидкий старт»). Журнали нашого запуску кожного розділу демо лежать у `docs/demo/` (таблиця вище).
Власний сервер і клієнт з'являться разом із кодом.

## Скриншоти

Знімки зроблено в терміналі користувача після прогону кожного розділу README демо; поруч вказано журнал сесії
з тими самими командами.

| Знімок | Що показано | Журнал |
|---|---|---|
| [s01-versions.png](docs/screenshots/s01-versions.png) | версії macOS, Go, protoc, grpcurl і protoc-плагінів | `docs/environment.txt` |
| [s02-demo-test.png](docs/screenshots/s02-demo-test.png) | `go test ./... -race -count=1` демо: три пакети з тестами зелені | `docs/demo/go-test.txt` |
| [s03-quickstart.png](docs/screenshots/s03-quickstart.png) | швидкий старт: `client get copy-1` і `curl` через gateway | `docs/demo/quickstart.txt` |
| [s04-unary-codes.png](docs/screenshots/s04-unary-codes.png) | унарні виклики: `NotFound`, `AlreadyExists`, grpcurl через server reflection | `docs/demo/unary.txt` |
| [s05-server-stream.png](docs/screenshots/s05-server-stream.png) | серверний стрімінг: три оновлення й `io.EOF`, скасування через Ctrl+C | `docs/demo/server-stream.txt` |
| [s06-client-bidi.png](docs/screenshots/s06-client-bidi.png) | клієнтський стрімінг (`bulk`: 5/0, повторно 0/5) і двонапрямлений (`sync`: `applied=true`/`false`) | `docs/demo/client-stream.txt`, `docs/demo/bidi.txt` |
| [s07-gateway.png](docs/screenshots/s07-gateway.png) | REST через gateway: 200, 409, 400 і потік JSON-рядків `watch` | `docs/demo/gateway.txt` |
| [s08-token.png](docs/screenshots/s08-token.png) | JWT: `Unauthenticated`, `PermissionDenied` для reader, успіх для librarian, REST 401/200 | `docs/demo/auth.txt` |
| [s09-mtls.png](docs/screenshots/s09-mtls.png) | mTLS: `gencerts` і чотири негативні сценарії (без TLS, без сертифіката, чужий CA, без токена) | `docs/demo/mtls.txt` |
| [s10-tests-bench.png](docs/screenshots/s10-tests-bench.png) | тести стрімінгу (`-run Watch -v`), розмір повідомлень і бенчмарк | `docs/demo/tests.txt`, `docs/demo/bench.txt` |
| [s11-smoke.png](docs/screenshots/s11-smoke.png) | `make smoke`: «ВСЕ ГАРАЗД» | `docs/demo/smoke.txt` |
| [s12-proto-regen.png](docs/screenshots/s12-proto-regen.png) | `make proto`: відмінність лише у двох рядках із версією protoc, відкат | `docs/demo/proto-regen.txt` |

## CI

Робочий процес `lesson13-check.yml` запускається на кожен push і pull request і містить два завдання:

- `static`: обов'язкові файли на місці, `gofmt`, `go vet` і `go build`, `go mod tidy` без змін, лише ASCII у власному
  Go-коді, у документації немає JWT-токенів, `.env` не закомічено, shell-скрипти проходять `bash -n`, контракт має
  пакет `albums.v1`, три rpc і рівно один `stream`, згенерований код актуальний (`make proto` у CI), обробники
  сервера повертають лише gRPC-статуси (слайд 46). Кожна перевірка
  потрапляє в підсумок запуску (Actions → run → Summary) як PASS/FAIL; крок падає наприкінці, якщо хоч одна
  перевірка не пройшла.
- `test`: власний модуль — `go vet`, тести з `-race` і профілем покриття `cover.out` (зберігається як артефакт запуску),
  `bash scripts/smoke.sh`.
- `demo`: демо з презентації в теці `libratrack-grpc-demo/` — `go vet`, `go mod tidy` без змін, тести з `-race`,
  `bash scripts/smoke.sh` (сервер + gateway + сценарії слайда 44) і перегенерація коду з `.proto` тими самими
  версіями плагінів (`make tools proto`; відмінність допускається лише в рядку з версією `protoc`).

Перший запуск з обома завданнями пройшов: [run 40538510618](https://github.com/tatarchuk/golang-course-lesson-13/actions/runs/40538510618).
