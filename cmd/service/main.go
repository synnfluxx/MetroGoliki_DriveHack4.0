package main

import (
	gigaapi "DriveHack/internal/GigaChat"
	"DriveHack/internal/salute"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// Структура для входящего запроса (должна соответствовать JSON-телу)
type ChatRequest struct {
	Req string `json:"req"`
}

// Структура для исходящего ответа (должна соответствовать JS-ожиданиям)
type ChatResponse struct {
	Response string `json:"response"`
}

// Структура для запроса TTS
type TTSRequest struct {
	Text string `json:"text"`
}

// Структура для ответа STT
type STTResponse struct {
	Text string `json:"text"`
}

// Обработчик для POST /api/chat
func handleChatRequest(c *gin.Context) {
	var reqData ChatRequest

	if err := c.ShouldBindJSON(&reqData); err != nil {
		log.Println("Ошибка привязки JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса (ожидался JSON)"})
		return
	}

	log.Println("Получен запрос:", reqData.Req)

	resp := gigaapi.GetResponse(reqData.Req)
	log.Println("Ответ GigaChat:", resp)

	c.JSON(http.StatusOK, ChatResponse{Response: resp})
}

// Обработчик для POST /api/tts
func handleTTSRequest(c *gin.Context) {
	var reqData TTSRequest

	if err := c.ShouldBindJSON(&reqData); err != nil {
		log.Println("Ошибка привязки JSON:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	if reqData.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Текст не может быть пустым"})
		return
	}

	if !salute.IsInitialized() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "TTS сервис недоступен"})
		return
	}

	log.Println("TTS запрос для текста длиной:", len(reqData.Text), "символов")

	// Генерация аудио
	audioData, err := salute.TextToSpeech(reqData.Text)
	if err != nil {
		log.Printf("Ошибка TTS: %v", err)
		log.Printf("Текст для озвучки: %s", reqData.Text)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка генерации аудио: %v", err)})
		return
	}

	log.Printf("TTS успешно сгенерировано, размер: %d байт", len(audioData))

	// Возвращаем аудио файл
	c.Header("Content-Type", "audio/wav")
	c.Header("Content-Disposition", "inline; filename=speech.wav")
	c.Data(http.StatusOK, "audio/wav", audioData)
}

// Обработчик для POST /api/stt
func handleSTTRequest(c *gin.Context) {
	if !salute.IsInitialized() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "STT сервис недоступен"})
		return
	}

	// Читаем аудио данные из тела запроса
	audioData, err := c.GetRawData()
	if err != nil {
		log.Printf("Ошибка чтения аудио данных: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Ошибка чтения аудио данных"})
		return
	}

	if len(audioData) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пустые аудио данные"})
		return
	}

	log.Printf("STT запрос, размер аудио: %d байт", len(audioData))

	// Распознавание речи
	recognizedText, err := salute.SpeechToText(audioData)
	if err != nil {
		log.Printf("Ошибка STT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Ошибка распознавания речи: %v", err)})
		return
	}

	log.Printf("STT успешно распознано: %s", recognizedText)

	// Возвращаем распознанный текст
	c.JSON(http.StatusOK, STTResponse{Text: recognizedText})
}

func main() {
	// Загрузка переменных окружения из .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Предупреждение: .env файл не найден, используются системные переменные окружения")
	}

	// Инициализация клиента GigaChat
	gigaapi.InitClient()
	defer gigaapi.CloseClient()

	// Инициализация Salute TTS (опционально)
	if err := salute.InitTTS(); err != nil {
		log.Printf("Предупреждение: TTS не инициализирован: %v", err)
		log.Println("Озвучка ответов будет недоступна")
	} else {
		log.Println("TTS сервис готов к работе")
	}

	// Настройка режима Gin (production/debug)
	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Настройка CORS
	config := cors.DefaultConfig()
	
	// Загрузка разрешенных origins из переменных окружения
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins != "" {
		// Разделяем список origins по запятой
		origins := strings.Split(allowedOrigins, ",")
		for i, origin := range origins {
			origins[i] = strings.TrimSpace(origin)
		}
		config.AllowOrigins = origins
		log.Printf("CORS настроен для origins: %v", origins)
	} else {
		// Fallback для разработки
		log.Println("ВНИМАНИЕ: ALLOWED_ORIGINS не установлен, используется AllowAllOrigins (небезопасно для production!)")
		config.AllowAllOrigins = true
	}
	
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}
	router.Use(cors.New(config))

	router.GET("/", func(c *gin.Context) {
		c.File("public/index.html")
	})

	router.POST("/api/chat", handleChatRequest)
	router.POST("/api/tts", handleTTSRequest)
	router.POST("/api/stt", handleSTTRequest)

	// Получение порта из переменных окружения
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080" // значение по умолчанию
	}
	
	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "localhost"
	}
	
	addr := fmt.Sprintf("%s:%s", host, port)
	log.Printf("Сервер запущен на http://%s", addr)
	
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}