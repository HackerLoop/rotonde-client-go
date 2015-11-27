package client

import (
	"errors"
	"fmt"
	"sync"

	"github.com/HackerLoop/rotonde/shared"
	"github.com/vitaminwater/handlers.go"
)

// TODO dry

type Client struct {
	mutex *sync.Mutex

	localDefinitions  map[string]rotonde.Definitions
	remoteDefinitions map[string]rotonde.Definitions

	jsonOutChan chan interface{}
	jsonInChan  chan interface{}

	definitionHandler         *handlers.HandlerManager
	namedDefinitionHandlers   map[string]*handlers.HandlerManager
	unDefinitionHandler       *handlers.HandlerManager
	namedUnDefinitionHandlers map[string]*handlers.HandlerManager
	eventHandler              *handlers.HandlerManager
	namedEventHandlers        map[string]*handlers.HandlerManager
	actionHandler             *handlers.HandlerManager
	namedActionHandlers       map[string]*handlers.HandlerManager
}

func NewClient(rotondeUrl string) (c *Client) {
	c = new(Client)
	c.mutex = &sync.Mutex{}
	c.localDefinitions = make(map[string]rotonde.Definitions)
	c.remoteDefinitions = make(map[string]rotonde.Definitions)

	c.jsonOutChan = make(chan interface{}, 30)
	c.jsonInChan = make(chan interface{}, 30)

	go startConnection(rotondeUrl, c.jsonInChan, c.jsonOutChan)

	mainHandler := handlers.NewHandlerManager(c.jsonOutChan, handlers.PassAll, handlers.Noop, handlers.Noop)

	c.definitionHandler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.Definition); return }, handlers.Noop, handlers.Noop)
	mainHandler.AddOutChan(c.definitionHandler.InChan)
	c.namedDefinitionHandlers = make(map[string]*handlers.HandlerManager)

	c.definitionHandler.Attach(func(d interface{}) bool {
		def := d.(rotonde.Definition)
		c.addRemoteDefinition(&def)
		return true
	})

	c.unDefinitionHandler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.UnDefinition); return }, handlers.Noop, handlers.Noop)
	mainHandler.AddOutChan(c.unDefinitionHandler.InChan)
	c.namedUnDefinitionHandlers = make(map[string]*handlers.HandlerManager)

	c.unDefinitionHandler.Attach(func(d interface{}) bool {
		def := d.(rotonde.UnDefinition)
		c.removeRemoteDefinition(def.Type, def.Identifier)
		return true
	})

	c.eventHandler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.Event); return }, handlers.Noop, handlers.Noop)
	mainHandler.AddOutChan(c.eventHandler.InChan)
	c.namedEventHandlers = make(map[string]*handlers.HandlerManager)

	c.actionHandler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.Action); return }, handlers.Noop, handlers.Noop)
	mainHandler.AddOutChan(c.actionHandler.InChan)
	c.namedActionHandlers = make(map[string]*handlers.HandlerManager)
	return
}

func (c *Client) addRemoteDefinition(d *rotonde.Definition) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	definitions, ok := c.remoteDefinitions[d.Type]
	if ok == false {
		definitions = make([]*rotonde.Definition, 0, 10)
	}
	_, err := definitions.GetDefinitionForIdentifier(d.Identifier)
	if err == nil {
		return
	}
	definitions = rotonde.PushDefinition(definitions, d)
	c.remoteDefinitions[d.Type] = definitions
}

func (c *Client) removeRemoteDefinition(typ string, identifier string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	definitions, ok := c.remoteDefinitions[typ]
	if ok == false {
		return
	}
	definitions = rotonde.RemoveDefinition(definitions, identifier)
	c.remoteDefinitions[typ] = definitions
}

func (c *Client) GetRemoteDefinition(typ string, identifier string) (*rotonde.Definition, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	definitions, ok := c.remoteDefinitions[typ]
	if ok == false {
		return nil, errors.New(fmt.Sprint(identifier, " Not found"))
	}
	d, err := definitions.GetDefinitionForIdentifier(identifier)
	return d, err
}

