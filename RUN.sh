#!/bin/bash

# Скрипт для быстрого запуска бота с базой знаний

echo "=== DriveHack - Метроша ==="
echo ""

# Проверяем наличие базы знаний
if [ ! -f "data/chunks.json" ]; then
    echo "⚠️  База знаний не найдена!"
    echo ""
    echo "Хотите собрать данные с sop.mosmetro.ru? (y/n)"
    echo "ВАЖНО: Запускайте только с русского IP!"
    read -r answer
    
    if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
        echo ""
        echo "Создаём директорию data..."
        mkdir -p data
        
        echo "Запускаем скрапер..."
        echo "(Это займёт 2-3 минуты)"
        echo ""
        
        go run cmd/scraper/main.go
        
        if [ $? -eq 0 ]; then
            echo ""
            echo "✓ Данные успешно собраны!"
        else
            echo ""
            echo "✗ Ошибка при сборе данных"
            exit 1
        fi
    else
        echo ""
        echo "Запускаем без базы знаний..."
        echo "(Бот будет работать только на знаниях GigaChat)"
    fi
else
    echo "✓ База знаний найдена: data/chunks.json"
fi

echo ""
echo "Запускаем сервер..."
echo ""

go run cmd/service/main.go
