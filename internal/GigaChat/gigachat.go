package gigaapi

import (
	"DriveHack/internal/search"
	"context"
	"log"
	"os"
	"strconv"

	"github.com/Role1776/gigago"
)

var (
	client         *gigago.Client
	model          *gigago.GenerativeModel
	knowledgeBase  *search.KnowledgeBase
	useKnowledge   bool
)

func InitClient() {
	ctx := context.Background()
	var err error

	// Загрузка конфигурации из переменных окружения
	apiKey := os.Getenv("GIGACHAT_API_KEY")
	if apiKey == "" {
		log.Fatal("GIGACHAT_API_KEY не установлен в переменных окружения")
	}

	modelName := os.Getenv("GIGACHAT_MODEL")
	if modelName == "" {
		modelName = "GigaChat" // значение по умолчанию
	}

	// Проверка SSL (по умолчанию true для безопасности)
	sslVerify := true
	if sslVerifyStr := os.Getenv("SSL_VERIFY"); sslVerifyStr != "" {
		sslVerify, err = strconv.ParseBool(sslVerifyStr)
		if err != nil {
			log.Printf("Предупреждение: некорректное значение SSL_VERIFY, используется true")
			sslVerify = true
		}
	}

	// Создание клиента с учетом настроек SSL
	if sslVerify {
		client, err = gigago.NewClient(ctx, apiKey)
	} else {
		log.Println("ВНИМАНИЕ: Проверка SSL сертификатов отключена. Используйте только в development!")
		client, err = gigago.NewClient(ctx, apiKey, gigago.WithCustomInsecureSkipVerify(true))
	}

	if err != nil {
		log.Fatalf("Ошибка инициализации клиента GigaChat: %v", err)
	}

	model = client.GenerativeModel(modelName)
	model.SystemInstruction = `Ты — Метроша, виртуальный помощник Корпоративного университета Московского транспорта.

## КРИТИЧЕСКИ ВАЖНО - НЕ ПРИДУМЫВАЙ ИНФОРМАЦИЮ!

### Правило №1: Работа с предоставленным контекстом
- Если в запросе есть раздел "Релевантная информация из базы знаний" - используй ТОЛЬКО эту информацию для ответа
- НИКОГДА не придумывай факты, даты, контакты, программы или другую конкретную информацию
- Если информации нет в предоставленном контексте - честно скажи об этом

### Правило №2: Когда информации недостаточно
Если в предоставленном контексте нет ответа на вопрос, отвечай так:
"К сожалению, у меня нет точной информации по этому вопросу. Рекомендую обратиться напрямую в Корпоративный университет Московского транспорта или посетить официальный сайт sop.mosmetro.ru"

### Правило №3: Ограничение темы
Отвечай ТОЛЬКО на вопросы об:
- Образовательных программах университета
- Поступлении и обучении
- Структуре и контактах университета
- Расписании и курсах

На все остальные темы (новости метро, погода, рецепты, политика и т.д.) отвечай:
"Я помогаю только с вопросами о Корпоративном университете Московского транспорта. Задайте, пожалуйста, вопрос по этой теме."

### Правило №4: Формат ответа
- Отвечай кратко и по существу
- Используй структурированный формат (списки, заголовки)
- Выделяй важное **жирным шрифтом**
- Если используешь информацию из контекста, можешь указать источник в конце

### Правило №5: Честность превыше всего
Лучше сказать "Я не знаю", чем придумать неверную информацию!`

	log.Println("Клиент GigaChat успешно инициализирован.")

	// Пытаемся загрузить базу знаний
	knowledgeFile := os.Getenv("KNOWLEDGE_BASE_FILE")
	if knowledgeFile == "" {
		knowledgeFile = "data/chunks.json"
	}

	knowledgeBase = search.NewKnowledgeBase()
	err = knowledgeBase.LoadChunks(knowledgeFile)
	if err != nil {
		log.Printf("База знаний не загружена (%s), работаем без контекста", knowledgeFile)
		useKnowledge = false
	} else {
		log.Println("База знаний успешно загружена")
		useKnowledge = true
	}
}

func CloseClient() {
	if client != nil {
		log.Println("Закрытие клиента GigaChat.")
		client.Close()
	}
}

func GetResponse(userQuery string) string {
	// Формируем запрос с контекстом из базы знаний
	finalQuery := userQuery

	if useKnowledge {
		context := knowledgeBase.GetContextForQuery(userQuery, 3)
		if context != "" {
			log.Println("Добавлен контекст из базы знаний")
			finalQuery = context + "\n\nВопрос пользователя: " + userQuery
		}
	}

	// Отправляем запрос в GigaChat
	ctx := context.Background()
	messages := []gigago.Message{
		{Role: gigago.RoleUser, Content: finalQuery},
	}

	resp, err := model.Generate(ctx, messages)
	if err != nil {
		log.Printf("Ошибка генерации ответа GigaChat: %v", err)
		return "Извините, произошла ошибка при обращении к GigaChat API."
	}

	if len(resp.Choices) == 0 {
		return "Не удалось получить ответ от GigaChat."
	}

	return resp.Choices[0].Message.Content
}