package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
	models "warehouse-service/models/sqlc"
	"warehouse-service/observability"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Handlers struct {
	db                *pgx.Conn
	queries           *models.Queries
	tracer            trace.Tracer
	prometheusMetrics *observability.PrometheusMetrics
}

func NewHandlers(db *pgx.Conn, prometheusMetrics *observability.PrometheusMetrics) *Handlers {
	return &Handlers{
		db:                db,
		queries:           models.New(db),
		tracer:            otel.Tracer("warehouse-service/handlers"),
		prometheusMetrics: prometheusMetrics,
	}
}

func (h *Handlers) GetWarehouse(ctx *gin.Context) {
	// Start a new span for this operation
	_, span := h.tracer.Start(ctx.Request.Context(), "GetWarehouse")
	defer span.End()

	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid warehouse ID format",
		})
		return
	}
	span.SetAttributes(attribute.Int64("warehouse.id", id))

	dbStart := time.Now()
	warehouse, err := h.queries.GetWarehouse(ctx, id)
	dbDuration := time.Since(dbStart)

	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("get", "warehouse", dbDuration, err)
	}

	if err != nil {
		slog.Error("Got an error while getting warehouse: ", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get warehouse",
		})
		return
	}

	// Record successful retrieval (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordInventoryOperation("get", warehouse.Name, warehouse.Address)
	}

	// Record successful operation
	span.SetAttributes(
		attribute.String("warehouse.name", warehouse.Name),
		attribute.String("operation.status", "success"),
	)
	ctx.JSON(200, gin.H{
		"message": "Get Warehouse Successfully",
		"data":    warehouse,
	})
}

func (h *Handlers) ListWarehouse(ctx *gin.Context) {
	// Start a new span for this operation
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "ListWarehouse")
	defer span.End()

	// Add attributes to the span
	span.SetAttributes(
		attribute.Int("warehouse.limit", 10),
		attribute.Int("warehouse.offset", 0),
	)

	dbStart := time.Now()
	warehouses, err := h.queries.ListWarehouse(spanCtx, models.ListWarehouseParams{
		Limit:  10,
		Offset: 0,
	})
	dbDuration := time.Since(dbStart)
	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("list", "inventory", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", "database_query_failed"))
		slog.Error("Got an error while listing warehouses: ", slog.Any("err", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list warehouses",
		})
		return
	}

	// Record successful list operation (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordInventoryOperation("list", "all", "all")
	}

	// Record successful operation
	span.SetAttributes(
		attribute.Int("warehouse.count", len(warehouses)),
		attribute.String("operation.status", "success"),
	)

	ctx.JSON(200, gin.H{
		"message": "List Warehouse Successfully",
		"data":    warehouses,
	})
}

func (h *Handlers) UpdateWarehouse(ctx *gin.Context) {
	// Start a new span for this operation
	_, span := h.tracer.Start(ctx.Request.Context(), "UpdateWarehouse")
	defer span.End()

	// Get warehouse ID from URL parameter
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid warehouse ID",
		})
		return
	}
	span.SetAttributes(attribute.Int64("warehouse.id", id))

	// Start database transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		slog.Error("Failed to start transaction", slog.Any("err", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback(ctx) // This will be ignored if tx.Commit() succeeds

	// Create queries with transaction
	qtx := h.queries.WithTx(tx)

	// Check if warehouse exists before updating
	dbStart := time.Now()
	_, err = qtx.GetWarehouse(ctx, id)
	dbDuration := time.Since(dbStart)

	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("get", "warehouse", dbDuration, err)
	}

	if err != nil {
		slog.Error("Warehouse not found", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "Warehouse not found",
		})
		return
	}

	// Update warehouse within transaction
	param := models.UpdateWarehouseParams{
		ID:      id,
		Name:    ctx.PostForm("Name"),
		Address: ctx.PostForm("Address"),
		Ward:    ctx.PostForm("Ward"),
		City:    ctx.PostForm("City"),
		Country: ctx.PostForm("Country"),
	}

	dbStart = time.Now()
	warehouse, err := qtx.UpdateWarehouse(ctx, param)
	dbDuration = time.Since(dbStart)

	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("update", "warehouse", dbDuration, err)
	}

	if err != nil {
		slog.Error("Could not update warehouse", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update warehouse",
		})
		return
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit transaction", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to commit transaction",
		})
		return
	}

	// Record successful update (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordInventoryOperation("update", warehouse.Name, warehouse.Address)
	}

	// Record successful operation
	span.SetAttributes(
		attribute.String("warehouse.name", warehouse.Name),
		attribute.String("operation.status", "success"),
	)

	ctx.JSON(200, gin.H{
		"message": "Update Warehouse Successfully",
		"data":    warehouse,
	})
}

func (h *Handlers) CreateWarehouse(ctx *gin.Context) {
	// Start a new span for this operation
	_, span := h.tracer.Start(ctx.Request.Context(), "CreateWarehouse")
	defer span.End()

	param := models.CreateWarehouseParams{
		Name:    ctx.PostForm("Name"),
		Address: ctx.PostForm("Address"),
		Ward:    ctx.PostForm("Ward"),
		City:    ctx.PostForm("City"),
		Country: ctx.PostForm("Country"),
	}

	span.SetAttributes(
		attribute.String("warehouse.name", param.Name),
		attribute.String("warehouse.address", param.Address),
	)

	dbStart := time.Now()
	warehouse, err := h.queries.CreateWarehouse(ctx, param)
	dbDuration := time.Since(dbStart)

	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("create", "warehouse", dbDuration, err)
	}

	if err != nil {
		slog.Error("Could not create warehouse: ", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create warehouse",
		})
		return
	}

	// Record successful creation (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordInventoryOperation("create", warehouse.Name, warehouse.Address)
		// Update active warehouse count (increment by 1)
		h.prometheusMetrics.UpdateInventoryCount(1) // This should be the actual total count in production
	}

	// Record successful operation
	span.SetAttributes(
		attribute.Int64("warehouse.id", warehouse.ID),
		attribute.String("operation.status", "success"),
	)

	ctx.JSON(200, gin.H{
		"message": "Create Warehouse Successfully",
		"data":    warehouse,
	})
}

func (h *Handlers) DeleteWarehouse(ctx *gin.Context) {
	// Start a new span for this operation
	_, span := h.tracer.Start(ctx.Request.Context(), "DeleteWarehouse")
	defer span.End()

	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid warehouse ID format",
		})
		return
	}
	span.SetAttributes(attribute.Int64("warehouse.id", id))

	dbStart := time.Now()
	err = h.queries.DeleteWarehouse(ctx, id)
	dbDuration := time.Since(dbStart)

	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("delete", "warehouse", dbDuration, err)
	}

	if err != nil {
		slog.Error("Failed to delete warehouse: ", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete warehouse",
		})
		return
	}

	// Record successful deletion (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordInventoryOperation("delete", "warehouse", "unknown")
	}

	// Record successful operation
	span.SetAttributes(
		attribute.String("operation.status", "success"),
	)

	ctx.JSON(200, gin.H{"message": "Delete Warehouse Successfully"})
}
