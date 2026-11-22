# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Оптимизация памяти

Профили сняты под нагрузкой `hey` (`POST /updates/`, `POST /value/`, `GET /`): `profiles/base.pprof` — до оптимизации,
`profiles/result.pprof` — после.

Главная проблема: gzip-middleware создавал новый `gzip.Writer` (~800 КБ) на каждый запрос — это ~97% всей выделенной
памяти. Что изменено:

- `gzip.Writer`/`gzip.Reader` переиспользуются через `sync.Pool`, writer создаётся только если ответ сжимается;
- буферы в `HashMiddleware` берутся из пула;
- убраны лишние аллокации в разборе `Accept-Encoding` и `service.GetAll`.

```
go tool pprof -top -sample_index=alloc_space -diff_base=profiles/base.pprof profiles/result.pprof
```

```
Type: alloc_space
Showing nodes accounting for -929.84GB, 90.40% of 1028.61GB total
Dropped 204 nodes (cum <= 5.14GB)
Showing top 10 nodes out of 33
      flat  flat%   sum%        cum   cum%
 -796.52GB 77.44% 77.44%  -967.99GB 94.11%  compress/flate.NewWriter (inline)
 -166.77GB 16.21% 93.65%  -166.77GB 16.21%  compress/flate.(*compressor).initDeflate (inline)
   20.23GB  1.97% 91.68%    20.23GB  1.97%  reflect.growslice
   13.77GB  1.34% 90.34%    13.81GB  1.34%  encoding/json.(*Decoder).refill
   -2.46GB  0.24% 90.58%    -4.70GB  0.46%  compress/flate.newHuffmanBitWriter (inline)
    1.48GB  0.14% 90.44%    38.64GB  3.76%  main.newRouter.MetricsListHandler.func6
    0.25GB 0.024% 90.42%  -317.61GB 30.88%  github.com/MaxPa1/go-metrics/internal/middleware.RequestLogger.func1.1
    0.11GB  0.01% 90.40%     -247GB 24.01%  github.com/MaxPa1/go-metrics/internal/handler.GetMetricsV2Handler.func1
    0.04GB 0.0041% 90.40%  -927.62GB 90.18%  net/http.(*conn).serve
    0.04GB 0.0036% 90.40%  -110.29GB 10.72%  text/template.(*Template).execute
```
