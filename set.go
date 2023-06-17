package main

type Set[V comparable] struct {
	values map[V]struct{}
}

func NewSet[V comparable]() *Set[V] {
	return &Set[V]{
		map[V]struct{}{},
	}
}

func (s *Set[V]) Contains(v V) bool {
	_, ok := s.values[v]
	return ok
}

func (s *Set[V]) Insert(v V) {
	s.values[v] = struct{}{}
}
