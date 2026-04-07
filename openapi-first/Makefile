run-auth:
	docker-compose up auth-api --build

migrate-auth:
	docker-compose up auth-db -d
	docker-compose run --rm auth-migrations

run-products:
	docker-compose up products-api --build

migrate-products:
	docker-compose up products-db -d
	docker-compose run --rm products-migrations
