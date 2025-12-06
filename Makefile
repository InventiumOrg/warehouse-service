postgres:
	podman run --network inventium --name postgres -e POSTGRES_USER="$(PG_USER)" -e POSTGRES_PASSWORD="$(PG_PASSWORD)" -p 5432:5432 -d postgres:16-alpine
createdb:
	podman exec -it postgres createdb --username="$(PG_USER)" --owner="$(PG_USER)" warehouse-service
dropdb:
	podman exec -it postgres dropdb --username="$(PG_USER)" inventium
migrateup:
	migrate -path ./models/migration -database "$(DB_SOURCE)" -verbose up
migratedown:
	migrate -path ./models/migration -database "$(DB_SOURCE)" -verbose down
sqlc:
	sqlc generate --no-remote
loaddata:
	PGPASSWORD=secret psql -h localhost -U root -d inventium -f data/sql/inventium.sql
runcontainer:
	podman run --network inventium --name warehouse-service -p 7450:7450 -d -e DB_SOURCE="$(DB_SOURCE)" -e CLERK_KEY="$(CLERK_KEY)" warehouse-service:1.0.0
.PHONY: postgres createdb dropdb migrateup migratedown sqlc loaddata runcontainer
