package node

import (
	"reflect"
	"babyboy-dag/rpc"
	"babyboy-dag/event"
	"babyboy-dag/accounts"
	"errors"
)

type Service interface {
	APIs() []rpc.API
}

type ServiceConstructor func(ctx *ServiceContext) (Service, error)

type ServiceContext struct {
	Services       map[reflect.Type]Service // Index of the already constructed services
	EventMux       *event.TypeMux           // Event multiplexer used for decoupled notifications
	AccountManager *accounts.Manager        // Account manager created by the node.
	Node		   *Node
}

// Service retrieves a currently running service registered of a specific type.
func (ctx *ServiceContext) Service(service interface{}) error {
	element := reflect.ValueOf(service).Elem()
	if running, ok := ctx.Services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}

	return errors.New("unknown service")
}
