package client

import "github.com/HackerLoop/rotonde/shared"

type Client struct {
	localDefinitions  map[string]*rotonde.Definitions
	remoteDefinitions map[string]*rotonde.Definitions

	DefinitionHandler   *HandlerManager
	UnDefinitionHandler *HandlerManager
	EventHandler        *HandlerManager
	ActionHandler       *HandlerManager
}

func NewClient(rotondeUrl string) (c *Client) {
	c = new(Client)
	c.localDefinitions = make(map[string]*rotonde.Definitions)
	c.remoteDefinitions = make(map[string]*rotonde.Definitions)

	jsonOutChan := make(chan interface{}, 30)
	jsonInChan := make(chan interface{}, 30)

	go startConnection(rotondeUrl, jsonInChan, jsonOutChan)

	mainHandler := NewHandlerManager(jsonOutChan, passAll, noop, noop)

	c.DefinitionHandler = NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.Definition); return }, noop, noop)
	mainHandler.AddOutChan(c.DefinitionHandler.inChan)

	c.UnDefinitionHandler = NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.UnDefinition); return }, noop, noop)
	mainHandler.AddOutChan(c.UnDefinitionHandler.inChan)

	c.EventHandler = NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.Event); return }, noop, noop)
	mainHandler.AddOutChan(c.EventHandler.inChan)

	c.ActionHandler = NewHandlerManager(make(chan interface{}, 10), func(m interface{}) (r interface{}, ok bool) { r, ok = m.(rotonde.Action); return }, noop, noop)
	mainHandler.AddOutChan(c.ActionHandler.inChan)
	return
}
