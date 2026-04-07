# Service Architecture Examples

Репозиторий с независимыми примерами сервисов с разными архиекурами.
Каждый пример показывает отдельный набор практик проектирования и разработки распределенных backend-систем, написанных на GO.

## Что внутри

### Marketplace (openapi-first)

REST-сервис маркетплейса с упором на:

- OpenAPI-first подход
- code generation
- CRUD для продуктов
- PostgreSQL + миграции
- JWT-аутентификацию
- RBAC
- валидацию и контрактные ошибки
- бизнес-логику заказов и промокодов

### Flight Booking (rest-grpc-redis)

Система бронирования авиабилетов из двух сервисов:

- **Booking Service** - REST API
- **Flight Service** - gRPC API

Покрывает:

- межсервисное взаимодействие по gRPC
- отдельные базы данных
- транзакционное резервирование мест
- Redis cache-aside
- Redis Sentinel
- retry / circuit breaker
- service-to-service authentication

### Online Cinema Analytics (event-streaming-pipeline)

Pipeline обработки событий онлайн-кинотеатра для аналитики на базе Kafka, ClickHouse и PostgreSQL.

Покрывает:

* event streaming через Kafka
* Schema Registry и версионирование схем событий
* Avro контракты
* Kafka producer с HTTP API
* ClickHouse Kafka Engine
* сервис агрегации бизнес-метрик
* материализацию агрегатов в ClickHouse
* выгрузку рассчитанных метрик в PostgreSQL
* Grafana dashboard
* экспорт агрегатов в S3 (MinIO)

## Запуск

```bash
docker-compose up --build
```
