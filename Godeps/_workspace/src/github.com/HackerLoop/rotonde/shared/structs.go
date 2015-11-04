package rotonde

import (
	"errors"
	"fmt"
)

// wrapper for json serialized connections
type Packet struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Definitions is a slice of Definition, adds findBy
type Definitions []*Definition

// GetDefinitionForIdentifier _
func (definitions Definitions) GetDefinitionForIdentifier(identifier string) (*Definition, error) {
	for _, definition := range definitions {
		if definition.Identifier == identifier {
			return definition, nil
		}
	}
	return nil, errors.New(fmt.Sprint(identifier, " Not found"))
}

func PushDefinition(definitions Definitions, def *Definition) Definitions {
	if _, err := definitions.GetDefinitionForIdentifier(def.Identifier); err == nil {
		return definitions
	}
	return append(definitions, def)
}

func RemoveDefinition(definitions Definitions, identifier string) Definitions {
	for i, definition := range definitions {
		if definition.Identifier == identifier {
			if i < len(definitions)-1 {
				copy(definitions[i:], definitions[i+1:])
			}
			definitions = definitions[0 : len(definitions)-1]
			return definitions
		}
	}
	return definitions
}

// Fields sortable slice of fields
type FieldDefinitions []*FieldDefinition

// FieldDefinition _
type FieldDefinition struct {
	Name  string `json:"name"`
	Type  string `json:"type"` // string, number or boolean
	Units string `json:"units"`
}

// Definition, used to expose an action or event
type Definition struct {
	Identifier string `json:"identifier"`
	Type       string `json:"type"` // action or event

	Fields FieldDefinitions `json:"fields"`
}

func (d *Definition) PushField(n, t, u string) {
	field := FieldDefinition{n, t, u}
	d.Fields = append(d.Fields, &field)
}

type UnDefinition Definition

// Object native representation of an event or action, just a map
type Object map[string]interface{}

type Event struct {
	Identifier string `json:"identifier"`
	Data       Object `json:"data"`
}

type Action struct {
	Identifier string `json:"identifier"`
	Data       Object `json:"data"`
}

// Subscription adds an objectID to the subscriptions of the sending connection
type Subscription struct {
	Identifier string `json:"identifier"`
}

// Unsubscription removes an objectID from the subscriptions of the sending connection
type Unsubscription struct {
	Identifier string `json:"identifier"`
}
