# protoc-gen-gostreamer

`protoc-gen-gostreamer` is a protobuf generator go which generates 'builder' objects that allow protobuf objects to be serialized directly to an `io.Writer`.

The primary use case for these builders is serializing large protobuf objects directly to the wire without requiring the whole protobuf object to be allocated in Go at once. This can be useful when objects exist long term in memory in a non-protobuf format.

