# marketaux-go

[![CI](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml/badge.svg)](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/marketaux-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/marketaux-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tigusigalpa/marketaux-go)](https://goreportcard.com/report/github.com/tigusigalpa/marketaux-go)
[![codecov](https://codecov.io/gh/tigusigalpa/marketaux-go/branch/main/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/marketaux-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-blue.svg)](go.mod)

[English](README.md) | **Русский**

Идиоматичный Go-клиент для финансового новостного API
[Marketaux](https://www.marketaux.com) **без единой внешней зависимости**.
Покрывает все 10 документированных эндпоинтов: типизированные параметры,
типизированные ошибки, `context.Context` в каждом методе — только стандартная
библиотека.

## Возможности

- Все 10 эндпоинтов API: новостная лента, похожие новости, статья по UUID,
  статистика сущностей (временной ряд + агрегация), трендовые сущности,
  поиск сущностей, типы сущностей, отрасли, источники новостей
- `context.Context` в каждом методе; `Client` потокобезопасен
- Functional options: таймауты, свой `http.Client`, ретраи, rate limit,
  логирование, ограничение размера ответа
- Типизированные ошибки, совместимые с `errors.Is`/`errors.As`
- Автоматические ретраи с экспоненциальным backoff, jitter и `Retry-After`
- Клиентский token-bucket rate limiter
- Ленивая пагинация через `iter.Seq2` (Go 1.23+)
- Клиентская валидация всех документированных параметров
- Маскирование API-токена в ошибках и debug-логах
- **Ноль внешних зависимостей**

## Содержание

- [Требования](#требования)
- [Установка](#установка)
- [Быстрый старт](#быстрый-старт)
- [Конфигурация](#конфигурация)
- [Карта сервисов](#карта-сервисов)
- [Методы](#методы)
- [Параметры и валидация](#параметры-и-валидация)
- [Пагинация](#пагинация)
- [Обработка ошибок](#обработка-ошибок)
- [Ретраи и rate limits](#ретраи-и-rate-limits)
- [Тарифы и лимиты](#тарифы-и-лимиты)
- [Конкурентность](#конкурентность)
- [Тестирование](#тестирование)
- [Совместимость](#совместимость)
- [Безопасность](#безопасность)
- [Документация по эндпоинтам](#документация-по-эндпоинтам)
- [Участие в разработке](#участие-в-разработке)
- [Лицензия](#лицензия)

## Требования

- Go 1.23+
- API-токен Marketaux — получить на <https://www.marketaux.com/register>

## Установка

```bash
go get github.com/tigusigalpa/marketaux-go
```

## Быстрый старт

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tigusigalpa/marketaux-go"
)

func main() {
	client, err := marketaux.NewClient(os.Getenv("MARKETAUX_API_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	page, err := client.News().All(context.Background(), marketaux.NewsParams{
		Symbols:        []string{"TSLA", "AMZN"},
		Languages:      []string{"en"},
		FilterEntities: marketaux.Bool(true),
		SentimentGTE:   marketaux.Float64(0),
		Limit:          50,
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, a := range page.Data {
		fmt.Println(a.Title, "-", a.URL)
		for _, e := range a.Entities {
			fmt.Println("  ", e.Symbol, e.SentimentScore)
		}
	}
}
```

## Конфигурация

`NewClient(token, opts...)` принимает functional options:

| Опция | По умолчанию | Описание |
| --- | --- | --- |
| `WithBaseURL(raw)` | `https://api.marketaux.com/v1` | Свой базовый URL (тесты, прокси) |
| `WithHTTPClient(hc)` | `http.Client{Timeout: 30s}` | Свой HTTP-клиент (не мутируется) |
| `WithTimeout(d)` | `30s` | Таймаут запроса |
| `WithUserAgent(ua)` | `marketaux-go/1.0.0` | Заголовок `User-Agent` |
| `WithRetry(p)` | `DefaultRetryPolicy` | Политика ретраев; `MaxAttempts=1` отключает |
| `WithRateLimit(rps, burst)` | выкл. | Клиентский token bucket |
| `WithLogger(l)` | discard | `*slog.Logger` для debug-вывода |
| `WithDebug(bool)` | `false` | Debug-логи запросов (токен маскируется) |
| `WithMaxResponseBytes(n)` | `32 MiB` | Лимит размера тела ответа |

Переменные окружения для примеров: `MARKETAUX_API_TOKEN`,
`MARKETAUX_BASE_URL` (опционально, только smoke-тест).

```go
client, err := marketaux.NewClient(token,
	marketaux.WithTimeout(30*time.Second),
	marketaux.WithRetry(marketaux.DefaultRetryPolicy),
	marketaux.WithRateLimit(5, 2),
	marketaux.WithDebug(true),
)
```

## Карта сервисов

| Метод | HTTP-эндпоинт | Доступность | Документация |
| --- | --- | --- | --- |
| `News().All` | `GET /news/all` | все тарифы | [#finance-news](https://www.marketaux.com/documentation#finance-news) |
| `News().Similar` | `GET /news/similar/{uuid}` | все тарифы | [#similar-news](https://www.marketaux.com/documentation#similar-news) |
| `News().ByUUID` | `GET /news/uuid/{uuid}` | все тарифы | [#news-by-uuid](https://www.marketaux.com/documentation#news-by-uuid) |
| `Stats().Intraday` | `GET /entity/stats/intraday` | Standard+ | [#entity-stats](https://www.marketaux.com/documentation#entity-stats) |
| `Stats().Aggregation` | `GET /entity/stats/aggregation` | Standard+ | [#entity-aggregation](https://www.marketaux.com/documentation#entity-aggregation) |
| `Stats().Trending` | `GET /entity/trending/aggregation` | Standard+ | [#trending-aggregation](https://www.marketaux.com/documentation#trending-aggregation) |
| `Entities().Search` | `GET /entity/search` | все тарифы | [#entity-search](https://www.marketaux.com/documentation#entity-search) |
| `Entities().Types` | `GET /entity/type/list` | все тарифы | [#entity-types](https://www.marketaux.com/documentation#entity-types) |
| `Entities().Industries` | `GET /entity/industry/list` | все тарифы | [#entity-industries](https://www.marketaux.com/documentation#entity-industries) |
| `Sources().All` | `GET /news/sources` | все тарифы | [#sources](https://www.marketaux.com/documentation#sources) |

Ошибки: <https://www.marketaux.com/documentation#errors>

## Методы

### Новостная лента — `News().All(ctx, NewsParams)`

Последние мировые финансовые новости с анализом сущностей (match score,
sentiment, highlights). Документация:
<https://www.marketaux.com/documentation#finance-news>

```go
page, err := client.News().All(ctx, marketaux.NewsParams{
	Symbols:          []string{"TSLA", "AMZN", "MSFT"},
	EntityTypes:      []marketaux.EntityType{marketaux.EntityTypeEquity},
	Countries:        []string{"us"},
	Languages:        []string{"en"},
	SentimentGTE:     marketaux.Float64(0.1),
	MinMatchScore:    marketaux.Float64(10),
	FilterEntities:   marketaux.Bool(true),
	MustHaveEntities: marketaux.Bool(true),
	Search:           `"ipo" -nyse`,
	Domains:          []string{"wsj.com", "bloomberg.com"},
	PublishedOn:      marketaux.Date(2026, time.November, 13),
	Sort:             marketaux.NewsSortEntitySentimentScore,
	SortOrder:        marketaux.SortOrderDesc,
	Limit:            50,
	Page:             1,
})
```

Поля `NewsParams` соответствуют параметрам запроса 1:1: `symbols`,
`entity_types`, `industries`, `countries`, `sentiment_gte`, `sentiment_lte`,
`min_match_score`, `filter_entities`, `must_have_entities`, `group_similar`,
`search`, `domains`, `exclude_domains`, `source_ids`, `exclude_source_ids`,
`language`, `published_before`, `published_after`, `published_on`, `sort`,
`sort_order`, `limit`, `page`.

Операторы поиска: `+` (AND), `|` (OR), `-` (отрицание), `"..."` (точная
фраза), `*` в конце (префикс), `()` (группировка), `\` (экранирование).
Пример: `"ipo" -nyse`.

Важно: API возвращает только `snippet` и ссылку, но не полный текст статьи
(см. [FAQ Marketaux](https://www.marketaux.com/faq)). Кроме того, в
документации значение сортировки называется `published_on`, хотя поле ответа
— `published_at`; SDK следует документации буквально.

### Похожие новости — `News().Similar(ctx, uuid, SimilarParams)`

Статьи, похожие на статью с известным UUID. Те же фильтры, что у ленты, но
без `search`, `sort`, `sort_order` и `min_match_score` (API их не принимает,
и выразить их в `SimilarParams` невозможно).
Документация: <https://www.marketaux.com/documentation#similar-news>

```go
similar, err := client.News().Similar(ctx,
	"70cb577e-c2dd-4dde-b501-f713823a4939",
	marketaux.SimilarParams{Limit: 10},
)
```

### Статья по UUID — `News().ByUUID(ctx, uuid)`

Одна ранее сохранённая статья по UUID. Для несуществующих статей возвращает
`ErrNotFound`. Документация:
<https://www.marketaux.com/documentation#news-by-uuid>

```go
article, err := client.News().ByUUID(ctx,
	"9b9fb67e-9438-4263-815e-ef6a7d36af07")
```

### Временной ряд статистики — `Stats().Intraday(ctx, StatsParams)`

Статистика сущностей с разбивкой по `interval` (`minute`/`hour`/`day`/
`week`/`month`/`quarter`/`year`). Без `symbols` сервер возвращает лучшие
сущности по метрике. Требуется тариф Standard или выше.
Документация: <https://www.marketaux.com/documentation#entity-stats>

```go
series, err := client.Stats().Intraday(ctx, marketaux.StatsParams{
	Symbols:         []string{"TSLA", "AMZN", "MSFT"},
	Interval:        marketaux.IntervalDay,
	GroupBy:         marketaux.GroupBySymbol,
	Countries:       []string{"us"},
	PublishedAfter:  marketaux.Date(2026, time.October, 1),
	PublishedBefore: marketaux.Date(2026, time.October, 31),
	Sort:            marketaux.StatsSortTotalDocuments,
	DateOrder:       marketaux.DateOrderDesc,
})
```

Максимальный период для интервала: minute — 7 дней, hour — 31 день, day —
3 года, week/month — 5 лет, quarter/year — 10 лет. Больший период сервер
молча обрезает — см. `Interval.MaxDays()`.

### Агрегация статистики — `Stats().Aggregation(ctx, AggregationParams)`

Те же фильтры, что у временного ряда, но за один период. В `Meta` ответа
заполнены только `Returned` и `Limit`.
Документация: <https://www.marketaux.com/documentation#entity-aggregation>

```go
agg, err := client.Stats().Aggregation(ctx, marketaux.AggregationParams{
	GroupBy: marketaux.GroupByIndustry,
})
```

### Трендовые сущности — `Stats().Trending(ctx, TrendingParams)`

Сущности, набирающие популярность за период, с ранжированием по `Score`.
Документация: <https://www.marketaux.com/documentation#trending-aggregation>

```go
trending, err := client.Stats().Trending(ctx, marketaux.TrendingParams{
	Countries: []string{"us"},
})
for _, e := range trending.Data {
	fmt.Println(e.Key, e.Score)
}
```

### Поиск сущностей — `Entities().Search(ctx, EntitySearchParams)`

Поиск поддерживаемых сущностей по имени/символу. Лимит страницы фиксирован —
50. Документация: <https://www.marketaux.com/documentation#entity-search>

```go
found, err := client.Entities().Search(ctx, marketaux.EntitySearchParams{
	Search: "tesla",
	Types:  []marketaux.EntityType{marketaux.EntityTypeEquity},
})
```

### Типы сущностей — `Entities().Types(ctx)`

Все поддерживаемые типы (`equity`, `index`, `etf`, `mutualfund`, `currency`,
`cryptocurrency` и любые будущие — неизвестные значения безопасно декодируются
в `EntityType`). Документация:
<https://www.marketaux.com/documentation#entity-types>

### Отрасли — `Entities().Industries(ctx)`

Все поддерживаемые отрасли.
Документация: <https://www.marketaux.com/documentation#entity-industries>

### Источники новостей — `Sources().All(ctx, SourcesParams)`

Источники для фильтров `domains`/`source_ids`. Лимит страницы фиксирован —
50; `DistinctDomain` группирует по уникальным доменам (`SourceID` станет
`nil`). Документация: <https://www.marketaux.com/documentation#sources>

```go
src, err := client.Sources().All(ctx, marketaux.SourcesParams{
	DistinctDomain: marketaux.Bool(true),
	Languages:      []string{"en"},
})
```

## Параметры и валидация

- Опциональные bool/float — указатели (`*bool`, `*float64`): `nil` означает
  «не отправлять», сервер применит значение по умолчанию тарифа. Хелперы:
  `marketaux.Bool`, `marketaux.Float64`, `marketaux.Int`, `marketaux.Time`,
  `marketaux.Date`.
- `Limit`/`Page` равные `0` не отправляются.
- Списки кодируются через запятую с триммингом, дедупликацией и приведением
  к нижнему регистру там, где этого ждёт API (страны, языки, типы сущностей).
  Домены нормализуются (убираются схема и путь, хост в нижнем регистре).
- Валидация выполняется на клиенте и возвращает `*ValidationError` с картой
  `Fields` до отправки запроса:

```go
_, err := client.News().All(ctx, marketaux.NewsParams{
	PublishedOn:     marketaux.Date(2026, 11, 1),
	PublishedBefore: marketaux.Date(2026, 11, 2), // конфликт
})
var verr *marketaux.ValidationError
if errors.As(err, &verr) {
	for field, reason := range verr.Fields {
		fmt.Println(field, reason)
	}
}
```

Проверяемые правила: sentiment в `[-1, 1]`, `published_on` несовместим с
диапазоном дат, `published_after < published_before`, `sort_order` только с
сортировками по score сущностей, `relevance_score` требует `search`,
валидные enum'ы и `page*limit <= 20000` (ограничение API).

Поддерживаемые языки: `ar bg bn cs da de el en es et fa fi fr he hi hr hu id
it ja ko lt multi nl no pl pt ro ru sk sv ta th tr uk vi zh` (`multi` —
мультиязычные источники). Список может расширяться — SDK не отклоняет
неизвестные языки.

## Пагинация

`Paginate` возвращает итератор `iter.Seq2`, который лениво подгружает
страницы:

```go
for article, err := range client.News().Paginate(ctx, marketaux.NewsParams{
	Symbols: []string{"TSLA"},
}) {
	if err != nil {
		return err
	}
	fmt.Println(article.Title)
}
```

Итерация останавливается на последней странице (`returned < limit`), на
пустых данных, на границе в 20 000 результатов, на ошибке (она один раз
отдаётся вторым значением) или при `break`. `Entities().Paginate` и
`Sources().Paginate` работают так же с фиксированным лимитом 50.

Для жадной загрузки используйте `News().AllPages(ctx, params, maxPages)`.

## Обработка ошибок

Любая ошибка — типизированное значение для `errors.Is`/`errors.As`:

```go
page, err := client.News().All(ctx, params)
switch {
case err == nil:
	// ok
case errors.Is(err, marketaux.ErrRateLimit):
	var apiErr *marketaux.APIError
	if errors.As(err, &apiErr) {
		fmt.Println("limit:", apiErr.RateLimitLimit, "retry after:", apiErr.RetryAfter)
	}
case errors.Is(err, marketaux.ErrUsageLimit):
	// исчерпана дневная квота тарифа
case errors.Is(err, marketaux.ErrForbidden):
	// эндпоинт недоступен на вашем тарифе (например, stats на Free/Basic)
case errors.Is(err, marketaux.ErrValidation):
	// ошибка параметров на клиенте
default:
	return err
}
```

| Sentinel | HTTP/код |
| --- | --- |
| `ErrInvalidConfig` | ошибка `NewClient`/опций |
| `ErrValidation` | клиентская валидация параметров |
| `ErrTransport` | сетевые сбои |
| `ErrTimeout` | таймаут запроса |
| `ErrDecode` | невалидный JSON в ответе |
| `ErrBadRequest` | 400 `malformed_parameters` |
| `ErrUnauthorized` | 401 `invalid_api_token` |
| `ErrUsageLimit` | 402 `usage_limit_reached` |
| `ErrForbidden` | 403 `endpoint_access_restricted` |
| `ErrNotFound` | 404 `resource_not_found` |
| `ErrEndpointNotFound` | 404 `invalid_api_endpoint` |
| `ErrRateLimit` | 429 `rate_limit_reached` |
| `ErrServer` | 5xx `server_error` |
| `ErrMaintenance` | 503 `maintenance_mode` |

`*APIError` содержит `StatusCode`, `Code`, `Message`, `URL` (с маской),
`RequestID`, `RateLimitLimit`, `UsageLimitLimit`, `RetryAfter` и `Body`
(обрезано).

## Ретраи и rate limits

`DefaultRetryPolicy`: 3 попытки, базовая задержка 500 мс, потолок 8 с, полный
jitter. Ретраятся: `429`, `500`, `502`, `503`, `504` и сетевые ошибки. Не
ретраятся: `4xx` (кроме 429), таймауты и отменённые контексты. `Retry-After`
(секунды или HTTP-date) имеет приоритет, если больше вычисленного backoff.

Сервер сообщает лимиты в заголовках `X-RateLimit-Limit` (запросов за 60 с,
при 429) и `X-UsageLimit-Limit` (дневная квота, при 402) — оба доступны в
`*APIError`.

Для превентивного троттлинга используйте `WithRateLimit(rps, burst)` —
локальный token bucket, безопасный для конкурентного использования.

## Тарифы и лимиты

| Тариф | Цена/мес | Запросов в день | Статей за запрос | Сущностей за stat-запрос | Stats API |
| --- | --- | --- | --- | --- | --- |
| Free | $0 | 100 | 3 | 0 | нет |
| Basic | $29 | 2 500 | 20 | 0 | нет |
| Standard | $49 | 10 000 | 50 | 20 | да |
| Pro | $99 | 25 000 | 100 | 100 | да |
| Pro 50K | $199 | 50 000 | 100 | 100 | да |

Источник: <https://www.marketaux.com/pricing>. На Free/Basic stats-эндпоинты
возвращают `403 endpoint_access_restricted`. У entity search и sources
лимит страницы фиксирован — 50. `page × limit` не должен превышать 20 000.

## Конкурентность

`Client` (и все его сервисы) неизменяем после создания и безопасен для
совместного использования из горутин. Проверено под `go test -race`.

## Тестирование

```bash
go test ./...        # юнит-тесты (только httptest, без сети)
go test -race ./...  # детектор гонок
make cover           # отчёт о покрытии (≥90%)

# живой smoke-тест (опционально):
MARKETAUX_API_TOKEN=... go test -tags=integration ./...
MARKETAUX_API_TOKEN=... go run ./examples/smoke
```

## Совместимость

Go 1.23–1.27, тестируется в CI на каждой поддерживаемой минорной версии. В
`go.mod` нет ни одного `require` — только стандартная библиотека. Ветка v1
сохраняет обратную совместимость публичного API.

## Безопасность

- `api_token` передаётся только query-параметром — API не поддерживает
  заголовок `Authorization`.
- Токен маскируется (`api_token=***`) в URL ошибок `*APIError`,
  `TransportError`, `DecodeError` и в debug-логах.
- Никогда не коммитьте токены; `.env` в `.gitignore`.
- Об уязвимостях сообщайте приватно — см. [SECURITY.md](SECURITY.md).

## Документация по эндпоинтам

| Эндпоинт | Документация |
| --- | --- |
| `/news/all` | https://www.marketaux.com/documentation#finance-news |
| `/news/similar/{uuid}` | https://www.marketaux.com/documentation#similar-news |
| `/news/uuid/{uuid}` | https://www.marketaux.com/documentation#news-by-uuid |
| `/entity/stats/intraday` | https://www.marketaux.com/documentation#entity-stats |
| `/entity/stats/aggregation` | https://www.marketaux.com/documentation#entity-aggregation |
| `/entity/trending/aggregation` | https://www.marketaux.com/documentation#trending-aggregation |
| `/entity/search` | https://www.marketaux.com/documentation#entity-search |
| `/entity/type/list` | https://www.marketaux.com/documentation#entity-types |
| `/entity/industry/list` | https://www.marketaux.com/documentation#entity-industries |
| `/news/sources` | https://www.marketaux.com/documentation#sources |

## Участие в разработке

См. [CONTRIBUTING.md](CONTRIBUTING.md). Баг-репорты и фича-реквесты —
в GitHub issues. Никогда не включайте API-токен в репорт.

## Лицензия

MIT © [Igor Sazonov](https://github.com/tigusigalpa) — см. [LICENSE](LICENSE).
