package main

import (
	"DriveHack/internal/scraper"
	"flag"
	"log"
	"time"
)

func main() {
	// Параметры командной строки
	baseURL := flag.String("url", "https://sop.mosmetro.ru/", "Базовый URL для парсинга")
	maxPages := flag.Int("max", 100, "Максимальное количество страниц")
	delay := flag.Int("delay", 1000, "Задержка между запросами (мс)")
	outputJSON := flag.String("output", "data/sop_data.json", "Файл для сохранения данных")
	chunksFile := flag.String("chunks", "data/chunks.json", "Файл для сохранения чанков")
	chunkSize := flag.Int("chunk-size", 1000, "Размер чанка в символах")
	
	flag.Parse()

	log.Println("=== Скрапер sop.mosmetro.ru ===")
	log.Println("ВАЖНО: Запускайте только с русского IP!")
	log.Printf("URL: %s", *baseURL)
	log.Printf("Максимум страниц: %d", *maxPages)
	log.Printf("Задержка: %d мс", *delay)

	// Создаем скрапер
	s := scraper.NewScraper(*baseURL, *maxPages, time.Duration(*delay)*time.Millisecond)

	// Запускаем обход
	log.Println("\nНачинаем обход сайта...")
	s.Crawl(*baseURL)

	// Сохраняем полные данные
	log.Println("\nСохранение данных...")
	if err := s.SaveToJSON(*outputJSON); err != nil {
		log.Fatalf("Ошибка сохранения данных: %v", err)
	}

	// Сохраняем чанки
	if err := s.SaveChunks(*chunksFile, *chunkSize); err != nil {
		log.Fatalf("Ошибка сохранения чанков: %v", err)
	}

	log.Println("\n✓ Готово!")
	log.Printf("Собрано страниц: %d", len(s.Pages))
}
