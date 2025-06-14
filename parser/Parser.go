package parser

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
)

// Parser and Callback types
type ParserFunc[T any] func([]byte) (bool, T)
type CallbackFunc[T any] func(T)

// messageHandler interface to unify all handler types
type messageHandler interface {
	tryParseAndCallback([]byte) bool
}

// Concrete handler for type T
type TypedHandler[T any] struct {
	parser   ParserFunc[T]
	callback CallbackFunc[T]
}

func (h TypedHandler[T]) tryParseAndCallback(data []byte) bool {
	if ok, val := h.parser(data); ok {
		h.callback(val)
		return true
	}
	return false
}

// MessageParsers_Registry to store generic handlers
type MessageParsers_Registry struct {
	mu       sync.RWMutex
	handlers []messageHandler
}

func (parserRegistry *MessageParsers_Registry) TryDispatch(msg []byte) (callback_called bool) {
	fmt.Println(len(parserRegistry.handlers), "handlers")
	for _, handler := range parserRegistry.handlers {
		if handler.tryParseAndCallback(msg) {
			return true
		}
	}

	return false
}

// Generic function to register parser and callback for type T
//
// if `parser` is nil, the callback will be called upon any non-error unmarshall of messages (not recommended unless it is the only message structure that is received)
func RegisterMessageParserCallback[T any](r *MessageParsers_Registry, parser ParserFunc[*T], callback CallbackFunc[*T]) *TypedHandler[*T] {
	r.mu.Lock()
	defer r.mu.Unlock()

	if parser == nil {
		parser = func(b []byte) (bool, *T) {
			var v T
			err := json.Unmarshal(b, &v)
			if err != nil {
				return false, nil
			}

			return true, &v
		}
	}

	messageHandler := TypedHandler[*T]{parser, callback}

	r.handlers = append(r.handlers, messageHandler)

	return &messageHandler
}

// Generic function to register parser and callback for type T
//
// if `parser` is nil, the callback will be called upon any non-error unmarshall of messages (not recommended unless it is the only message structure that is received)
func DeregisterMessageParserCallback[T any](r *MessageParsers_Registry, handler *TypedHandler[*T]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, h := range r.handlers {
		if typed, ok := h.(TypedHandler[*T]); ok {
			if &typed == handler {
				// Remove element from slice
				r.handlers = slices.Delete(r.handlers, i, i+1)
				break
			}
		}
	}
}

// This is how it should be in the struct
//
// // In order to register the parsed messages
// MessageParserRegistry messageParsers_Registry
