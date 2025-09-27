CREATE TABLE "warehouse" (
  "id" bigserial PRIMARY KEY, 
  "name" varchar NOT NULL,
  "address" varchar NOT NULL,
  "ward" varchar NOT NULL,
  "district" varchar NOT NULL,
  "city" varchar NOT NULL,
  "country" varchar NOT NULL
);

CREATE TABLE "storage_room" (
  "id" int PRIMARY KEY,
  "name" varchar NOT NULL,
  "number" varchar NOT NULL,
  "warehouse_id" int NOT NULL
);

ALTER TABLE "storage_room" ADD FOREIGN KEY ("warehouse_id") REFERENCES "warehouse" ("id");
