package main

import (
	"context"
	"errors"
	"event-drive/internal/shutdown"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Root context — владелец lifecycle всего сервиса.
	// Отменяется ТОЛЬКО в одном месте.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для системных сигналов.
	// Buffer = 1, чтобы не потерять сигнал, если никто не слушает мгновенно.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// HTTP handler.
	// Пока без роутеров — важно показать базу.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Используем контекст запроса — на собесе любят спросить
		// что будет, если клиент отвалится.
		select {
		case <-r.Context().Done():
			return
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		}
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Канал для ошибок сервера.
	// Не закрываем его вручную — пишет только владелец (goroutine ниже).
	serverErrCh := make(chan error, 1)

	go func() {
		log.Println("HTTP server started on :8080")

		// ListenAndServe блокирует горутину.
		// Shutdown() заставит его вернуть http.ErrServerClosed.
		if err := server.ListenAndServe(); err != nil {
			serverErrCh <- err
		}
	}()

	// Основной select — точка управления жизненным циклом.
	select {
	case sig := <-sigCh:
		log.Printf("received signal: %s, shutting down", sig.String())

	case err := <-serverErrCh:
		// Если сервер упал сам — это тоже повод завершаться.
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
	}

	// Отменяем root context.
	// ВСЕ зависимые операции должны начать завершение.
	cancel()

	// Контекст для graceful shutdown.
	// ВАЖНО: отдельный, с таймаутом.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Корректное завершение HTTP сервера.
	if err := shutdown.HTTPServer(shutdownCtx, server); err != nil {
		log.Printf("error during server shutdown: %v", err)
	} else {
		log.Println("server shutdown gracefully")
	}
}
