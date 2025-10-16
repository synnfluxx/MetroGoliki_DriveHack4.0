package search

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Document представляет документ для поиска
type Document struct {
	ID    int    `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// SearchResult результат поиска
type SearchResult struct {
	Document Document
	Score    float64
}

// TFIDF простая реализация TF-IDF алгоритма
type TFIDF struct {
	Documents    []Document
	DocLengths   []int
	DocFreqs     []map[string]int
	IDF          map[string]float64
	NumDocs      int
}

// NewTFIDF создает новый TF-IDF индекс
func NewTFIDF() *TFIDF {
	return &TFIDF{
		DocFreqs: []map[string]int{},
		IDF:      make(map[string]float64),
	}
}

// tokenize разбивает текст на токены
func tokenize(text string) []string {
	// Приводим к нижнему регистру
	text = strings.ToLower(text)
	
	// Оставляем только буквы и цифры
	re := regexp.MustCompile(`\w+`)
	tokens := re.FindAllString(text, -1)
	
	return tokens
}

// BuildIndex строит индекс для поиска
func (tf *TFIDF) BuildIndex(documents []Document) {
	tf.Documents = documents
	tf.NumDocs = len(documents)
	
	// Токенизируем все документы
	tokenizedDocs := make([][]string, tf.NumDocs)
	tf.DocLengths = make([]int, tf.NumDocs)
	
	for i, doc := range documents {
		// Объединяем заголовок и текст (заголовок важнее - дублируем)
		combinedText := doc.Title + " " + doc.Title + " " + doc.Text
		tokens := tokenize(combinedText)
		tokenizedDocs[i] = tokens
		tf.DocLengths[i] = len(tokens)
	}
	
	// Подсчитываем частоту термов в каждом документе
	tf.DocFreqs = make([]map[string]int, tf.NumDocs)
	for i, tokens := range tokenizedDocs {
		freq := make(map[string]int)
		for _, token := range tokens {
			freq[token]++
		}
		tf.DocFreqs[i] = freq
	}
	
	// Подсчитываем document frequency для каждого терма
	df := make(map[string]int)
	for _, docFreq := range tf.DocFreqs {
		for term := range docFreq {
			df[term]++
		}
	}
	
	// Вычисляем IDF (классическая формула)
	for term, freq := range df {
		tf.IDF[term] = math.Log(float64(tf.NumDocs) / float64(freq))
	}
	
	log.Printf("Индекс построен: %d документов, %d уникальных термов", tf.NumDocs, len(tf.IDF))
}

// score вычисляет TF-IDF score для документа
func (tfidf *TFIDF) score(queryTokens []string, docIdx int) float64 {
	score := 0.0
	docLen := float64(tfidf.DocLengths[docIdx])
	docFreq := tfidf.DocFreqs[docIdx]
	
	for _, term := range queryTokens {
		freq := float64(docFreq[term])
		if freq == 0 {
			continue
		}
		
		// TF (term frequency) с нормализацией по длине документа
		termFreq := freq / docLen
		
		// IDF (inverse document frequency)
		idf := tfidf.IDF[term]
		
		// TF-IDF
		score += termFreq * idf
	}
	
	return score
}

// Search ищет наиболее релевантные документы
func (tf *TFIDF) Search(query string, topK int) []SearchResult {
	queryTokens := tokenize(query)
	
	// Вычисляем score для всех документов и сразу сохраняем результаты
	var results []SearchResult
	for i := 0; i < tf.NumDocs; i++ {
		score := tf.score(queryTokens, i)
		if score > 0 {
			results = append(results, SearchResult{
				Document: tf.Documents[i],
				Score:    score,
			})
		}
	}
	
	// Сортируем по убыванию score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	
	// Возвращаем только topK результатов
	if len(results) > topK {
		return results[:topK]
	}
	return results
}

// KnowledgeBase база знаний с поиском
type KnowledgeBase struct {
	SearchEngine *TFIDF
	Chunks       []Document
}

// NewKnowledgeBase создает новую базу знаний
func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		SearchEngine: NewTFIDF(),
		Chunks:       []Document{},
	}
}

// LoadChunks загружает чанки из JSON файла
func (kb *KnowledgeBase) LoadChunks(filename string) error {
	log.Printf("Загрузка чанков из %s...", filename)
	
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}
	
	err = json.Unmarshal(data, &kb.Chunks)
	if err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}
	
	log.Printf("Загружено %d чанков", len(kb.Chunks))
	
	// Строим индекс
	kb.SearchEngine.BuildIndex(kb.Chunks)
	
	return nil
}

// Search ищет релевантные чанки
func (kb *KnowledgeBase) Search(query string, topK int) []SearchResult {
	return kb.SearchEngine.Search(query, topK)
}

// GetContextForQuery получает контекст для добавления в промпт
func (kb *KnowledgeBase) GetContextForQuery(query string, maxChunks int) string {
	results := kb.Search(query, maxChunks)
	
	if len(results) == 0 {
		return ""
	}
	
	// Собираем контекст
	context := "Релевантная информация из базы знаний:\n\n"
	
	for i, result := range results {
		context += fmt.Sprintf("--- Источник %d: %s ---\n", i+1, result.Document.Title)
		context += result.Document.Text
		context += fmt.Sprintf("\n(URL: %s)\n\n", result.Document.URL)
	}
	
	return context
}
