postgres:
	podman run --network inventium --name postgres-1 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=secret -p 5432:5432 -d postgres:16-alpine
createdb:
	podman exec -it postgres-1 createdb --username=root --owner=root warehouse-service
dropdb:
	podman exec -it postgres-1 dropdb --username=root inventium
migrateup:
	migrate -path ./models/migration -database "postgresql://root:secret@localhost:5432/warehouse-service?sslmode=disable" -verbose up
migratedown:
	migrate -path ./models/migration -database "postgresql://root:secret@localhost:5432/warehouse-service?sslmode=disable" -verbose down
sqlc:
	sqlc generate --no-remote
loaddata:
	PGPASSWORD=secret psql -h localhost -U root -d inventium -f data/sql/inventium.sql
runcontainer:
	podman run --network inventium --name warehouse-service -p 7450:7450 -d -e DB_SOURCE="postgresql://root:secret@postgres-1:5432/warehouse-service?sslmode=disable" -e CLERK_KEY="sk_test_XhHg2KNAIqm9I65JwOgQbLajZj6UqeeLTnpjx1p4oa" warehouse-service:1.0.0
.PHONY: postgres createdb dropdb migrateup migratedown sqlc loaddata runcontainer