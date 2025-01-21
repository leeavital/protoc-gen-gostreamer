package main

import (
	"fmt"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
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
			}

			if file.Package != nil {
				outFile.pkg = *file.Package
			}

			outFile.P("// THIS IS A GENERATED FILE")
			outFile.P("// DO NOT EDIT")

			outFile.P("package ", packageShortName(goPkg))

			for _, message := range file.MessageType {
				if err := handleDescriptor(outFile, "", message); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func handleDescriptor(outFile *FileContext, prefix string, message *descriptorpb.DescriptorProto) error {
	builderTypeName := prefix + *message.Name + "Builder"
	constructorName := "New" + builderTypeName

	identBytesBuffer := outFile.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "Buffer",
		GoImportPath: "bytes",
	})

	outFile.P("type ", builderTypeName, " struct {")
	outFile.P("writer ", outFile.SymIoWriter())
	outFile.P("buf ", identBytesBuffer)
	outFile.P("scratch []byte")

	seenTypes := NewSet[string]()
	for _, f := range message.Field {
		if *f.Type == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE && !seenTypes.Contains(*f.TypeName) {
			outFile.P(lowerCaseFirstLetter(getTypeName(outFile, f))+"Builder", " ", getTypeName(outFile, f)+"Builder")
			seenTypes.Insert(*f.TypeName)
		}
	}

	outFile.P("}") // end builder struct definition

	// start constructor
	outFile.P("func ", constructorName, "(writer io.Writer) *", builderTypeName, "{")
	outFile.P("return &", builderTypeName, "{")
	outFile.P("writer: writer,")
	outFile.P("}")
	outFile.P("}")
	// end constructor

	// start Reset() method
	outFile.P("func (x *", builderTypeName, ") Reset(writer io.Writer) {")
	outFile.P("x.buf.Reset()")
	outFile.P("x.writer = writer")
	outFile.P("}")
	// end Reset() method

	for _, field := range message.Field {
		funcPrefix := "func(x *" + builderTypeName + ") "

		switch *field.Type {

		// varint encoded fields
		case descriptorpb.FieldDescriptorProto_TYPE_INT64:
			handleVarintField(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_INT32:
			handleVarintField(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_UINT32:
			handleVarintField(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_UINT64:
			handleVarintField(outFile, builderTypeName, field)

		case descriptorpb.FieldDescriptorProto_TYPE_SINT32:
			handleSigned(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_SINT64:
			handleSigned(outFile, builderTypeName, field)

			// fixed int encoded fields
		case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
			handleFixed64(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
			handleFixed64(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
			handleFixed64(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
			handleFixed64(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
			handleFixed64(outFile, builderTypeName, field)
		case descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
			handleFixed64(outFile, builderTypeName, field)

			// everything else
		case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
			fieldTag := fmt.Sprintf("0x%x", (*field.Number<<3)|0)
			funcName := getSetterName(field)
			outFile.P(funcPrefix, " ", funcName, "(v bool) {")
			outFile.P("if v {")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch[:0], ", fieldTag, ")")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, 1)")
			outFile.P("x.writer.Write(x.scratch)")
			outFile.P("}") // end if
			outFile.P("}")

		case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
			fieldTag := fmt.Sprintf("0x%x", (*field.Number<<3)|0)
			funcName := getSetterName(field)
			outFile.P(funcPrefix, funcName, "(v uint64) {")
			outFile.P("if v != 0 {")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch[:0], ", fieldTag, ")")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, v)")
			outFile.P("x.writer.Write(x.scratch)")
			outFile.P("}") // end if
			outFile.P("}")

		case descriptorpb.FieldDescriptorProto_TYPE_STRING:
			fieldTag := fmt.Sprintf("0x%x", (*field.Number<<3)|2)
			funcName := getSetterName(field)
			outFile.P(funcPrefix, funcName, "(v string) {")
			outFile.P("x.scratch = x.scratch[:0]")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, ", fieldTag, ")")
			outFile.P("x.scratch = ", outFile.SymAppendString(), "(x.scratch, v)")
			outFile.P("x.writer.Write(x.scratch)")
			outFile.P("}")

		case descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
			fieldTag := fmt.Sprintf("0x%x", (*field.Number<<3)|2)
			funcName := getSetterName(field)

			subType := getTypeName(outFile, field)
			subWriter := lowerCaseFirstLetter(subType + "Builder")
			subWriterType := capitalizeFirstLetter(subType + "Builder")
			outFile.P(funcPrefix, funcName+"(cb func(w *"+subWriterType, ")) {")
			outFile.P("x.buf.Reset()")
			outFile.P("x.", subWriter, ".writer = &x.buf")
			outFile.P("x.", subWriter, ".scratch = x.scratch")
			outFile.P("cb(&x.", subWriter, ")")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch[:0], ", fieldTag, ")")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, uint64(x.buf.Len()))")
			outFile.P("x.writer.Write(x.scratch)")
			outFile.P("x.writer.Write(x.buf.Bytes())")
			outFile.P("}")

		case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
			fieldTag := fmt.Sprintf("0x%x", (*field.Number<<3)|2)
			funcName := getSetterName(field)

			outFile.P(funcPrefix, funcName, "(cb func(b *bytes.Buffer)) {")
			outFile.P("x.buf.Reset()")
			outFile.P("cb(&x.buf)")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch[:0], ", fieldTag, ")")
			outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, uint64(x.buf.Len()))")
			outFile.P("x.writer.Write(x.scratch)")
			outFile.P("x.writer.Write(x.buf.Bytes())")

			outFile.P("}")

		default:
			return fmt.Errorf("unhandled type for field: %s", (*field).String())
		}
	}

	for _, m := range message.NestedType {
		if err := handleDescriptor(outFile, capitalizeFirstLetter(*message.Name)+"_", m); err != nil {
			return err
		}
	}
	return nil
}

