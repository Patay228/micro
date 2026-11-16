package common

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	DefaultExchangeName = "circus.exchange"
	DefaultRabbitMQURL  = "amqp://guest:guest@localhost:5672/"
)

type EventHandler func(event BaseEvent)

// InMemoryBus хранит подписчиков в памяти для работы без запущенного RabbitMQ и для тестов.
type InMemoryBus struct {
	mu          sync.RWMutex
	subscribers map[string][]EventHandler
	history     []BaseEvent
}

var (
	globalBus     *InMemoryBus
	globalBusOnce sync.Once
)

func GetGlobalInMemoryBus() *InMemoryBus {
	globalBusOnce.Do(func() {
		globalBus = &InMemoryBus{
			subscribers: make(map[string][]EventHandler),
			history:     make([]BaseEvent, 0),
		}
	})
	return globalBus
}

func (b *InMemoryBus) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers = make(map[string][]EventHandler)
	b.history = make([]BaseEvent, 0)
}

func (b *InMemoryBus) Subscribe(topic string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[topic] = append(b.subscribers[topic], handler)
	log.Printf("[InMemoryBus] Subscribed to topic: %s", topic)
}

func (b *InMemoryBus) Publish(topic string, event BaseEvent) {
	b.mu.Lock()
	b.history = append(b.history, event)
	var matchedHandlers []EventHandler

	for subTopic, handlers := range b.subscribers {
		if subTopic == topic || (strings.HasSuffix(subTopic, ".*") && strings.HasPrefix(topic, strings.TrimSuffix(subTopic, ".*"))) {
			matchedHandlers = append(matchedHandlers, handlers...)
		}
	}
	b.mu.Unlock()

	log.Printf("[InMemoryBus] Published event %s (ID: %s) to topic %s (handlers: %d)",
		event.EventType, event.EventID, topic, len(matchedHandlers))

	for _, h := range matchedHandlers {
		handler := h
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[InMemoryBus] Panic in event handler for topic %s: %v", topic, r)
				}
			}()
			handler(event)
		}()
	}
}

// Broker представляет клиент для работы с событиями (RabbitMQ с автоматическим fallback на InMemoryBus).
type Broker struct {
	serviceName   string
	rabbitURL     string
	conn          *amqp.Connection
	ch            *amqp.Channel
	isRabbitReady bool
	inMemory      *InMemoryBus
	mu            sync.Mutex
}

func NewBroker(serviceName string) *Broker {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = DefaultRabbitMQURL
	}

	b := &Broker{
		serviceName: serviceName,
		rabbitURL:   rabbitURL,
		inMemory:    GetGlobalInMemoryBus(),
	}

	// Попытка подключиться к RabbitMQ с коротким таймаутом
	if os.Getenv("FORCE_IN_MEMORY") != "true" {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		conn, err := amqp.DialConfig(rabbitURL, amqp.Config{Dial: dialer.Dial})
		if err == nil {
			ch, err := conn.Channel()
			if err == nil {
				err = ch.ExchangeDeclare(
					DefaultExchangeName,
					"topic",
					true,  // durable
					false, // auto-deleted
					false, // internal
					false, // no-wait
					nil,
				)
				if err == nil {
					b.conn = conn
					b.ch = ch
					b.isRabbitReady = true
					log.Printf("[%s] Connected to RabbitMQ at %s", serviceName, rabbitURL)
					return b
				}
			}
		}
		log.Printf("[%s] RabbitMQ not reachable (%v). Using In-Memory Event Bus fallback.", serviceName, err)
	} else {
		log.Printf("[%s] FORCE_IN_MEMORY=true, using In-Memory Event Bus.", serviceName)
	}

	return b
}

func (b *Broker) Publish(topic string, event BaseEvent) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Если подключен RabbitMQ
	if b.isRabbitReady && b.ch != nil {
		body, err := json.Marshal(event)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			err = b.ch.PublishWithContext(
				ctx,
				DefaultExchangeName,
				topic,
				false,
				false,
				amqp.Publishing{
					ContentType:  "application/json",
					DeliveryMode: amqp.Persistent,
					MessageId:    event.EventID,
					Timestamp:    time.Now().UTC(),
					Body:         body,
				},
			)
			if err == nil {
				log.Printf("[%s] [RabbitMQ] Published event %s to topic %s", b.serviceName, event.EventType, topic)
				return nil
			}
			log.Printf("[%s] [RabbitMQ] Publish failed: %v. Falling back to In-Memory.", b.serviceName, err)
		}
	}

	// Fallback в In-Memory
	b.inMemory.Publish(topic, event)
	return nil
}

func (b *Broker) Subscribe(topic string, handler EventHandler) {
	// Всегда регистрируем и в in-memory
	b.inMemory.Subscribe(topic, handler)

	if b.isRabbitReady && b.ch != nil {
		go func() {
			queueName := b.serviceName + "." + topic
			q, err := b.ch.QueueDeclare(queueName, true, false, false, false, nil)
			if err != nil {
				log.Printf("[%s] Failed to declare queue %s: %v", b.serviceName, queueName, err)
				return
			}

			err = b.ch.QueueBind(q.Name, topic, DefaultExchangeName, false, nil)
			if err != nil {
				log.Printf("[%s] Failed to bind queue %s to %s: %v", b.serviceName, q.Name, topic, err)
				return
			}

			msgs, err := b.ch.Consume(q.Name, "", false, false, false, false, nil)
			if err != nil {
				log.Printf("[%s] Failed to consume queue %s: %v", b.serviceName, q.Name, err)
				return
			}

			log.Printf("[%s] [RabbitMQ] Listening to queue %s (topic: %s)", b.serviceName, q.Name, topic)
			for d := range msgs {
				var ev BaseEvent
				if err := json.Unmarshal(d.Body, &ev); err == nil {
					handler(ev)
					_ = d.Ack(false)
				} else {
					log.Printf("[%s] Error unmarshaling event: %v", b.serviceName, err)
					_ = d.Nack(false, false)
				}
			}
		}()
	}
}

func (b *Broker) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ch != nil {
		_ = b.ch.Close()
	}
	if b.conn != nil {
		_ = b.conn.Close()
	}
}
