package middlewares

import (
  "github.com/clerk/clerk-sdk-go/v2/jwt"
  "log/slog"
  "net/http"
  "strings"

  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5"
)

func ClerkAuth(db *pgx.Conn) gin.HandlerFunc {
  return func(c *gin.Context) {
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
      c.JSON(http.StatusUnauthorized, gin.H{
        "error":   "Unauthorized",
        "message": "Authorization header required",
      })
      slog.Error("Unable to get authorization header")
      c.Abort()
      return
    }
    sessionToken := strings.TrimPrefix(authHeader, "Bearer")
    if sessionToken == authHeader || sessionToken == "" {
      c.JSON(http.StatusUnauthorized, gin.H{
        "error":   "Unauthorized",
        "message": "Bearer token required",
      })
      slog.Error("Unable to get authorization header")
      c.Abort()
      return
    }
    claims, err := jwt.Verify(c.Request.Context(), &jwt.VerifyParams{
      Token: sessionToken,
    })
    if err != nil {
      c.JSON(http.StatusUnauthorized, gin.H{
        "error":   "Unauthorized",
        "message": "Invalid or expired token",
        "detail":  err.Error(),
      })
      slog.Error("User token is invalid: ", slog.Any("ERROR", err.Error()))
      c.Abort()
      return
    }
    c.Set("claims", claims)
    c.Set("user_id", claims.Subject)
    c.Next()
  }
}
