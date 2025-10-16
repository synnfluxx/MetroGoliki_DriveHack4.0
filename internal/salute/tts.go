package salute

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var (
	bearerToken    string
	voiceName      string
	authKey        string // Authorization Key для получения токенов
	tokenExpiresAt int64  // Время истечения токена (unix timestamp в миллисекундах)
)

// InitTTS инициализирует клиент Salute TTS
func InitTTS() error {
	apiKey := os.Getenv("SALUTE_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("SALUTE_API_KEY не установлен в переменных окружения")
	}

	// Определяем тип ключа по длине: короткий = Authorization Key, длинный = Access Token
	if len(apiKey) < 500 {
		authKey = apiKey
		log.Println("Обнаружен Authorization Key - токены будут обновляться автоматически")
		if err := refreshToken(); err != nil {
			return fmt.Errorf("не удалось получить первый токен: %w", err)
		}
	} else {
		bearerToken = apiKey
		log.Println("Используется готовый Access Token (без автообновления)")
	}

	voiceName = os.Getenv("SALUTE_VOICE")
	if voiceName == "" {
		voiceName = "Nec_24000"
	}
	
	log.Printf("TTS настроен: голос=%s, формат=wav16", voiceName)
	log.Println("Клиент Salute TTS успешно инициализирован (прямые HTTP запросы)")
	return nil
}

// TextToSpeech конвертирует текст в аудио и возвращает байты WAV файла
func TextToSpeech(text string) ([]byte, error) {
	if bearerToken == "" {
		return nil, fmt.Errorf("TTS клиент не инициализирован")
	}

	// Проверяем и обновляем токен если нужно
	if err := ensureValidToken(); err != nil {
		return nil, fmt.Errorf("не удалось обновить токен: %w", err)
	}

	log.Printf("[TTS] Синтез: %s", text)
	apiURL := fmt.Sprintf("https://smartspeech.sber.ru/rest/v1/text:synthesize?voice=%s&format=wav16", voiceName)
	
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(text))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/text")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	client := createHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[TTS] Ошибка HTTP запроса: %v", err)
		return nil, fmt.Errorf("ошибка HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[TTS] Тело ответа при ошибке: %s", string(bodyBytes))
		
		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("ошибка авторизации (401): проверьте SALUTE_API_KEY")
		}
		if resp.StatusCode == 403 {
			return nil, fmt.Errorf("доступ запрещен (403): нет прав на использование API")
		}
		
		return nil, fmt.Errorf("ошибка API: HTTP %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// Чтение аудио данных
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[TTS] Ошибка при чтении данных: %v", err)
		return nil, fmt.Errorf("ошибка чтения аудио данных: %w", err)
	}

	log.Printf("[TTS] Успешно: %d байт", len(audioData))

	if len(audioData) == 0 {
		return nil, fmt.Errorf("получены пустые аудио данные")
	}

	return audioData, nil
}

// SpeechToText распознает речь из аудио данных
func SpeechToText(audioData []byte) (string, error) {
	if bearerToken == "" {
		return "", fmt.Errorf("STT клиент не инициализирован")
	}

	// Проверяем и обновляем токен если нужно
	if err := ensureValidToken(); err != nil {
		return "", fmt.Errorf("не удалось обновить токен: %w", err)
	}

	log.Printf("[STT] Распознавание: %d байт", len(audioData))
	apiURL := "https://smartspeech.sber.ru/rest/v1/speech:recognize"
	
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(audioData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "audio/x-pcm;bit=16;rate=16000")
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	client := createHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[STT] Ошибка HTTP запроса: %v", err)
		return "", fmt.Errorf("ошибка HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Проверка статуса ответа
	if resp.StatusCode != http.StatusOK {
		log.Printf("[STT] Тело ответа при ошибке: %s", string(bodyBytes))
		
		if resp.StatusCode == 401 {
			return "", fmt.Errorf("ошибка авторизации (401): проверьте SALUTE_API_KEY")
		}
		if resp.StatusCode == 403 {
			return "", fmt.Errorf("доступ запрещен (403): нет прав на использование API")
		}
		
		return "", fmt.Errorf("ошибка API: HTTP %d - %s", resp.StatusCode, string(bodyBytes))
	}

	// Парсинг JSON ответа
	type STTResponse struct {
		Result []string `json:"result"`
	}

	var sttResp STTResponse
	if err := json.Unmarshal(bodyBytes, &sttResp); err != nil {
		log.Printf("[STT] Ошибка парсинга JSON: %v, тело: %s", err, string(bodyBytes))
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if len(sttResp.Result) == 0 {
		log.Printf("[STT] Пустой результат распознавания")
		return "", fmt.Errorf("речь не распознана")
	}

	recognizedText := sttResp.Result[0]
	log.Printf("[STT] Распознанный текст: %s", recognizedText)

	return recognizedText, nil
}

// IsInitialized проверяет, инициализирован ли TTS клиент
func IsInitialized() bool {
	return bearerToken != ""
}

// createHTTPClient создает HTTP клиент с учетом SSL настроек
func createHTTPClient() *http.Client {
	sslVerify := true
	if sslVerifyStr := os.Getenv("SSL_VERIFY"); sslVerifyStr != "" {
		var err error
		sslVerify, err = strconv.ParseBool(sslVerifyStr)
		if err != nil {
			log.Printf("Предупреждение: некорректное значение SSL_VERIFY, используется true")
			sslVerify = true
		}
	}

	if !sslVerify {
		log.Println("[TTS] ВНИМАНИЕ: Проверка SSL сертификатов отключена для TTS API")
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}

	return &http.Client{}
}

// ensureValidToken проверяет срок действия токена и обновляет его при необходимости
func ensureValidToken() error {
	if authKey == "" {
		return nil
	}

	now := time.Now().UnixMilli()
	if tokenExpiresAt > 0 && now < (tokenExpiresAt-60000) {
		return nil
	}

	log.Println("[TTS] Токен истек или скоро истечет, обновляем...")
	return refreshToken()
}

// refreshToken получает новый access token через OAuth
func refreshToken() error {
	type TokenResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
	}

	data := url.Values{}
	data.Set("scope", "SALUTE_SPEECH_PERS")

	req, err := http.NewRequest("POST", "https://ngw.devices.sberbank.ru:9443/api/v2/oauth", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+authKey)
	req.Header.Set("RqUID", fmt.Sprintf("%d", time.Now().UnixNano()))

	client := createHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка HTTP запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка получения токена: HTTP %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	bearerToken = tokenResp.AccessToken
	tokenExpiresAt = tokenResp.ExpiresAt

	expiresIn := time.Until(time.UnixMilli(tokenResp.ExpiresAt))
	log.Printf("[TTS] Новый токен получен, действителен до %s (еще %v)", 
		time.UnixMilli(tokenResp.ExpiresAt).Format("15:04:05"), 
		expiresIn.Round(time.Second))

	return nil
}
