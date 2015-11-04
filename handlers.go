package client

import log "github.com/Sirupsen/logrus"

/**
 * HandlerManagers are used to process incoming messages from rotone.
 * Simply presented, it is just a graph of conditionnal demultiplexers.
 * They have an input channel on one side, and multiple output channels on
 * the other side.
 * Between the input channel and the output channels there is a function that
 * can filter messages, or alter them.
 */

// HandlerManagerFilterFunc type for the function that can filter or alter messages
type HandlerManagerFilterFunc func(interface{}) (interface{}, bool)
type HandlerFunc func(interface{}) bool

// handlerCtrl interface for HandlerManager control methods
// eg. when you call addOutChan, it actually sends a Ctrl message
// in the ctrlChan of the handler.
type handlerCtrl interface {
	exec(h *HandlerManager)
}

// ctrlAddOutChan handlerCtrl message that packages the chan to add
type ctrlAddOutChan struct {
	outChan chan interface{}
}

func (ctrl *ctrlAddOutChan) exec(h *HandlerManager) {
	log.Info("ctrlAddOutChan.exec")
	h.outChans = append(h.outChans, ctrl.outChan)
	if len(h.outChans) == 1 {
		h.firstFn()
	}
}

// ctrlAddOutChan handlerCtrl message that packages the chan to add
type ctrlRemoveOutChan struct {
	outChan chan interface{}
}

func (ctrl *ctrlRemoveOutChan) exec(h *HandlerManager) {
	log.Info("ctrlRemoveOutChan.exec")
	for i, outChan := range h.outChans {
		if outChan == ctrl.outChan {
			if i < len(h.outChans)-1 {
				copy(h.outChans[i:], h.outChans[i+1:])
			}
			h.outChans = h.outChans[0 : len(h.outChans)-1]
			if len(h.outChans) == 0 {
				h.lastFn()
			}
			return
		}
	}
}

// HandlerManager main struct
type HandlerManager struct {
	inChan   chan interface{}
	outChans []chan interface{}
	ctrlChan chan handlerCtrl

	fn HandlerManagerFilterFunc

	firstFn func()
	lastFn  func()
}

func NewHandlerManager(inChan chan interface{}, fn HandlerManagerFilterFunc, firstFn func(), lastFn func()) (h *HandlerManager) {
	h = new(HandlerManager)
	h.inChan = inChan
	h.ctrlChan = make(chan handlerCtrl, 5)
	h.outChans = make([]chan interface{}, 0, 20)

	h.fn = fn

	h.firstFn = firstFn
	h.lastFn = lastFn

	go h.start()
	return
}

func (h *HandlerManager) Attach(fn HandlerFunc) {
	c := make(chan interface{}, 10)

	go func() {
		defer h.RemoveOutChan(c)
		for m := range c {
			if fn(m) == false {
				break
			}
		}
	}()
	h.AddOutChan(c)
}

func (h *HandlerManager) AddOutChan(outChan chan interface{}) {
	h.ctrlChan <- &ctrlAddOutChan{outChan}
}

func (h *HandlerManager) RemoveOutChan(outChan chan interface{}) {
	h.ctrlChan <- &ctrlRemoveOutChan{outChan}
}

func (h *HandlerManager) dispatchMessage(message interface{}) {
	for _, c := range h.outChans {
		go func(c chan interface{}) {
			defer func() {
				if recover() == nil {
					return
				}
				h.RemoveOutChan(c)
			}()
			c <- message
		}(c)
	}
}

func (h *HandlerManager) start() {
	for {
		select {
		case ctrl := <-h.ctrlChan:
			ctrl.exec(h)
		case m := <-h.inChan:
			if m, ok := h.fn(m); ok == true {
				h.dispatchMessage(m)
			}
		}
	}
}

// utils

func noop() {}

func passAll(m interface{}) (interface{}, bool) {
	return m, true
}
