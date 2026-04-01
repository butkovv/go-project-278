package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"url-shortener/handlers"
)

type Config struct {
	Port string
}

func main() {
	config := Config{}
	flag.StringVar(&config.Port, "port", "8080", "Порт для работы сервера")
	flag.Parse()

	router := handlers.SetupRouter()

	addr := fmt.Sprintf(":%s", config.Port)
	slog.Info("Сервер запущен", "host", "http://localhost", "port", addr)

	err := router.Run(addr)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
