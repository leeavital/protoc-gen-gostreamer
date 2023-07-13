# protoc-gen-gostreamer

`protoc-gen-gostreamer` is a protobuf generator go which generates 'builder' objects that allow protobuf objects to be serialized directly to an `io.Writer`.

The primary use case for these builders is serializing large protobuf objects directly to onto the wire without requiring the whole protobuf object or raw bytes to be allocated in Go at once. This can be useful when objects exist long term in memory in a non-protobuf format.



## Example

Take the following code which implements:

```
func getWidgets(ids []int, resp *http.ResponseWriter) {
    widgetResponse := WidgetResponse{}
    for _, i := range ids {
        widgetResponse.widgets = append(widgetResponse.widgets, convertToProto(store.get(id)))
    }
    marshalled, _ := widgetResponse.Marshal()
    resp.Write(marshalled)
}
```


It can create quite a few short lived short-lived allocations.

1. the []byte from `widgetResponse.Marshal()` is an allocation.
2. each widget object returned by `convertToProto` is an allocation.

Using gostreamer, the same endpoint can be written as:

```
func getWidgets(ids []int, resp *http.ResponseWriter) {
    builder := NewWidgetResponseBuilder(resp)
    for _, i := range ids {
        widget := store.get(i)
        builder.AddWidget(func(wb *WidgetBuilder) {
            wb.SetName(widget.Name)
            wb.SetId(widget.Id)
        })
    }
}
```

In this style, only fixed size buffers are allocated in NewWidgetResponseBuilder, and the allocations remain constant no matter how many widgets are ultimately serialized.
