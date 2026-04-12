package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// NotifyUser отправляет личное сообщение пользователю через Telegram Bot API.
// Требует, чтобы пользователь ранее начал диалог с ботом (нажал /start).
// Вызов всегда неблокирующий — запускайте в горутине.
func NotifyUser(telegramID uint, text string) {
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" || telegramID == 0 {
		return
	}
	sendTgMsg(botToken, int64(telegramID), text)
}

// sendTgMsg — внутренний хелпер, отправляет HTML-сообщение в любой чат.
func sendTgMsg(botToken string, chatID int64, text string) {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	resp, err := http.Post(
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		log.Printf("WARN: TG sendMsg to %d: %v", chatID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("WARN: TG sendMsg to %d: HTTP %d", chatID, resp.StatusCode)
	}
}
