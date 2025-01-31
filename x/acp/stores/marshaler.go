package stores

import (
	gogoproto "github.com/cosmos/gogoproto/proto"
	raccoon "github.com/sourcenetwork/raccoondb"
)

var _ raccoon.Marshaler[gogoproto.Message] = (*gogoProtoMarshaler[gogoproto.Message])(nil)

// NewGogoProtoMarshaler returns a raccoon Marshaler
// which marshals and unmarshals a Message using gogoproto.
//
// Requires a factory method which returns an instance of T
func NewGogoProtoMarshaler[T gogoproto.Message](factory func() T) raccoon.Marshaler[T] {
	return &gogoProtoMarshaler[T]{
		factory: factory,
	}
}

type gogoProtoMarshaler[T gogoproto.Message] struct {
	factory func() T
}

func (m *gogoProtoMarshaler[T]) Marshal(t *T) ([]byte, error) {
	return gogoproto.Marshal(*t)
}

func (m *gogoProtoMarshaler[T]) Unmarshal(bytes []byte) (T, error) {
	t := m.factory()
	err := gogoproto.Unmarshal(bytes, t)
	if err != nil {
		return t, err
	}

	return t, nil
}
