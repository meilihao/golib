package sse

import (
	"encoding/json"
	"sync"

	"github.com/meilihao/golib/v2/log"
	"go.uber.org/zap"
)

type Client[T, X comparable] struct {
	Id      T
	Account X
	Channel chan *Data[X]
}

func (c *Client[T, X]) Close() {
	close(c.Channel)
}

type Manager[T, X comparable] struct {
	lock           sync.RWMutex
	clients        map[T]*Client[T, X]
	accounts       map[X]map[T]*Client[T, X]
	Register       chan *Client[T, X]
	Unregister     chan *Client[T, X]
	ForwardMessage chan *Data[X]
}

func NewManager[T, X comparable]() *Manager[T, X] {
	m := &Manager[T, X]{
		clients:        make(map[T]*Client[T, X]),
		accounts:       make(map[X]map[T]*Client[T, X]),
		Register:       make(chan *Client[T, X]),
		Unregister:     make(chan *Client[T, X]),
		ForwardMessage: make(chan *Data[X], 32),
	}

	go m.Run()

	return m
}

func (m *Manager[T, X]) HasClient(client *Client[T, X]) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	_, ok := m.clients[client.Id]
	return ok
}

func (m *Manager[T, X]) Run() {
	for {
		select {
		case client := <-m.Register:
			m.lock.Lock()

			if _, exists := m.clients[client.Id]; !exists {
				m.clients[client.Id] = client
			}
			if _, exists := m.accounts[client.Account]; !exists {
				m.accounts[client.Account] = make(map[T]*Client[T, X], 3)
			}
			m.accounts[client.Account][client.Id] = client

			log.Glog.Info("client registered", zap.Any("id", client.Id), zap.Any("account", client.Account))

			m.lock.Unlock()
		case client := <-m.Unregister:
			m.lock.Lock()

			if c, exists := m.clients[client.Id]; exists {
				delete(m.clients, client.Id)
				delete(m.accounts[client.Account], client.Id)

				c.Close()
			}

			log.Glog.Info("client unregistered", zap.Any("id", client.Id), zap.Any("account", client.Account))

			m.lock.Unlock()
		case data := <-m.ForwardMessage:
			m.lock.Lock()

			log.Glog.Debug("forward message", zap.Any("msg", data))

			if cs, exists := m.accounts[data.To]; exists {
				for _, c := range cs {
					c.Channel <- data
				}
			}

			m.lock.Unlock()
		}
	}
}

// Data to be broadcasted to a client.
type Data[X comparable] struct {
	Message json.RawMessage `json:"message"`
	From    X               `json:"sender"`
	To      X               `json:"receiver"`
}
