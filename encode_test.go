package main

import (
	"bytes"
	"github.com/leeavital/protobuilder/example/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestEncodeAndDecode(t *testing.T) {
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

	var decoded pb.Thing
	err := proto.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)

	expected := pb.Thing{
		X: 1,
		Y: 5,
		Things: []*pb.Thing2{
			{Z: 5},
			{Z: 6},
		},
	}
	assert.Truef(t, proto.Equal(&expected, &decoded), "expected equal %s and %s", expected.String(), decoded.String())
}