func handleVarintField(outFile *FileContext, builderTypeName string, field *descriptorpb.FieldDescriptorProto) {
	fieldTag := fmt.Sprintf("0x%x", (uint32(*field.Number)<<3)|uint32(0))
	funcName := getSetterName(field)
	funcPrefix := "func(x *" + builderTypeName + ") "

	var argType string
	switch *field.Type {
	case descriptorpb.FieldDescriptorProto_TYPE_INT32:
		argType = "int32"
	case descriptorpb.FieldDescriptorProto_TYPE_INT64:
		argType = "int64"
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32:
		argType = "uint32"
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64:
		argType = "uint64"
	}

	outFile.P(funcPrefix, funcName, "(v ", argType, " ) {")
	outFile.P("x.scratch = x.scratch[:0]")
	outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, ", fieldTag, ")")
	outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, uint64(v))")
	outFile.P("x.writer.Write(x.scratch)")
	outFile.P("}")
}

func handleFixed64(outFile *FileContext, builderTypeName string, field *descriptorpb.FieldDescriptorProto) {

	var argType string
	uint64Convert := "uint64"
	appender := outFile.SymAppendFixed64()
	wireType := protowire.Fixed64Type
	switch *field.Type {
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		argType = "uint64"
	case descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		argType = "int64"
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		argType = "float64"
		uint64Convert = outFile.SymMathFloat64Bits()
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		wireType = protowire.Fixed32Type
		argType = "float32"
		uint64Convert = outFile.SymMathFloat32Bits()
		appender = outFile.SymAppendFixed32()
	case descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		wireType = protowire.Fixed32Type
		argType = "uint32"
		uint64Convert = "uint32"
		appender = outFile.SymAppendFixed32()
	case descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		wireType = protowire.Fixed32Type
		argType = "int32"
		uint64Convert = "uint32"
		appender = outFile.SymAppendFixed32()

	default:
		panic("here")
	}

	fieldTag := fmt.Sprintf("0x%x", (uint32(*field.Number)<<3)|uint32(wireType))
	funcName := getSetterName(field)
	funcPrefix := "func(x *" + builderTypeName + ") "

	outFile.P(funcPrefix, funcName, "(v ", argType, " ) {")
	outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch[:0], ", fieldTag, ")")
	outFile.P("x.scratch = ", appender, "(x.scratch, ", uint64Convert, "(v))")
	outFile.P("x.writer.Write(x.scratch)")
	outFile.P("}")
}

func handleSigned(outFile *FileContext, builderTypeName string, field *descriptorpb.FieldDescriptorProto) {
	fieldTag := fmt.Sprintf("0x%x", (uint32(*field.Number)<<3)|uint32(protowire.VarintType))
	funcName := getSetterName(field)
	funcPrefix := "func(x *" + builderTypeName + ") "

	outFile.SymAppendVarint()

	var argType string
	switch *field.Type {
	case descriptorpb.FieldDescriptorProto_TYPE_SINT32:
		argType = "int32"
	case descriptorpb.FieldDescriptorProto_TYPE_SINT64:
		argType = "int64"
	default:
		panic("here " + (*field).Type.String())
	}

	outFile.P(funcPrefix, funcName, "(v ", argType, " ) {")
	outFile.P("x.scratch = x.scratch[:0]")
	outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, ", fieldTag, ")")
	outFile.P("x.scratch = ", outFile.SymAppendVarint(), "(x.scratch, ", outFile.SymEncodeZigZag(), "(int64(v)))")
	outFile.P("x.writer.Write(x.scratch)")
	outFile.P("}")
}

func getSetterName(field *descriptorpb.FieldDescriptorProto) string {
	if field.Label != nil && *field.Label == descriptorpb.FieldDescriptorProto_LABEL_REPEATED {
		return "Add" + capitalizeFirstLetter(*field.Name)
	}
	return "Set" + capitalizeFirstLetter(*field.Name)
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

func (fc *FileContext) SymEncodeZigZag() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "EncodeZigZag",
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

func (fc *FileContext) SymAppendFixed64() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "AppendFixed64",
		GoImportPath: "google.golang.org/protobuf/encoding/protowire",
	})
}

func (fc *FileContext) SymAppendFixed32() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "AppendFixed32",
		GoImportPath: "google.golang.org/protobuf/encoding/protowire",
	})
}

func (fc *FileContext) SymMathFloat64Bits() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "Float64bits",
		GoImportPath: "math",
	})
}

func (fc *FileContext) SymMathFloat32Bits() string {
	return fc.generatedFile.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "Float32bits",
		GoImportPath: "math",
	})
}

func packageShortName(pkg string) string {
	parts := strings.Split(pkg, "/")
	return parts[len(parts)-1]
}

func capitalizeFirstLetter(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:len(s)]
}

func lowerCaseFirstLetter(s string) string {
	return strings.ToLower(s[0:1]) + s[1:len(s)]
}
