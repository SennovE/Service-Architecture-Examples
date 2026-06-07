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

### Warehouse Events Consumer (event-driven-cassandra)

Consumer-сервис обработки складских событий на базе Kafka, Avro Schema Registry и Cassandra.

Покрывает:

* event-driven обработку сообщений из Kafka
* Avro контракты и версионирование схем
* Schema Registry
* идемпотентную обработку событий
* сохранение read-моделей в Cassandra
* DLQ для некорректных событий
* Prometheus metrics
* Grafana dashboard

## Запуск

```bash
docker-compose up --build
```
