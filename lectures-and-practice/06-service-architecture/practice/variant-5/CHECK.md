# Что сделано (вариант 5: модерация комментариев)

Сервис отрефакторен из одного `main.go` в слои. Поведение по ТЗ не менялось.
Старый код сохранён как [main_old.go](main_old.go) (тег `//go:build ignore`, в сборку не входит) - для сравнения.

## Структура

| Путь | Что там |
|---|---|
| [main.go](main.go) | composition root: собирает адаптеры, сервис, хендлеры |
| [domain/comment.go](domain/comment.go) | богатая модель `Comment`: `NewComment`, `ChangeText`, `ApplyModeration`, `ApplyEditCheck`, откат `published → pending` |
| [domain/author.go](domain/author.go) | автор и репутация (`CanSkipManualModeration`) |
| [domain/errors.go](domain/errors.go) | доменные ошибки |
| [service/ports.go](service/ports.go) | порты: `CommentRepository`, `AuthorRepository`, `SpamFilter`, `ToxicityChecker` |
| [service/comment_service.go](service/comment_service.go) | сценарии Create / Get / EditText |
| [repository/memory.go](repository/memory.go) | адаптеры хранилища, отдельные модели записей + маппинг |
| [adapter/http_clients.go](adapter/http_clients.go) | HTTP-адаптеры спам-фильтра и токсичности |
| [handler/comment_handler.go](handler/comment_handler.go) | HTTP, `writeError` - единственное место перевода ошибок в статусы |
| [service/comment_service_test.go](service/comment_service_test.go) | тесты с fake-портами |

## По пяти пунктам

1. **Слои** - handler → service → domain; repository/adapter реализуют порты service. Граница видна по импортам (domain ничего не импортирует из проекта).
2. **Богатые модели** - статус меняется только методами `Comment`; типизированный `Status`; откат `published → pending` в `ApplyEditCheck`.
3. **Ошибки по слоям** - `domain.Err*`; `sql.ErrNoRows` и `findByID` убраны, репозиторий отдаёт `domain.ErrNotFound`; перевод в 400/404/409 - в `writeError`.
4. **Порты и адаптеры** - 4 интерфейса. Разная семантика отказа видна в сервисе: ошибка спам-фильтра игнорируется (fail-open, `Create`), ошибка токсичности превращается в `domain.ToxicityUnknown` (fail-safe, `checkToxicity`). Адаптеры только возвращают ошибку сети.
5. **DDD** - доменная `Comment` отдельно от `commentRecord` в repository; `Author`/`Reputation` - отдельный контекст за `AuthorRepository`, участвует в `ApplyModeration(author, toxicity)`.

## Как проверить

```bash
cd practice/variant-5
go vet ./... && go test ./...     # тесты покрывают все ветки ТЗ, в т.ч. published и откат
go run .                          # ручная проверка curl из README.md
```

Заметка: JSON `GET`/`PATCH` сохраняет прежние имена полей (`ID`, `AuthorID`, ...), чтобы не менять контракт.
