package main

import (
  "context"
  "github.com/clerk/clerk-sdk-go/v2"
  "github.com/jackc/pgx/v5"
  "log/slog"
  "os"
  "time"
  "warehouse-service/api"
  "warehouse-service/config"
)

var conn *pgx.Conn

const attemptThreshold = 5

func main() {
  config, err := config.LoadConfig(".")
  if err != nil {
    slog.Error("Failed to load config: ", slog.Any("ERROR", err))
    os.Exit(1)
  }
  clerk.SetKey(config.ClerKKey)
  slog.Info("Connecting to database", slog.String("db_source", config.DBSource))
  attempt := 1
  for attempt <= attemptThreshold {
    conn, err = pgx.Connect(context.Background(), config.DBSource)
    if err == nil {
      slog.Info("Connected to database successfully")
      // defer conn.Close(context.Background())
      break
    }
    slog.Error("Failed to connect to database",
      slog.Int("attempt", attempt),
      slog.Int("maxAttempts", attemptThreshold),
      slog.Any("error", err),
    )

    if attempt == attemptThreshold {
      slog.Error("Max connection attempts reached, exiting", slog.Any("ERROR", err))
      os.Exit(1)
    }

    backoffDuration := time.Duration(1<<(attempt-1)) * time.Second
    slog.Info("Retrying connection",
      slog.Int("attempt", attempt+1),
      slog.Duration("backoff", backoffDuration),
    )

    time.Sleep(backoffDuration)
    attempt++

  }
  router := api.NewServer(conn)
  // q := models.New(conn)
  router.Run(":7450")

}
