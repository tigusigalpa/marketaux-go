# marketaux-go

[![CI](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml/badge.svg)](https://github.com/tigusigalpa/marketaux-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/tigusigalpa/marketaux-go.svg)](https://pkg.go.dev/github.com/tigusigalpa/marketaux-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/tigusigalpa/marketaux-go)](https://goreportcard.com/report/github.com/tigusigalpa/marketaux-go)
[![codecov](https://codecov.io/gh/tigusigalpa/marketaux-go/branch/main/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/marketaux-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.23-blue.svg)](go.mod)

[English](README.md) | **Русский**

Подключите финансовые новости [Marketaux](https://www.marketaux.com) к своему
Go-приложению: находите статьи о компаниях, следите за настроениями рынка и
узнавайте, какие активы привлекают внимание. `marketaux-go` берёт на себя
HTTP-запросы, переход между страницами и повторные попытки при временных сбоях.
Вы работаете с обычными структурами Go.

Пакет охватывает все 10 документированных эндпоинтов, использует только
стандартную библиотеку и требует Go 1.23 или новее. Это независимый SDK,
который развивается отдельно от Marketaux.

API возвращает метаданные статей, анализ упомянутых сущностей, короткие
фрагменты текста и ссылки на публикации. Полного текста статей в API нет;
подробнее — в [FAQ Marketaux](https://www.marketaux.com/faq).

## Возможности

- Все 10 эндпоинтов: новости, похожие статьи, статья по UUID, статистика,
  трендовые сущности, поиск сущностей, типы, отрасли и источники.
- `context.Context` во всех сетевых методах; один клиент можно использовать
  из нескольких горутин.
- Настраиваемые таймауты, HTTP-клиент, ретраи, частота запросов, журналирование
  и ограничение размера ответа.
- Типизированные параметры с проверкой недопустимых значений и сочетаний.
- Типизированные ошибки для `errors.Is` и `errors.As`.
- Экспоненциальная задержка повторных попыток, jitter и поддержка `Retry-After`.
- Ленивая пагинация через `iter.Seq2` и локальное ограничение частоты запросов.
- Маскирование токена в диагностике SDK.
- Ноль внешних зависимостей.

## Содержание

- [Требования](#требования)
- [Установка](#установка)
- [Быстрый старт](#быстрый-старт)
- [Конфигурация](#конфигурация)
- [Карта сервисов](#карта-сервисов)
- [Методы](#методы)
- [Как читать ответы](#как-читать-ответы)
- [Параметры и валидация](#параметры-и-валидация)
- [Пагинация](#пагинация)
- [Обработка ошибок](#обработка-ошибок)
- [Ретраи и ограничения частоты](#ретраи-и-ограничения-частоты)
- [Тарифы и лимиты](#тарифы-и-лимиты)
- [Конкурентность](#конкурентность)
- [Тестирование](#тестирование)
- [Совместимость](#совместимость)
- [Безопасность](#безопасность)
- [Частые проблемы](#частые-проблемы)
- [Примеры](#примеры)
- [Документация по эндпоинтам](#документация-по-эндпоинтам)
- [Участие в разработке](#участие-в-разработке)
- [Лицензия](#лицензия)

## Требования

- Go 1.23+. Для production используйте поддерживаемую версию с актуальными
  исправлениями безопасности.
- [API-токен Marketaux](https://www.marketaux.com/register). В быстром старте
  запрашиваются 3 статьи — такой лимит подходит для бесплатного тарифа.

## Установка

```bash
go get github.com/tigusigalpa/marketaux-go
```

## Быстрый старт

Задайте токен в терминале, из которого будете запускать программу:

```bash
export MARKETAUX_API_TOKEN='your-token'
```

В PowerShell:

```powershell
$env:MARKETAUX_API_TOKEN = 'your-token'
```

Сохраните код в `main.go` внутри своего Go-модуля и выполните `go run .`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tigusigalpa/marketaux-go"
)

func main() {
	client, err := marketaux.NewClient(os.Getenv("MARKETAUX_API_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	defer client.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	page, err := client.News().All(ctx, marketaux.NewsParams{
		Symbols:        []string{"TSLA", "AMZN"},
		Languages:      []string{"en"},
		FilterEntities: marketaux.Bool(true),
		SentimentGTE:   marketaux.Float64(0),
		Limit:          3,
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

Создайте один клиент и используйте его повторно. Успешный запрос может вернуть
пустой `page.Data` — это нормальный результат. Библиотека сама не читает
переменные окружения и `.env`: приложение передаёт токен в `NewClient`.

## Конфигурация

`NewClient(token, opts...)` принимает следующие опции:

| Опция | По умолчанию | Назначение |
| --- | --- | --- |
| `WithBaseURL(raw)` | `https://api.marketaux.com/v1` | Базовый URL для прокси и тестов |
| `WithHTTPClient(hc)` | `http.Client{Timeout: 30s}` | Поверхностная копия; транспорт и cookie jar остаются общими |
| `WithTimeout(d)` | `30s` | Таймаут одной HTTP-попытки, включая чтение тела ответа |
| `WithUserAgent(ua)` | `marketaux-go/1.0.0` | Заголовок `User-Agent` |
| `WithRetry(p)` | `DefaultRetryPolicy` | Политика ретраев; `MaxAttempts=1` отключает повторы |
| `WithRateLimit(rps, burst)` | выключено | Локальное ограничение частоты запросов |
| `WithLogger(l)` | вывод отбрасывается | Логгер `*slog.Logger` |
| `WithDebug(bool)` | `false` | Диагностика запросов с маскированием токена |
| `WithMaxResponseBytes(n)` | `32 MiB` | Максимальный размер тела ответа |

Примеры используют `MARKETAUX_API_TOKEN`. Smoke-пример и интеграционный тест
также поддерживают необязательную `MARKETAUX_BASE_URL`.

Опции применяются слева направо. У своего HTTP-клиента должен быть положительный
`Timeout`; иначе укажите `WithTimeout` после `WithHTTPClient`:

```go
client, err := marketaux.NewClient(token,
	marketaux.WithHTTPClient(&http.Client{Transport: http.DefaultTransport}),
	marketaux.WithTimeout(10*time.Second),
	marketaux.WithRateLimit(5, 2),
)
if err != nil {
	return err
}
```

`WithTimeout` ограничивает отдельную попытку. Чтобы задать бюджет всей операции,
включая ожидание лимитера и ретраев, используйте `context.WithTimeout`.
При пагинации этот контекст действует на весь обход страниц.

Для журналирования передайте логгер с обработчиком уровня debug:

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))
client, err := marketaux.NewClient(token,
	marketaux.WithLogger(logger),
	marketaux.WithDebug(true),
)
if err != nil {
	return err
}
```

Сам по себе `WithDebug(true)` не создаёт место для вывода логов. SDK пишет
метод, URL с маской, статус, номер попытки и длительность. Тела ответов
в журнал не попадают.

`WithBaseURL` принимает HTTP(S)-адрес, сохраняет префикс пути, удаляет query
и fragment. Вне локальных тестов используйте HTTPS. Адреса с логином или
паролем отклоняются. Редиректы разрешены в пределах исходных схемы, хоста
и порта. Переход на другой origin блокируется до отправки запроса, в том
числе при собственном обработчике `CheckRedirect`.

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

Ошибки API: [официальный справочник](https://www.marketaux.com/documentation#errors).

## Методы

Фрагменты ниже предполагают, что `client` и `ctx` уже созданы, а окружающая
функция возвращает ошибку. Полные программы находятся в [examples](examples/README.md).

### Новостная лента — `News().All(ctx, NewsParams)`

Статьи с фильтрами по компаниям, настроению, языкам, источникам и датам.
Документация: [Finance news](https://www.marketaux.com/documentation#finance-news).

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
if err != nil {
	return err
}
for _, article := range page.Data {
	fmt.Println(article.Title, article.PublishedAt, article.URL)
}
```

Поля `NewsParams` соответствуют параметрам API: `symbols`, `entity_types`,
`industries`, `countries`, `sentiment_gte`, `sentiment_lte`, `min_match_score`,
`filter_entities`, `must_have_entities`, `group_similar`, `search`, `domains`,
`exclude_domains`, `source_ids`, `exclude_source_ids`, `language`,
`published_before`, `published_after`, `published_on`, `sort`, `sort_order`,
`limit`, `page`.

Операторы поиска: `+` — AND, `|` — OR, `-` — исключение, `"..."` — фраза,
`*` в конце — префикс, `()` — группировка, `\` — экранирование.
Например, `"ipo" -nyse`. Значение сортировки по дате в документации называется
`published_on`, а поле ответа — `published_at`; SDK сохраняет эти имена.

### Похожие новости — `News().Similar(ctx, uuid, SimilarParams)`

Статьи, похожие на публикацию с указанным UUID. Поддерживает фильтры ленты,
кроме `Search`, `MinMatchScore`, `Sort` и `SortOrder`; этих полей нет
в `SimilarParams`. Неизвестный UUID может дать пустую выдачу или 404.
Документация: [Similar news](https://www.marketaux.com/documentation#similar-news).

```go
similar, err := client.News().Similar(ctx,
	"70cb577e-c2dd-4dde-b501-f713823a4939",
	marketaux.SimilarParams{Limit: 10},
)
if err != nil {
	return err
}
fmt.Println(len(similar.Data), "similar articles")
```

### Статья по UUID — `News().ByUUID(ctx, uuid)`

Одна статья без обёртки `Data`. Для отсутствующей статьи возвращается
`ErrNotFound`. Документация: [News by UUID](https://www.marketaux.com/documentation#news-by-uuid).

```go
article, err := client.News().ByUUID(ctx,
	"9b9fb67e-9438-4263-815e-ef6a7d36af07")
if err != nil {
	return err
}
fmt.Println(article.Title, article.Snippet)
```

### Временной ряд — `Stats().Intraday(ctx, StatsParams)`

Статистика по интервалам `minute`, `hour`, `day`, `week`, `month`, `quarter`
или `year`. Без `Symbols` сервер выбирает сущности по метрике сортировки.
Нужен Standard или выше. Документация: [Entity stats](https://www.marketaux.com/documentation#entity-stats).

```go
series, err := client.Stats().Intraday(ctx, marketaux.StatsParams{
	Symbols:         []string{"TSLA", "AMZN", "MSFT"},
	Interval:        marketaux.IntervalDay,
	GroupBy:         marketaux.GroupBySymbol,
	Countries:       []string{"us"},
	PublishedAfter:  marketaux.Date(2026, time.October, 1),
	PublishedBefore: marketaux.Date(2026, time.November, 1),
	Sort:            marketaux.StatsSortTotalDocuments,
	DateOrder:       marketaux.DateOrderDesc,
})
if err != nil {
	return err
}
for _, point := range series.Data {
	fmt.Println(point.Date, point.Data)
}
```

Максимальный период: minute — 7 дней, hour — 31 день, day — 3 года,
week/month — 5 лет, quarter/year — 10 лет. Больший период сервер обрезает;
см. `Interval.MaxDays()`. Даты с нулевым количеством документов включаются
в результат.

### Агрегация — `Stats().Aggregation(ctx, AggregationParams)`

Те же фильтры за один период; без `Interval` и `DateOrder`. В `Meta` заполнены
только `Returned` и `Limit`. Документация: [Entity aggregation](https://www.marketaux.com/documentation#entity-aggregation).

```go
agg, err := client.Stats().Aggregation(ctx, marketaux.AggregationParams{
	GroupBy: marketaux.GroupByIndustry,
})
if err != nil {
	return err
}
for _, stat := range agg.Data {
	fmt.Println(stat.Key, stat.TotalDocuments)
}
```

### Тренды — `Stats().Trending(ctx, TrendingParams)`

Сущности, которые набирают популярность за выбранный период; поле `Score`
содержит оценку тренда. Документация: [Trending aggregation](https://www.marketaux.com/documentation#trending-aggregation).

```go
trending, err := client.Stats().Trending(ctx, marketaux.TrendingParams{
	Countries: []string{"us"},
})
if err != nil {
	return err
}
for _, e := range trending.Data {
	fmt.Println(e.Key, e.Score)
}
```

### Поиск сущностей — `Entities().Search(ctx, EntitySearchParams)`

Находите поддерживаемые компании и активы по названию, символу и фильтрам.
Размер страницы фиксирован: 50. Документация: [Entity search](https://www.marketaux.com/documentation#entity-search).

```go
found, err := client.Entities().Search(ctx, marketaux.EntitySearchParams{
	Search: "tesla",
	Types:  []marketaux.EntityType{marketaux.EntityTypeEquity},
})
if err != nil {
	return err
}
for _, entity := range found.Data {
	fmt.Println(entity.Symbol, entity.Name, entity.Country)
}
```

### Типы сущностей — `Entities().Types(ctx)`

Список типов: `equity`, `index`, `etf`, `mutualfund`, `currency`,
`cryptocurrency` и возможные будущие значения.
Документация: [Entity types](https://www.marketaux.com/documentation#entity-types).

```go
types, err := client.Entities().Types(ctx)
if err != nil {
	return err
}
fmt.Println(types)
```

### Отрасли — `Entities().Industries(ctx)`

Список поддерживаемых отраслей.
Документация: [Entity industries](https://www.marketaux.com/documentation#entity-industries).

```go
industries, err := client.Entities().Industries(ctx)
if err != nil {
	return err
}
fmt.Println(industries)
```

Типы и отрасли удобно кэшировать на стороне приложения. SDK выполняет запрос
при каждом вызове и не хранит такой кэш.

### Источники — `Sources().All(ctx, SourcesParams)`

Источники для фильтров `Domains` и `SourceIDs`. Размер страницы — 50.
При `DistinctDomain=true` результат группируется по доменам, а `SourceID`
может быть `nil`. Документация: [Sources](https://www.marketaux.com/documentation#sources).

```go
src, err := client.Sources().All(ctx, marketaux.SourcesParams{
	DistinctDomain: marketaux.Bool(true),
	Languages:      []string{"en"},
})
if err != nil {
	return err
}
for _, source := range src.Data {
	fmt.Println(source.Domain, source.Language)
}
```

## Как читать ответы

| Метод | Результат | Основные поля |
| --- | --- | --- |
| Лента / похожие новости | `*NewsResponse` | `Meta`, `Data []Article` |
| Статья по UUID | `*Article` | `Title`, `Snippet`, `URL`, `PublishedAt`, `Entities`, `Similar` |
| Временной ряд | `*TimeSeriesResponse` | `Data []TimeSeriesPoint`; у точки есть `Date` и `Data []EntityStat` |
| Агрегация | `*StatsResponse` | `Data []EntityStat`, `Meta.Returned`, `Meta.Limit` |
| Тренды | `*TrendingResponse` | `Data []TrendingEntity`, поле `Score` и встроенный `EntityStat` |
| Поиск сущностей | `*EntitiesResponse` | `Data []EntityInfo`, `Meta` |
| Источники | `*SourcesResponse` | `Data []Source`, `Meta` |
| Типы / отрасли | `[]EntityType` / `[]string` | Сразу список без обёртки |

`Meta.Found` — размер подходящей выборки; он может превышать окно пагинации
в 20 000 результатов. `Meta.Returned` — количество на текущей странице,
`Meta.Limit` — её фактический размер. У временного ряда `Meta` нет.

Даты декодируются в `time.Time`, включая дробные секунды. Проверяйте на `nil`
необязательные `Article.RelevanceScore`, `Entity.Exchange`,
`Entity.ExchangeLong`, `EntityStat.SentimentAvg` и `Source.SourceID`.
Текстовые поля вроде `Snippet` при JSON null получают пустую строку.
Пустые и null-коллекции можно обходить через range, но null-слайс может остаться nil.

Sentiment относится к упоминанию сущности: одна статья может иметь разные
оценки для разных компаний. `Entity.IsPositive()` и `IsNegative()` проверяют
знак оценки. `Article.IsAbout("TSLA")` сравнивает символы без учёта регистра.
`Highlight.Text()` убирает `<em>`, раскрывает HTML-сущности и нормализует
пробелы. При вставке результата в HTML его нужно экранировать.

Неизвестные типы сущностей в ответах сохраняются. `Entity.TypeKnown()`
и `EntityType.Valid()` показывают, знакомо ли SDK значение. Неизвестные
JSON-поля игнорируются для совместимости с расширениями API.

## Параметры и валидация

- Необязательные bool/float задаются через указатели: `nil` не отправляется,
  `Bool(false)` и `Float64(0)` отправляют явно заданное значение.
- `Limit` и `Page`, равные нулю, не отправляются в обычных запросах:
  сервер применяет значения по умолчанию.
- Списки кодируются через запятую: пробелы по краям и дубли убираются.
  Страны, языки и типы приводятся к нижнему регистру; у доменов убираются
  схема и путь, имя хоста приводится к нижнему регистру.
- Ошибки параметров возвращаются как `*ValidationError` с картой `Fields`
  до сетевого запроса.

```go
_, err := client.News().All(ctx, marketaux.NewsParams{
	PublishedOn:     marketaux.Date(2026, 11, 1),
	PublishedBefore: marketaux.Date(2026, 11, 2), // conflict
})
var verr *marketaux.ValidationError
if errors.As(err, &verr) {
	for field, reason := range verr.Fields {
		fmt.Println(field, reason)
	}
}
```

Проверяются sentiment в `[-1, 1]`, конечность чисел, неотрицательный
`MinMatchScore`, enum-значения, отрицательные номера страниц и лимиты,
`page × limit <= 20000`. `PublishedOn` несовместим с диапазоном дат;
`PublishedAfter` должен быть раньше `PublishedBefore`. Для новостей
`SortOrder` разрешён только с сортировками по оценке сущностей,
а `relevance_score` требует `Search`.

Доступ к эндпоинтам и ограничения тарифа проверяет Marketaux: SDK не знает,
какой план подключён к вашему токену.

### Справочник параметров

| Поля Go | Параметры запроса | Где доступны |
| --- | --- | --- |
| `Symbols`, `EntityTypes`, `Industries`, `Countries` | `symbols`, `entity_types`, `industries`, `countries` | Новости, похожие, вся статистика |
| `SentimentGTE`, `SentimentLTE` | `sentiment_gte`, `sentiment_lte` | Новости, похожие, вся статистика |
| `Domains`, `ExcludeDomains` | `domains`, `exclude_domains` | Новости, похожие, вся статистика |
| `SourceIDs`, `ExcludeSourceIDs` | `source_ids`, `exclude_source_ids` | Новости, похожие, вся статистика |
| `Languages` | `language`, в единственном числе | Новости, похожие, вся статистика, источники |
| `PublishedBefore`, `PublishedAfter`, `PublishedOn` | `published_before`, `published_after`, `published_on` | Новости, похожие, вся статистика |
| `FilterEntities`, `MustHaveEntities`, `GroupSimilar` | `filter_entities`, `must_have_entities`, `group_similar` | Новости, похожие |
| `MinMatchScore` | `min_match_score` | Новости |
| `Search` | `search` | Новости, вся статистика, поиск сущностей |
| `Sort`, `SortOrder` | `sort`, `sort_order` | Новости и статистика; типы enum различаются |
| `Limit` | `limit` | Новости, похожие, вся статистика |
| `Page` | `page` | Новости, похожие, поиск сущностей, источники |
| `GroupBy`, `MinDocCount` | `group_by`, `min_doc_count` | Вся статистика |
| `SentimentAvgGTE`, `SentimentAvgLTE` | `sentiment_avg_gte`, `sentiment_avg_lte` | Вся статистика |
| `Interval`, `DateOrder` | `interval`, `date_order` | Временной ряд |
| `Search`, `Symbols`, `Exchanges`, `Types`, `Industries`, `Countries` | `search`, `symbols`, `exchanges`, `types`, `industries`, `countries` | Поиск сущностей |
| `DistinctDomain` | `distinct_domain` | Источники |

Точные типы полей — в [Go reference](https://pkg.go.dev/github.com/tigusigalpa/marketaux-go).

`Countries` фильтрует по стране биржи сущности, а `Languages` — по языку
источника. `FilterEntities` оставляет внутри статьи только сущности,
соответствующие фильтрам. `MustHaveEntities` требует хотя бы одну распознанную
сущность. `GroupSimilar` управляет группировкой похожих статей в поле `Similar`.

Явный ноль отличается от отсутствующего значения:

```go
params := marketaux.NewsParams{
	FilterEntities: marketaux.Bool(false), // explicitly send false
	SentimentGTE:   marketaux.Float64(0),  // explicitly include neutral sentiment
	PublishedOn:    marketaux.Date(2026, time.November, 1),
}
```

Хелперы `Bool`, `Float64`, `Time`, `Date`, а также общая функция `Int`
возвращают указатели. `Time(t)` сохраняет момент времени, `Date(y, m, d)`
создаёт полночь UTC. Before/after отправляются в UTC с точностью до секунды;
`PublishedOn` — как календарная дата UTC. API принимает и укороченные даты,
но SDK использует один канонический формат. Верхняя граница в полночь
не включает остаток этого дня.

Примеры языков: `ar bg bn cs da de el en es et fa fi fr he hi hr hu id it ja
ko lt multi nl no pl pt ro ru sk sv ta th tr uk vi zh`. `multi` обозначает
мультиязычный источник. Список может расширяться; неизвестный язык не
отклоняется SDK.

## Пагинация

`Paginate` возвращает ленивый итератор `iter.Seq2`. До начала обхода запросов
нет; `break` не подгружает следующую страницу:

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

Начальная страница берётся из `params.Page`; ноль означает страницу 1.
Если тариф уменьшил размер страницы, SDK использует фактический `Meta.Limit`
в следующих запросах. Обход заканчивается при `returned < limit`, пустых
данных, ошибке или перед страницей, которая нарушила бы
`page × limit <= 20000`. При размере 3 последняя допустимая страница — 6666.
Для более широкой выборки разбивайте её фильтрами, например по датам.

Ошибка отдаётся один раз вторым значением, после чего итератор завершается.
`Entities().Paginate` и `Sources().Paginate` работают так же, с размером
страницы 50. Для похожих новостей задавайте `SimilarParams.Page` вручную.

Если нужен готовый слайс:

```go
articles, err := client.News().AllPages(ctx, marketaux.NewsParams{
	Symbols: []string{"TSLA"},
	Limit:   3,
}, 5) // at most 5 pages
if err != nil {
	return err
}
fmt.Println(len(articles))
```

При `maxPages <= 0` загружается не больше 400 страниц — это может быть
меньше всей выдачи. `AllPages` хранит результат в памяти и при ошибке
возвращает `nil`, отбрасывая уже загруженное. Итератор позволяет сохранить
то, что приложение успело обработать. Каждая страница и возможные ретраи
расходуют дополнительные API-запросы.

## Обработка ошибок

`errors.Is` помогает выбрать действие, `errors.As` — получить подробности.
Отмена и дедлайн контекста сохраняют стандартные ошибки Go:

```go
page, err := client.News().All(ctx, params)
switch {
case err == nil:
	fmt.Println(len(page.Data), "articles")
case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
	return err
case errors.Is(err, marketaux.ErrRateLimit):
	var apiErr *marketaux.APIError
	if errors.As(err, &apiErr) {
		fmt.Println("limit:", apiErr.RateLimitLimit, "retry after:", apiErr.RetryAfter)
	}
case errors.Is(err, marketaux.ErrUsageLimit):
	// daily plan quota exhausted
case errors.Is(err, marketaux.ErrForbidden):
	// endpoint not on your plan (e.g. stats on Free/Basic)
case errors.Is(err, marketaux.ErrValidation):
	// client-side parameter problem
default:
	return err
}
```

| Sentinel | Значение |
| --- | --- |
| `ErrInvalidConfig` | Ошибка `NewClient` или опций |
| `ErrValidation` | Недопустимые параметры запроса |
| `ErrTransport` | Ошибка сети или чтения ответа |
| `ErrTimeout` | Таймаут HTTP-клиента |
| `ErrDecode` | Невалидный JSON или неожиданный тип поля |
| `ErrResponseTooLarge` | Превышен `WithMaxResponseBytes`; также соответствует `ErrTransport` |
| `ErrBadRequest` | 400 `malformed_parameters` |
| `ErrUnauthorized` | 401 `invalid_api_token` |
| `ErrUsageLimit` | 402 `usage_limit_reached` |
| `ErrForbidden` | 403 `endpoint_access_restricted` |
| `ErrNotFound` | 404 `resource_not_found` |
| `ErrEndpointNotFound` | 404 `invalid_api_endpoint` |
| `ErrRateLimit` | 429 `rate_limit_reached` |
| `ErrServer` | 5xx, кроме 503 |
| `ErrMaintenance` | 503 `maintenance_mode` |

`*APIError` содержит `StatusCode`, `Code`, `Message`, URL с маской,
`RequestID`, `RateLimitLimit`, `UsageLimitLimit`, `RetryAfter` и `Body`
с удалённым токеном, обрезанное до 2 000 байт. У `*DecodeError` есть
аналогичный фрагмент до 500 байт. Непройденный редирект возвращается как
`*APIError` с исходным статусом 3xx.

Таймаут HTTP-клиента соответствует `ErrTimeout` и `ErrTransport`. Дедлайн
контекста вызывающей стороны соответствует `context.DeadlineExceeded`.
Учитывайте обе ситуации при решении о повторах на уровне приложения.

## Ретраи и ограничения частоты

`DefaultRetryPolicy` делает до 3 попыток суммарно. Начальная задержка —
500 мс, потолок backoff — 8 секунд. Jitter выбирает от 50% до 100%
вычисленной задержки; это equal jitter.

Повторяются `429`, `500`, `502`, `503`, `504` и сетевые сбои. Не повторяются
остальные 4xx, таймауты, отменённые контексты, ошибки декодирования
и слишком большие ответы. `Retry-After` принимается в секундах или как
HTTP-date и имеет приоритет, если он больше рассчитанной задержки.

Копируйте политику по умолчанию перед изменением:

```go
policy := marketaux.DefaultRetryPolicy
policy.MaxAttempts = 4
policy.BaseDelay = 250 * time.Millisecond
client, err := marketaux.NewClient(token, marketaux.WithRetry(policy))
if err != nil {
	return err
}
// To disable: marketaux.WithRetry(marketaux.RetryPolicy{MaxAttempts: 1})
```

`MaxDelay` ограничивает backoff SDK, но не уменьшает больший `Retry-After`.
Дедлайн контекста прерывает любое ожидание. Если добавляете собственный цикл
повторов, учитывайте общее число попыток вместе с ретраями SDK.

При 429 сервер сообщает `X-RateLimit-Limit` — лимит запросов за 60 секунд;
при 402 — дневную квоту `X-UsageLimit-Limit`. Оба заголовка доступны
в `*APIError`.

`WithRateLimit(rps, burst)` задаёт среднюю частоту и допустимый начальный
всплеск. Лимитер применяется к каждой попытке, включая ретраи. Он работает
только внутри одного клиента: не координирует процессы и не контролирует
общую дневную квоту аккаунта. Отмена ожидания не занимает будущую ёмкость.
Повторные запросы могут расходовать дневную квоту.

## Тарифы и лимиты

Цены при помесячной оплате и лимиты проверены 17 ноября 2026 года. Перед
выбором тарифа сверяйтесь с [актуальными условиями](https://www.marketaux.com/pricing).

| Тариф | Цена/месяц | Запросов/день | Статей/запрос | Сущностей/stat-запрос | Stats API |
| --- | --- | --- | --- | --- | --- |
| Free | $0 | 100 | 3 | 0 | нет |
| Basic | $29 | 2 500 | 20 | 0 | нет |
| Standard | $49 | 10 000 | 50 | 20 | да |
| Pro | $99 | 25 000 | 100 | 100 | да |
| Pro 50K | $199 | 50 000 | 100 | 100 | да |

На Free/Basic статистика недоступна; ограничение доступа возвращается как
403. У поиска сущностей и источников размер страницы фиксирован: 50.
Окно пагинации ограничено `page × limit <= 20000`.

## Конкурентность

После создания `Client` и его сервисы безопасны для совместного использования
из горутин. Проверяется под `go test -race`.

Собственный транспорт, обработчик логов и cookie jar также должны быть
потокобезопасными. Не меняйте общий транспорт, `DefaultRetryPolicy`, слайсы
и значения по указателям в параметрах, пока запросы их используют. Ответы
принадлежат вызывающему коду.

`CloseIdleConnections` освобождает простаивающие соединения транспорта,
не отменяя активные запросы и не закрывая клиент навсегда. При общем
транспорте могут закрыться и его другие простаивающие соединения.

## Тестирование

```bash
go test ./...        # локальные HTTP-серверы, без обращений к Marketaux
go test -race ./...  # детектор гонок
make cover           # покрытие библиотеки, целевой уровень >= 90%

# Опциональные проверки с настоящим токеном:
MARKETAUX_API_TOKEN=... go test -tags=integration ./...
MARKETAUX_API_TOKEN=... go run ./examples/smoke
```

В PowerShell сначала задайте `$env:MARKETAUX_API_TOKEN` и запускайте команды
без Unix-префикса присваивания переменной. Для отчёта о покрытии:

```powershell
go test -race '-coverprofile=coverage.out' '-covermode=atomic' .
go tool cover '-func=coverage.out'
```

Покрытие измеряется для пакета библиотеки; примеры собираются и проверяются
отдельно. Живые тесты расходуют API-запросы и пропускаются без токена.
Подробности — в [CONTRIBUTING.md](CONTRIBUTING.md).

## Совместимость

Матрица CI охватывает Go 1.23–1.27. В `go.mod` нет `require`: только
стандартная библиотека. Ветка v1 сохраняет совместимость публичного API.

## Безопасность

- Токен передаётся в query-параметре `api_token`; API не использует
  `Authorization` для этой аутентификации.
- URL ошибок и debug-логи маскируют токен. Известные представления токена
  также удаляются из возвращённых сервером сообщений, диагностических
  фрагментов тела и заголовков ошибки.
- Для `errors.Is`/`errors.As` сохраняется исходная цепочка ошибок. Будьте
  внимательны при журналировании развёрнутых ошибок собственного транспорта
  или произвольных данных ответа.
- Собственный транспорт, HTTP tracing и access-логи прокси видят реальный
  URL. Настройте их так, чтобы query-токен не записывался.
- Ноль внешних зависимостей не отменяет обновлений стандартной библиотеки.
  Используйте исправленный Go и регулярно запускайте `govulncheck ./...`.
- Не коммитьте токены; `.env` добавлен в `.gitignore`.
- Об уязвимостях сообщайте приватно: [SECURITY.md](SECURITY.md).

## Частые проблемы

| Симптом | Что проверить |
| --- | --- |
| `ErrInvalidConfig` | Токен задан, таймаут положительный; при нулевом таймауте своего клиента `WithTimeout` стоит после `WithHTTPClient` |
| `ErrUnauthorized` | Правильный аккаунт и токен, отсутствие лишних пробелов |
| `ErrForbidden` на статистике | Доступ к эндпоинту в вашем тарифе |
| `ErrUsageLimit` | Дневная квота; ретраи её не восстановят |
| `ErrRateLimit` | Минутный лимит; уменьшите RPS/burst и проверьте `RetryAfter` |
| Статей меньше, чем `Limit` | Тариф может уменьшить страницу; смотрите `Meta.Limit` |
| Пустой результат | Упростите фильтры, проверьте UTC-даты и точный символ сущности |
| Нет debug-логов | Нужны `WithLogger`, debug-уровень его обработчика и `WithDebug(true)` |
| `ErrDecode` | Посмотрите фрагмент ответа с маской: прокси мог вернуть HTML или другой JSON |
| `ErrResponseTooLarge` | Уменьшите выдачу или осознанно увеличьте `WithMaxResponseBytes` |

## Примеры

Полные программы — в [examples/README.md](examples/README.md):
[новости](examples/news/main.go), [сущности](examples/entities/main.go),
[статистика](examples/stats/main.go), [пагинация](examples/pagination/main.go),
[ошибки](examples/errors/main.go) и [smoke](examples/smoke/main.go).
Задайте токен и запустите, например, `go run ./examples/news` из корня
репозитория. Для примеров статистики нужен доступ к stats-эндпоинтам.

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

См. [CONTRIBUTING.md](CONTRIBUTING.md). Ошибки и предложения можно описывать
в GitHub issues; никогда не прикладывайте API-токен.

## Лицензия

MIT © [Igor Sazonov](https://github.com/tigusigalpa) — см. [LICENSE](LICENSE).
