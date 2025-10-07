package events

import (
	"log/slog"
	"sync"
	"time"
)

// Event representa un evento del sistema
type Event struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// EventBus maneja las conexiones SSE
type EventBus struct {
	clients map[chan Event]bool
	mutex   sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		clients: make(map[chan Event]bool),
	}
}

// Subscribe agrega un nuevo cliente
func (eb *EventBus) Subscribe() chan Event {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	client := make(chan Event, 10) // Buffer para evitar bloqueos
	eb.clients[client] = true

	slog.Info("Nuevo cliente suscrito a eventos", "total_clients", len(eb.clients))
	return client
}

// Unsubscribe elimina un cliente
func (eb *EventBus) Unsubscribe(client chan Event) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	if _, exists := eb.clients[client]; exists {
		close(client)
		delete(eb.clients, client)
		slog.Info("Cliente desuscrito de eventos", "total_clients", len(eb.clients))
	}
}

// Broadcast envía un evento a todos los clientes
func (eb *EventBus) Broadcast(event Event) {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()

	event.Timestamp = time.Now()

	if len(eb.clients) == 0 {
		return
	}

	slog.Info("Enviando evento", "type", event.Type, "clients", len(eb.clients))

	for client := range eb.clients {
		select {
		case client <- event:
			// Evento enviado exitosamente
		default:
			// Cliente lento, eliminar para evitar bloqueos
			// slog.Warn("Cliente lento, eliminando suscripción")
			// go eb.Unsubscribe(client)
			// slog.Warn("Cliente lento, omitiendo evento", "event_type", event.Type)
		}
	}
}

// GetClientCount retorna el número de clientes conectados
func (eb *EventBus) GetClientCount() int {
	eb.mutex.RLock()
	defer eb.mutex.RUnlock()
	return len(eb.clients)
}
