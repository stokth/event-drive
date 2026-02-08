package shutdown

import (
	"context"
	"net/http"
)

// HTTPServer инкапсулирует shutdown-логику.
// Это удобно для тестирования и повторного использования.
func HTTPServer(ctx context.Context, srv *http.Server) error {
	// Shutdown:
	// - перестаёт принимать новые соединения
	// - ждёт завершения активных запросов
	// - уважает ctx.Done()

	return srv.Shutdown(ctx)
}
