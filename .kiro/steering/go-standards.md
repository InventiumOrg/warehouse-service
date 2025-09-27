# Go Development Standards

## Code Style
- Follow standard Go formatting with `go fmt`
- Use tabs for indentation (Go standard)
- Keep functions focused and small
- Use meaningful variable names

## Error Handling
- Always handle errors explicitly
- Use structured logging with slog
- Return appropriate HTTP status codes
- Include context in error messages

## Database Operations
- Use transactions for multi-step operations
- Always defer rollback after beginning transaction
- Use proper type conversions (int32 vs int64)
- Include proper error handling for database queries

## API Handlers
- Validate input parameters
- Check authentication/authorization
- Use consistent response formats
- Include proper HTTP status codes
- Add tracing spans for observability