func (c *Client) AddLocalDefinition(d *rotonde.Definition) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	definitions, ok := c.localDefinitions[d.Type]
	if ok == false {
		definitions = make([]*rotonde.Definition, 0, 10)
		c.localDefinitions[d.Type] = definitions
	}
	_, err := definitions.GetDefinitionForIdentifier(d.Identifier)
	if err == nil {
		return
	}
	definitions = append(definitions, d)
	c.localDefinitions[d.Type] = definitions
	c.jsonInChan <- *d
}

func (c *Client) RemoveLocalDefinition(typ string, identifier string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	definitions, ok := c.localDefinitions[typ]
	if ok == false {
		return
	}
	definition, err := definitions.GetDefinitionForIdentifier(identifier)
	if err != nil {
		return
	}
	definitions = rotonde.RemoveDefinition(definitions, identifier)
	c.localDefinitions[typ] = definitions
	c.jsonInChan <- rotonde.UnDefinition{definition.Identifier, definition.Type, definition.IsArray, definition.Fields}
}

func (c *Client) SendMessage(message interface{}) {
	c.jsonInChan <- message
}

func (c *Client) SendEvent(identifier string, data rotonde.Object) {
	c.jsonInChan <- rotonde.Event{
		identifier,
		data,
	}
}

func (c *Client) SendAction(identifier string, data rotonde.Object) {
	c.jsonInChan <- rotonde.Action{
		identifier,
		data,
	}
}

func (c *Client) OnDefinition(fn handlers.HandlerFunc) {
	c.definitionHandler.Attach(fn)
}

func (c *Client) OnNamedDefinition(identifier string, fn handlers.HandlerFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	handler, ok := c.namedDefinitionHandlers[identifier]
	if ok == false {
		handler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (interface{}, bool) { return m, m.(rotonde.Definition).Identifier == identifier }, handlers.Noop, handlers.Noop)
		c.definitionHandler.AddOutChan(handler.InChan)
		c.namedDefinitionHandlers[identifier] = handler
	}
	handler.Attach(fn)
}

func (c *Client) OnUnDefinition(fn handlers.HandlerFunc) {
	c.unDefinitionHandler.Attach(fn)
}

func (c *Client) OnNamedUnDefinition(identifier string, fn handlers.HandlerFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	handler, ok := c.namedUnDefinitionHandlers[identifier]
	if ok == false {
		handler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (interface{}, bool) { return m, m.(rotonde.UnDefinition).Identifier == identifier }, handlers.Noop, handlers.Noop)
		c.unDefinitionHandler.AddOutChan(handler.InChan)
		c.namedUnDefinitionHandlers[identifier] = handler
	}
	handler.Attach(fn)
}

func (c *Client) OnEvent(fn handlers.HandlerFunc) {
	c.eventHandler.Attach(fn)
}

func (c *Client) OnNamedEvent(identifier string, fn handlers.HandlerFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	handler, ok := c.namedEventHandlers[identifier]
	if ok == false {
		handler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (interface{}, bool) { return m, m.(rotonde.Event).Identifier == identifier }, func() {
			c.SendMessage(rotonde.Subscription{identifier})
		}, func() {
			c.SendMessage(rotonde.Unsubscription{identifier})
		})
		c.eventHandler.AddOutChan(handler.InChan)
		c.namedEventHandlers[identifier] = handler
	}
	handler.Attach(fn)
}

func (c *Client) OnAction(fn handlers.HandlerFunc) {
	c.actionHandler.Attach(fn)
}

func (c *Client) OnNamedAction(identifier string, fn handlers.HandlerFunc) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	handler, ok := c.namedActionHandlers[identifier]
	if ok == false {
		handler = handlers.NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (interface{}, bool) { return m, m.(rotonde.Action).Identifier == identifier }, handlers.Noop, handlers.Noop)
		c.actionHandler.AddOutChan(handler.InChan)
		c.namedActionHandlers[identifier] = handler
	}
	handler.Attach(fn)
}
