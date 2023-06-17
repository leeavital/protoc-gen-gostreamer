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
			goPkg := file.GetOptions().GetGoPackage()

			outFile := &FileContext{
				generatedFile: gen.NewGeneratedFile(path.Join(goPkg, *file.Name+"_builder.go"), protogen.GoImportPath(goPkg)),
				pkg:           *file.Package,
			}

			outFile.P("// THIS IS A GENERATED FILE")
			outFile.P("// DO NOT EDIT")

			outFile.P("package ", packageShortName(goPkg))

			for _, message := range file.MessageType {
				handleDescriptor(outFile, "", message)
			}
		}

		return nil
	})
}

func handleDescriptor(outFile *FileContext, prefix string, message *descriptorpb.DescriptorProto) {
	builderTypeName := prefix + *message.Name + "Builder"
	constructorName := "New" + builderTypeName

	identBytesBuffer := outFile.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "Buffer",
		GoImportPath: "bytes",
	})

	outFile.P("type ", builderTypeName, " struct {")
	outFile.P("writer ", outFile.SymIoWriter())
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
			fieldTag := fmt.Sprintf("%d", (uint32(*field.Number)<<3)|uint32(0))
			outFile.P(funcPrefix, "Set", capitalizeFirstLetter(*field.Name), "(v int64)", "{")
			outFile.P("var b []byte")
			outFile.P("b = ", outFile.SymAppendVarint(), "(b, ", fieldTag, ")")
			outFile.P("b = ", outFile.SymAppendVarint(), "(b, uint64(v))")
			outFile.P("x.writer.Write(b)")
			outFile.P("}")
		}

		if *field.Type == descriptorpb.FieldDescriptorProto_TYPE_INT32 {
			fieldTag := fmt.Sprintf("%d", (uint32(*field.Number)<<3)|uint32(0))
			outFile.P(funcPrefix, "Set", capitalizeFirstLetter(*field.Name), "(v int32)", "{")
			outFile.P("var b []byte")
			outFile.P("b = ", outFile.SymAppendVarint(), "(b, ", fieldTag, ")")
			outFile.P("b = ", outFile.SymAppendVarint(), "(b, uint64(v))")
			outFile.P("x.writer.Write(b)")
			outFile.P("}")
		}

		if *field.Type == descriptorpb.FieldDescriptorProto_TYPE_STRING {
			fieldTag := fmt.Sprintf("0x%x", (*field.Number<<3)|2)
			outFile.P(funcPrefix, "Set", capitalizeFirstLetter(*field.Name), "(v string) {")
			outFile.P("var b []byte")
			outFile.P("b = ", outFile.SymAppendVarint(), "(b, ", fieldTag, ")")
			outFile.P("b = ", outFile.SymAppendVarint(), "(b, uint64(len(v)))")
			outFile.P("x.writer.Write(b)")
			outFile.P("x.writer.Write([]byte(v))")
			outFile.P("}")
		}

		if *field.Type == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
			fieldTag := fmt.Sprintf("0x%x", (*field.Number<<3)|2)

			subType := getTypeName(outFile, field)
			subWriterType := subType + "Builder"
			outFile.P(funcPrefix, "Add"+capitalizeFirstLetter(*field.Name)+"(cb func(w *"+subWriterType, ")) {")
			outFile.P("x.buf.Reset()")
			outFile.P("subW := New" + subWriterType + "(&x.buf)")
			outFile.P("cb(subW)")
			outFile.P("b := ", outFile.SymAppendVarint(), "(nil, ", fieldTag, ")")
			outFile.P("b = ", outFile.SymAppendVarint(), "(b, uint64(x.buf.Len()))")
			outFile.P("x.writer.Write(b)")
			outFile.P("x.writer.Write(x.buf.Bytes())")
			outFile.P("}")
		}
	}

	for _, m := range message.NestedType {
		handleDescriptor(outFile, capitalizeFirstLetter(*message.Name)+"_", m)
	}

	// TODO: handle message.NestedType

}

func getTypeName(fc *FileContext, field *descriptorpb.FieldDescriptorProto) string {
	norm := strings.TrimPrefix(*field.TypeName, "."+fc.pkg)[1:]
	return strings.ReplaceAll(norm, ".", "_")
}

type FileContext struct {
	generatedFile *protogen.GeneratedFile
	pkg           string
}

func (fc *FileContext) P(parts ...any) {
	fc.generatedFile.P(parts...)
}

func (fc *FileContext) SymAppendVarint() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "AppendVarint",
		GoImportPath: "google.golang.org/protobuf/encoding/protowire",
	})
}

func (fc *FileContext) SymAppendString() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "AppendString",
		GoImportPath: "google.golang.org/protobuf/encoding/protowire",
	})
}

func (fc *FileContext) SymIoWriter() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "Writer",
		GoImportPath: "io",
	})
}

func packageShortName(pkg string) string {
	parts := strings.Split(pkg, "/")
	return parts[len(parts)-1]
}

func capitalizeFirstLetter(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:len(s)]
}
