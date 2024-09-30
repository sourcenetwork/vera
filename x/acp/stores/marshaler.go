package stores

import (
	gogoproto "github.com/cosmos/gogoproto/proto"
	raccoon "github.com/sourcenetwork/raccoondb"
)

var _ raccoon.Marshaler[gogoproto.Message] = (*GogoProtoMarshaler[gogoproto.Message])(nil)

func NewGogoProtoMarshaler[T gogoproto.Message](factory func() T) *GogoProtoMarshaler[T] {
	return &GogoProtoMarshaler[T]{
		factory: factory,
	}
}

type GogoProtoMarshaler[T gogoproto.Message] struct {
	factory func() T
}

func (m *GogoProtoMarshaler[T]) Marshal(t *T) ([]byte, error) {
	return gogoproto.Marshal(*t)
}

func (m *GogoProtoMarshaler[T]) Unmarshal(bytes []byte) (T, error) {
	t := m.factory()
	err := gogoproto.Unmarshal(bytes, t)
	if err != nil {
		return t, err
	}

	return t, nil
}
