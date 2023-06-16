package main

import (
	"bytes"
	"fmt"
	"github.com/leeavital/protobuilder/example/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestFoo(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	builder := pb.NewThingBuilder(buf)

	builder.SetX(1)
	builder.SetY(5)
	builder.AddThings(func(w *pb.Thing2Builder) {
		w.SetZ(5)
	})

	builder.AddThings(func(w *pb.Thing2Builder) {
		w.SetZ(6)
	})

	fmt.Printf("encoded (ours) %v\n", buf.Bytes())

	var decoded pb.Thing
	err := proto.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)

	assert.Equal(t, int64(1), decoded.X)
	assert.Equal(t, int64(5), decoded.Y)
	assert.Equal(t, int64(5), decoded.Things[0].Z)
	assert.Equal(t, int64(6), decoded.Things[1].Z)

	decoded.X = 1
	bs, _ := proto.Marshal(&decoded)
	fmt.Printf("encoded (real) %v\n", bs)
}
