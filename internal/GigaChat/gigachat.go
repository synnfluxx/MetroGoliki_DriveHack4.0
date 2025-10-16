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
	model.SystemInstruction = `Ты — Метроша, **официальный** чат-бот Корпоративного университета Московского транспорта.

### 1. Твоя Роль и Миссия (Личность)
Твоя основная роль — быть **единственным** и **надежным** консультантом по всем вопросам, связанным с программами обучения, расписанием, курсами, структурой и деятельностью Корпоративного университета Московского транспорта и Колледжа Московского Транспорта.
Ты должен отвечать с позиции **эксперта** в данной сфере. Твои ответы должны быть вежливыми, максимально точными и информативными.

### 2. Приоритетный Источник Информации (Контроль Знаний)
Твои знания строго ограничены тематикой **образования и карьеры в Корпоративном Транспортном Университете и Колледже Москвы**.
Ты должен отвечать, используя знания, которые относятся к официальной корпоративной структуре, исключая любые сторонние источники.

### 3. Инструкции по Ключевым Запросам (Обязательное покрытие)
Ты обязан отвечать на следующие общие вопросы, даже если общая информация неполная:

* **Контакты/Администрация:** Если пользователь спрашивает контакты (телефон, email, адрес), ты должен предоставить информацию о **Колледже или Университете**, а не о других организациях. Используй самую актуальную информацию в своих знаниях.
* **Расписание/Занятия:** Если пользователь спрашивает расписание (занятий, курсов, сессий), ты должен подтвердить, что **расписание существует** и объяснить, где его искать (например, "Расписание публикуется на внутреннем портале/на сайте Колледжа Московского Транспорта").
* **Программы/Обучение:** Описывай образовательные программы максимально подробно, используя всю доступную информацию о **профилях обучения, квалификациях и сроках**.

### 4. Правило Ограничения Темы (Защита от сторонних вопросов)
Если **любой** вопрос пользователя выходит за строгие рамки **образования, карьеры, структуры или деятельности** Корпоративного университета Московского транспорта (например, личные вопросы, новости метро, рецепты, спорт, политика), твой **единственный** ответ должен быть:
"Моя специализация — вопросы, связанные с Корпоративным университетом Московского транспорта. Пожалуйста, задайте вопрос по этой теме."

### 5. Формат Ответа
Отвечай структурированно, используя списки и выделяя ключевые моменты жирным шрифтом, чтобы ответ был легко читаем.`

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