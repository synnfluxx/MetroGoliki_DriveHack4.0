package scraper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// PageData содержит данные одной страницы
type PageData struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Length int    `json:"length"`
}

// Chunk представляет фрагмент текста
type Chunk struct {
	ID       int    `json:"id"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Text     string `json:"text"`
	StartPos int    `json:"start_pos"`
	EndPos   int    `json:"end_pos"`
}

// Scraper обходит сайт и собирает данные
type Scraper struct {
	BaseURL     string
	VisitedURLs map[string]bool
	Pages       []PageData
	Collector   *colly.Collector
	MaxPages    int
}

// NewScraper создает новый скрапер
func NewScraper(baseURL string, maxPages int, delay time.Duration) *Scraper {
	c := colly.NewCollector(
		colly.AllowedDomains(extractDomain(baseURL)),
		colly.MaxDepth(10),
		colly.Async(false),
	)

	// Настройки лимитов
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Delay:       delay,
		RandomDelay: delay / 2,
	})

	// Таймаут
	c.SetRequestTimeout(30 * time.Second)

	// User-Agent
	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	return &Scraper{
		BaseURL:     baseURL,
		VisitedURLs: make(map[string]bool),
		Pages:       []PageData{},
		Collector:   c,
		MaxPages:    maxPages,
	}
}

// extractDomain извлекает домен из URL
func extractDomain(urlStr string) string {
	if strings.HasPrefix(urlStr, "http://") {
		urlStr = strings.TrimPrefix(urlStr, "http://")
	} else if strings.HasPrefix(urlStr, "https://") {
		urlStr = strings.TrimPrefix(urlStr, "https://")
	}
	
	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}
	
	return urlStr
}

// shouldSkipURL проверяет, нужно ли пропустить URL
func shouldSkipURL(url string) bool {
	skipExtensions := []string{".pdf", ".jpg", ".jpeg", ".png", ".gif", ".zip", ".doc", ".docx", ".xls", ".xlsx", ".mp4", ".avi"}
	lowerURL := strings.ToLower(url)
	for _, ext := range skipExtensions {
		if strings.HasSuffix(lowerURL, ext) {
			return true
		}
	}
	return false
}

// cleanText очищает текст от лишних пробелов
func cleanText(text string) string {
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(cleaned, " ")
}

// Crawl рекурсивно обходит сайт используя Colly
func (s *Scraper) Crawl(startURL string) {
	// Обработчик HTML элементов
	s.Collector.OnHTML("html", func(e *colly.HTMLElement) {
		// Проверяем лимит страниц
		if len(s.Pages) >= s.MaxPages {
			return
		}

		// Получаем заголовок
		title := e.ChildText("title")
		if title == "" {
			title = e.Request.URL.String()
		}

		// Извлекаем текст, исключая скрипты, стили и навигацию
		e.ForEach("script, style, nav, footer, header", func(_ int, el *colly.HTMLElement) {
			el.DOM.Remove()
		})

		text := cleanText(e.DOM.Text())

		// Сохраняем данные страницы
		if text != "" {
			pageData := PageData{
				URL:    e.Request.URL.String(),
				Title:  title,
				Text:   text,
				Length: len(text),
			}
			s.Pages = append(s.Pages, pageData)
			log.Printf("Обработано: %d/%d - %s", len(s.Pages), s.MaxPages, e.Request.URL.String())
		}
	})

	// Обработчик ссылок
	s.Collector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		// Проверяем лимит страниц
		if len(s.Pages) >= s.MaxPages {
			return
		}

		link := e.Attr("href")
		
		// Пропускаем файлы
		if shouldSkipURL(link) {
			return
		}

		// Переходим по ссылке
		e.Request.Visit(link)
	})

	// Обработчик перед запросом
	s.Collector.OnRequest(func(r *colly.Request) {
		// Проверяем лимит страниц
		if len(s.Pages) >= s.MaxPages {
			r.Abort()
			return
		}

		// Проверяем, не посещали ли уже
		if s.VisitedURLs[r.URL.String()] {
			r.Abort()
			return
		}
		s.VisitedURLs[r.URL.String()] = true
	})

	// Обработчик ошибок
	s.Collector.OnError(func(r *colly.Response, err error) {
		log.Printf("Ошибка при обработке %s: %v", r.Request.URL, err)
	})

	// Запускаем обход
	log.Printf("Начинаем обход с %s", startURL)
	err := s.Collector.Visit(startURL)
	if err != nil {
		log.Printf("Ошибка запуска обхода: %v", err)
	}

	// Ждем завершения всех запросов
	s.Collector.Wait()
}

// SaveToJSON сохраняет данные в JSON файл
func (s *Scraper) SaveToJSON(filename string) error {
	data, err := json.MarshalIndent(s.Pages, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	log.Printf("Данные сохранены в %s (страниц: %d)", filename, len(s.Pages))
	return nil
}

// SaveChunks разбивает текст на чанки и сохраняет
func (s *Scraper) SaveChunks(filename string, chunkSize int) error {
	var chunks []Chunk
	chunkID := 0

	for _, page := range s.Pages {
		text := page.Text

		// Разбиваем на чанки
		for i := 0; i < len(text); i += chunkSize {
			end := i + chunkSize
			if end > len(text) {
				end = len(text)
			}

			chunk := Chunk{
				ID:       chunkID,
				URL:      page.URL,
				Title:    page.Title,
				Text:     text[i:end],
				StartPos: i,
				EndPos:   end,
			}

			chunks = append(chunks, chunk)
			chunkID++
		}
	}

	data, err := json.MarshalIndent(chunks, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	log.Printf("Создано %d чанков в %s", len(chunks), filename)
	return nil
}
