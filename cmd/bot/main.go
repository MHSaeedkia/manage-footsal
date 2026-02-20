package main

import (
	"os"
	"strconv"

	"futsal-bot/internal/bot"
	"futsal-bot/internal/database"
	"futsal-bot/internal/handlers"
	"futsal-bot/pkg/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	// Logger config from env (LOG_LEVEL, LOG_FORMAT, LOG_OUTPUT)
	loggerConfig := &logger.Config{
		Level:  getEnv("LOG_LEVEL", "info"),
		Format: getEnv("LOG_FORMAT", "json"),
		Output: getEnv("LOG_OUTPUT", "stdout"),
	}
	zapLogger, err := logger.New(loggerConfig, logger.DefaultServiceName)
	if err != nil {
		_, _ = os.Stderr.WriteString("failed to init logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer func() { _ = zapLogger.Sync() }()
	zap.ReplaceGlobals(zapLogger)

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		zap.L().Fatal("BOT_TOKEN is required")
	}

	defaultAdminIDStr := os.Getenv("DEFAULT_ADMIN_ID")
	if defaultAdminIDStr == "" {
		zap.L().Fatal("DEFAULT_ADMIN_ID is required")
	}

	defaultAdminID, err := strconv.ParseInt(defaultAdminIDStr, 10, 64)
	if err != nil {
		zap.L().Fatal("Invalid DEFAULT_ADMIN_ID", zap.Error(err))
	}

	dbConfig := database.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	db, err := database.New(dbConfig)
	if err != nil {
		zap.L().Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	zap.L().Info("Running database migrations...")
	if err := db.RunMigrations(); err != nil {
		zap.L().Fatal("Failed to run migrations", zap.Error(err))
	}

	b, err := bot.New(botToken, db, defaultAdminID)
	if err != nil {
		zap.L().Fatal("Failed to create bot", zap.Error(err))
	}

	zap.L().Info("Bot started successfully")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.API.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			if update.Message.Chat.IsPrivate() {
				if update.Message.IsCommand() {
					switch update.Message.Command() {
					case "start":
						handlers.HandleStart(b, update.Message)
					default:
						b.SendMessage(update.Message.Chat.ID,
							"دستور نامعتبر. از /start استفاده کنید.", nil)
					}
				} else {
					handlers.HandleMessage(b, update.Message)
				}
			} else {
				handlers.HandleGroupMessage(b, update.Message)
			}
		} else if update.CallbackQuery != nil {
			handlers.HandleCallbackQuery(b, update.CallbackQuery)
		}
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
