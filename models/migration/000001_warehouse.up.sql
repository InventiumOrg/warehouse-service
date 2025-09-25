CREATE TABLE "warehouse" (
  "id" int PRIMARY KEY, 
  "name" varchar NOT NULL,
  "address" varchar NOT NULL,
  "ward" int NOT NULL,
  "district" varchar NOT NULL,
  "city" varchar NOT NULL,
  "country" varchar NOT NULL
);

CREATE TABLE "storage_room" (
  "id" int PRIMARY KEY,
  "name" char,
  "number" int,
  "warehouse" varchar
);

ALTER TABLE "storage_room" ADD FOREIGN KEY ("warehouse") REFERENCES "warehouse" ("id");
