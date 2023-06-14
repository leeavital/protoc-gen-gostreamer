package main

import (
	"fmt"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
	"path"
	"strings"
)

func main() {
	opts := protogen.Options{}

	opts.Run(func(gen *protogen.Plugin) error {
		files := gen.Request.GetProtoFile()

		for _, file := range files {
			pkg := file.GetOptions().GetGoPackage()
			outFile := gen.NewGeneratedFile(path.Join(pkg, *file.Name+"_builder.go"), protogen.GoImportPath(pkg))

			outFile.P("package ", packageShortName(pkg))

			for _, message := range file.MessageType {
				builderTypeName := *message.Name + "Builder"
				constructorName := "New" + builderTypeName

				identIOWriter := outFile.QualifiedGoIdent(protogen.GoIdent{
					GoName:       "Writer",
					GoImportPath: "io",
				})

				identBytesBuffer := outFile.QualifiedGoIdent(protogen.GoIdent{
					GoName:       "Buffer",
					GoImportPath: "bytes",
				})

				outFile.P("type ", builderTypeName, " struct {")
				outFile.P("writer ", identIOWriter)
				outFile.P("buf ", identBytesBuffer)
				outFile.P("}")

				outFile.P("func ", constructorName, "(writer io.Writer) *", builderTypeName, "{")
				outFile.P("return &", builderTypeName, "{")
				outFile.P("writer: writer,")
				outFile.P("}")
				outFile.P("}")

				for _, field := range message.Field {
					funcPrefix := "func(x *" + builderTypeName + ") "

					if *field.Type == descriptorpb.FieldDescriptorProto_TYPE_INT64 { // TODO: type int64

						fieldTag := fmt.Sprintf("%d", (*field.Number)<<3|0)
						identBinaryAppendVarint := outFile.QualifiedGoIdent(protogen.GoIdent{
							GoName:       "AppendVarint",
							GoImportPath: "encoding/binary",
						})

						outFile.P(funcPrefix, "Set", capitalizeFirstLetter(*field.Name), "(v int64)", "{")
						outFile.P("var b []byte")
						outFile.P("b = ", identBinaryAppendVarint, "(b, "+fieldTag+")")
						outFile.P("b = ", identBinaryAppendVarint, "(b, v)")
						outFile.P("x.writer.Write(b)")
						outFile.P("}")
					}

					if *field.Type == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {

						fieldTag := fmt.Sprintf("%d", (*field.Number)<<3|2)
						identBinaryAppendVarint := outFile.QualifiedGoIdent(protogen.GoIdent{
							GoName:       "AppendVarint",
							GoImportPath: "encoding/binary",
						})

						subType := capitalizeFirstLetter((*field.TypeName)[1:])
						subWriterType := subType + "Builder"
						outFile.P(funcPrefix, "Add"+subType+"(cb func(w *"+subWriterType, ")) {")
						outFile.P("x.buf.Reset()")
						outFile.P("subW := New" + subWriterType + "(&x.buf)")
						outFile.P("cb(subW)")
						outFile.P("b := ", identBinaryAppendVarint, "(nil, ", fieldTag, ")")
						outFile.P("b = ", identBinaryAppendVarint, "(b, int64(x.buf.Len()))")
						outFile.P("x.writer.Write(b)")
						outFile.P("x.writer.Write(x.buf.Bytes())")
						outFile.P("}")

					}
				}

			}
		}

		return nil
	})
}

func packageShortName(pkg string) string {
	parts := strings.Split(pkg, "/")
	return parts[len(parts)-1]
}

func capitalizeFirstLetter(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:len(s)]
}
