package utils

import (
	"errors"
	"sync"
)

// StreamInterface - интерфейс для представления потоков (можно заменить на конкретный тип).
type StreamInterface interface {
	Send(interface{}) error
	Close() error
	IsClosed() bool // Новый метод для проверки, закрыт ли поток
}

type MapConnectionsChat struct {
	mu            sync.Mutex
	MapChat       map[string][]StreamInterface        // Карта для хранения списка потоков по chatId
	NewStreamFunc func(chatId string) StreamInterface // Функция для создания нового соединения
}

// NewMapConnectionsChat - конструктор
func NewMapConnectionsChat(newStreamFunc func(chatId string) StreamInterface) *MapConnectionsChat {
	return &MapConnectionsChat{
		MapChat:       make(map[string][]StreamInterface),
		NewStreamFunc: newStreamFunc,
	}
}

// AddChatStream - добавляет новый поток к списку потоков для chatId
func (m *MapConnectionsChat) AddChatStream(chatId string, stream StreamInterface) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.MapChat[chatId] = append(m.MapChat[chatId], stream)
}

// RemoveChatStream - удаляет поток из списка потоков для chatId
func (m *MapConnectionsChat) RemoveChatStream(chatId string, stream StreamInterface) {
	m.mu.Lock()
	defer m.mu.Unlock()

	streams, exists := m.MapChat[chatId]
	if !exists {
		return
	}

	// Удаляем поток из списка
	for i, s := range streams {
		if s == stream {
			streams = append(streams[:i], streams[i+1:]...)
			_ = s.Close() // Закрываем поток
			break
		}
	}

	// Обновляем карту
	if len(streams) > 0 {
		m.MapChat[chatId] = streams
	} else {
		delete(m.MapChat, chatId)
	}
}

// BroadcastMessage - отправляет сообщение всем потокам в заданном чате
func (m *MapConnectionsChat) BroadcastMessage(chatId string, message interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	streams, exists := m.MapChat[chatId]
	if !exists {
		return errors.New("нет активных потоков для этого чата")
	}

	for _, stream := range streams {
		if err := stream.Send(message); err != nil {
			return err
		}
	}
	return nil
}
