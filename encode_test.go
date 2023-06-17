package main

import (
	"bytes"
	"github.com/leeavital/protoc-gen-gostreamer/example/pb"
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
		w.SetMyThirtyTwo(400)
	})

	builder.AddS(func(w *pb.Thing_SubMessageBuilder) {
		w.SetX(100)
	})

	builder.AddThings(func(w *pb.Thing2Builder) {
		w.SetZ(6)
	})
	builder.SetMyname("hello 🙃")

	var decoded pb.Thing
	err := proto.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)

	expected := pb.Thing{
		X: 1,
		Y: 5,
		S: &pb.Thing_SubMessage{X: 100},
		Things: []*pb.Thing2{
			{Z: 5, MyThirtyTwo: 400},
			{Z: 6},
		},
		Myname: "hello 🙃",
	}
	assert.Truef(t, proto.Equal(&expected, &decoded), "expected equal %s and %s", expected.String(), decoded.String())
}

var sink any

func BenchmarkEncode(b *testing.B) {

	b.Run("protoc-gen-gostreamer", func(b *testing.B) {

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := bytes.NewBuffer(nil)
			builder := pb.NewThingBuilder(w)
			builder.SetMyname("hello")
			builder.SetY(1)
			builder.SetX(2)
			sink = w.Bytes()
		}
	})

	b.Run("protoc-vanilla", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {

			thing := pb.Thing{
				Myname: "hello",
				Y:      1,
				X:      2,
			}
			var err error
			sink, err = proto.Marshal(&thing)
			require.NoError(b, err)
		}
	})

}
