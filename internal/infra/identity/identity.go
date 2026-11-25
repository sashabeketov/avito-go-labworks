package identity

import "uuid"

type Generator struct{}

func (Generator) New() uuid.UUID { return uuid.New() }
