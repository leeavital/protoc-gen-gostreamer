package main

import (
	"bytes"
	"github.com/leeavital/protoc-gen-gostreamer/example/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"math"
	"testing"
)

func TestEncodeAndDecode(t *testing.T) {

	buf := bytes.NewBuffer(nil)
	builder := pb.NewThingBuilder(buf)

	sampleBytes := []byte{0xF, 0xE, 0xE, 0xD}

	builder.SetX(1)
	builder.SetY(5)

	builder.SetS(func(w *pb.Thing_SubMessageBuilder) {
		w.SetX(100)
	})

	builder.AddThings(func(w *pb.Thing2Builder) {
		w.SetZ(5)
		w.SetMyThirtyTwo(400)
		w.SetRatio(100.0)
		w.SetRawMessage(func(b *bytes.Buffer) {
			b.Write(sampleBytes)
		})

	})
	builder.AddThings(func(w *pb.Thing2Builder) {
		w.SetZ(math.MaxInt64)
		w.SetMyThirtyTwo(math.MaxInt32)
		w.SetMyFixed64(math.MaxUint64)
		w.SetMySfixed64(math.MaxInt64)

	})

	builder.AddThings(func(w *pb.Thing2Builder) {
		w.SetZ(math.MinInt64)
		w.SetMyThirtyTwo(math.MinInt32)
		w.SetMyFixed64(0)
		w.SetMySfixed64(math.MinInt64)

		w.AddMyIntegers(0)
		w.AddMyIntegers(1)
		w.AddMyIntegers(2)

	})

	builder.AddMyname("hello 🙃")

	builder.SetWhat_color(uint64(pb.Color_Blue))
	builder.SetIs_valid(true)

	builder.SetMyBigUint(600)
	builder.SetMySmallerUint(300)

	builder.AddMy_map(func(w *pb.Thing_MyMapEntryBuilder) {
		w.SetKey(3)
		w.SetValue(300)
	})

	builder.AddMy_map(func(w *pb.Thing_MyMapEntryBuilder) {
		w.SetKey(4)
		w.SetValue(400)
	})

	builder.AddMy_map(func(w *pb.Thing_MyMapEntryBuilder) {
		w.SetKey(100)
		w.SetValue(0)
	})

	builder.AddMy_map(func(w *pb.Thing_MyMapEntryBuilder) {
		w.SetKey(0)
		w.SetValue(0)
	})

	var decoded pb.Thing
	err := proto.Unmarshal(buf.Bytes(), &decoded)
	require.NoError(t, err)

	expected := pb.Thing{
		X:         1,
		Y:         5,
		S:         &pb.Thing_SubMessage{X: 100},
		WhatColor: pb.Color_Blue,
		IsValid:   true,
		Things: []*pb.Thing2{
			{Z: 5, MyThirtyTwo: 400, Ratio: 100.0, RawMessage: sampleBytes},
			{Z: math.MaxInt64, MyThirtyTwo: math.MaxInt32, MyFixed64: math.MaxUint64, MySfixed64: math.MaxInt64},
			{Z: math.MinInt64, MyThirtyTwo: math.MinInt32, MyFixed64: 0, MySfixed64: math.MinInt64, MyIntegers: []int32{0, 1, 2}},
		},
		Myname:        []string{"hello 🙃"},
		MyBigUint:     600,
		MySmallerUint: 300,
		MyMap: map[int32]uint32{
			3:   300,
			4:   400,
			100: 0,
			0:   0,
		},
	}
	assert.Truef(t, proto.Equal(&expected, &decoded), "expected equal\n\t%s\n\t%s", expected.String(), decoded.String())

	newBytesBuf := bytes.NewBuffer(nil)
	builder.Reset(newBytesBuf)
	builder.SetX(1)

	proto.Unmarshal(newBytesBuf.Bytes(), &decoded)
	newExpected := &pb.Thing{
		X: 1,
	}
	assert.Truef(t, proto.Equal(newExpected, &decoded), "expected equal\n\t%s\n\t%s", expected.String(), decoded.String())
}

var sink any

func BenchmarkEncode(b *testing.B) {

	longString := "hello this is an extremely medium string"

	b.Run("protoc-gen-gostreamer", func(b *testing.B) {

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			w := bytes.NewBuffer(nil)
			builder := pb.NewThingBuilder(w)
			for i := 0; i < 100; i++ {
				builder.AddMyname(longString)
				builder.AddThings(func(w *pb.Thing2Builder) {
					w.SetZ(100)
				})
			}
			sink = w.Bytes()
		}
	})

	b.Run("protoc-vanilla", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			thing := pb.Thing{}
			for i := 0; i < 100; i++ {
				thing.Myname = append(thing.Myname, longString)
				thing.Things = append(thing.Things, &pb.Thing2{
					Z: 100,
				})
			}
			var err error
			sink, err = proto.Marshal(&thing)
			require.NoError(b, err)
		}
	})

}
