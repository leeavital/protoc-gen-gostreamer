package main

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/descriptorpb"
	"testing"
)

func TestSet(t *testing.T) {

	s := NewSet[descriptorpb.FieldDescriptorProto_Type]()

	assert.False(t, s.Contains(descriptorpb.FieldDescriptorProto_TYPE_INT64))
	assert.False(t, s.Contains(descriptorpb.FieldDescriptorProto_TYPE_INT32))

	s.Insert(descriptorpb.FieldDescriptorProto_TYPE_INT32)
	assert.False(t, s.Contains(descriptorpb.FieldDescriptorProto_TYPE_INT64))
	assert.True(t, s.Contains(descriptorpb.FieldDescriptorProto_TYPE_INT32))
}
