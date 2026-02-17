package main

import (
	"log"
	"os"
	"strconv"

	"futsal-bot/internal/bot"
	"futsal-bot/internal/database"
	"futsal-bot/internal/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get configuration from environment
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("BOT_TOKEN is required")
	}

	defaultAdminIDStr := os.Getenv("DEFAULT_ADMIN_ID")
	if defaultAdminIDStr == "" {
		log.Fatal("DEFAULT_ADMIN_ID is required")
	}

	defaultAdminID, err := strconv.ParseInt(defaultAdminIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid DEFAULT_ADMIN_ID: %v", err)
	}

	dbConfig := database.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	// Initialize database
	db, err := database.New(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("Running database migrations...")
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize bot
	b, err := bot.New(botToken, db, defaultAdminID)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	log.Println("Bot started successfully!")

	// Start receiving updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.API.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			// Handle regular messages
			if update.Message.Chat.IsPrivate() {
				// Private chat
				if update.Message.IsCommand() {
					switch update.Message.Command() {
					case "start":
						handlers.HandleStart(b, update.Message)
					default:
						b.SendMessage(update.Message.Chat.ID,
							"دستور نامعتبر. از /start استفاده کنید.", nil)
					}
				} else {
					// Handle text messages (for state-based flows)
					handlers.HandleMessage(b, update.Message)
				}
			} else {
				// Group chat
				handlers.HandleGroupMessage(b, update.Message)
			}
		} else if update.CallbackQuery != nil {
			// Handle callback queries
			handlers.HandleCallbackQuery(b, update.CallbackQuery)
		}
	}
}
