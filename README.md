# 🚇 DriveHack - Метроша

Интеллектуальный чат-бот для Корпоративного университета Московского транспорта с поддержкой голосового ввода и озвучки ответов.

## ✨ Возможности

- 💬 **Текстовый чат** с GigaChat AI
- 🎤 **Голосовой ввод** через Salute STT
- 🔊 **Озвучка ответов** через Salute TTS
- 🎨 **Современный UI** с темной/светлой темой
- 🔄 **Автообновление токенов** Salute API
- 📱 **Адаптивный дизайн**
- 🔍 **База знаний** с TF-IDF поиском по sop.mosmetro.ru

## 🚀 Быстрый старт

### 1. Установка и запуск

```bash
git clone <repository>
cd DriveHack
go mod download
```

### 2. Настройка

Создайте `.env`:
```env
GIGACHAT_API_KEY=ваш_ключ_gigachat
SALUTE_API_KEY=ваш_authorization_key_salute
SERVER_PORT=8080
SSL_VERIFY=false
KNOWLEDGE_BASE_FILE=data/chunks.json
```

### 3. Запуск

```bash
./RUN.sh
```

Откройте http://localhost:8080

## 🛠 Технологии

**Backend:**
- Go 1.21+
- Gin Web Framework
- GigaChat API
- Salute Speech API
- TF-IDF Search (custom implementation)

**Frontend:**
- Vanilla JavaScript
- MediaRecorder API
- Web Audio API
- Marked.js (Markdown)

## 📋 API Endpoints

| Endpoint | Метод | Описание |
|----------|-------|----------|
| `/` | GET | Веб-интерфейс |
| `/api/chat` | POST | Отправка сообщения |
| `/api/tts` | POST | Синтез речи |
| `/api/stt` | POST | Распознавание речи |

## ⚙️ Переменные окружения

| Переменная | Описание | По умолчанию |
|-----------|----------|--------------|
| `GIGACHAT_API_KEY` | API ключ GigaChat | *обязательно* |
| `SALUTE_API_KEY` | Authorization Key Salute | *обязательно* |
| `SERVER_PORT` | Порт сервера | `8080` |
| `SERVER_HOST` | Хост сервера | `localhost` |
| `SSL_VERIFY` | Проверка SSL | `true` |
| `SALUTE_VOICE` | Голос для TTS | `Nec_24000` |
| `ENVIRONMENT` | Окружение | `development` |

## 🎙️ Доступные голоса

- `Nec_24000` - Наталья (женский)
- `Bys_24000` - Борис (мужской)
- `May_24000` - Марфа (женский)
- `Tur_24000` - Тарас (мужской)
- `Ost_24000` - Александра (женский)

## 🔧 Разработка

```bash
# Сборка
go build -o bin/drivehack cmd/service/main.go

# Запуск
./bin/drivehack

# Тесты
go test ./...
```

## 🐛 Troubleshooting

### TTS не работает
- Проверьте `SALUTE_API_KEY` в `.env`
- Используйте Authorization Key, а не Access Token
- Проверьте логи сервера

### GigaChat не отвечает
- Проверьте `GIGACHAT_API_KEY`
- Для development: `SSL_VERIFY=false`

### Порт занят
```bash
lsof -i :8080
kill -9 <PID>
```

## 📄 Лицензия

MIT License

## 👥 Авторы

Разработано в рамках хакатона DriveHack4.0
