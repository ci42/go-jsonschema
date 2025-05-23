package generator

import (
	"fmt"
	"math"
	"strings"

	"github.com/atombender/go-jsonschema/pkg/codegen"
)

const (
	formatYAML  = "yaml"
	YAMLPackage = "gopkg.in/yaml.v3"
)

type yamlFormatter struct{}

func (yf *yamlFormatter) generate(
	output *output,
	declType codegen.TypeDecl,
	validators []validator,
) func(*codegen.Emitter) error {
	var (
		beforeValidators []validator
		afterValidators  []validator
	)

	forceBefore := false

	for _, v := range validators {
		desc := v.desc()
		if desc.beforeJSONUnmarshal {
			beforeValidators = append(beforeValidators, v)
		} else {
			afterValidators = append(afterValidators, v)
			forceBefore = forceBefore || desc.requiresRawAfter
		}
	}

	return func(out *codegen.Emitter) error {
		out.Commentf("Unmarshal%s implements %s.Unmarshaler.", strings.ToUpper(formatYAML), formatYAML)
		out.Printlnf("func (j *%s) Unmarshal%s(value *yaml.Node) error {", declType.Name, strings.ToUpper(formatYAML))
		out.Indent(1)

		if forceBefore || len(beforeValidators) != 0 {
			out.Printlnf("var %s map[string]interface{}", varNameRawMap)
			out.Printlnf("if err := value.Decode(&%s); err != nil { return err }", varNameRawMap)
		}

		for _, v := range beforeValidators {
			if err := v.generate(out, "yaml"); err != nil {
				return fmt.Errorf("cannot generate before validators: %w", err)
			}
		}

		tp := typePlain

		if tp == declType.Name {
			for i := 0; !output.isUniqueTypeName(tp) && i < math.MaxInt; i++ {
				tp = fmt.Sprintf("%s_%d", typePlain, i)
			}
		}

		out.Printlnf("type %s %s", tp, declType.Name)
		out.Printlnf("var %s %s", varNamePlainStruct, tp)
		out.Printlnf("if err := value.Decode(&%s); err != nil { return err }", varNamePlainStruct)

		for _, v := range afterValidators {
			if err := v.generate(out, "yaml"); err != nil {
				return fmt.Errorf("cannot generate after validators: %w", err)
			}
		}

		if structType, ok := declType.Type.(*codegen.StructType); ok {
			for _, f := range structType.Fields {
				if f.Name == "AdditionalProperties" {
					out.Printlnf("st := reflect.TypeOf(Plain{})")
					out.Printlnf("for i := range st.NumField() {")
					out.Indent(1)
					out.Printlnf("delete(raw, st.Field(i).Name)")
					out.Printlnf("delete(raw, strings.Split(st.Field(i).Tag.Get(\"json\"), \",\")[0])")
					out.Indent(-1)
					out.Printlnf("}")
					out.Printlnf("if err := mapstructure.Decode(raw, &plain.AdditionalProperties); err != nil {")
					out.Indent(1)
					out.Printlnf("return err")
					out.Indent(-1)
					out.Printlnf("}")

					break
				}
			}
		}

		out.Printlnf("*j = %s(%s)", declType.Name, varNamePlainStruct)
		out.Printlnf("return nil")
		out.Indent(-1)
		out.Printlnf("}")

		return nil
	}
}

func (yf *yamlFormatter) enumMarshal(declType codegen.TypeDecl) func(*codegen.Emitter) error {
	return func(out *codegen.Emitter) error {
		out.Commentf("Marshal%s implements %s.Marshal.", strings.ToUpper(formatYAML), formatYAML)
		out.Printlnf("func (j *%s) Marshal%s() (interface{}, error) {", declType.Name, strings.ToUpper(formatYAML))
		out.Indent(1)
		out.Printlnf("return %s.Marshal(j.Value)", formatYAML)
		out.Indent(-1)
		out.Printlnf("}")

		return nil
	}
}

func (yf *yamlFormatter) enumUnmarshal(
	declType codegen.TypeDecl,
	enumType codegen.Type,
	valueConstant *codegen.Var,
	wrapInStruct bool,
) func(*codegen.Emitter) error {
	return func(out *codegen.Emitter) error {
		out.Commentf("Unmarshal%s implements %s.Unmarshaler.", strings.ToUpper(formatYAML), formatYAML)
		out.Printlnf("func (j *%s) Unmarshal%s(value *yaml.Node) error {", declType.Name, strings.ToUpper(formatYAML))
		out.Indent(1)
		out.Printf("var v ")

		if err := enumType.Generate(out); err != nil {
			return fmt.Errorf("cannot unmarshal enum content: %w", err)
		}

		out.Newline()

		varName := "v"
		if wrapInStruct {
			varName += ".Value"
		}

		out.Printlnf("if err := value.Decode(&%s); err != nil { return err }", varName)
		out.Printlnf("var ok bool")
		out.Printlnf("for _, expected := range %s {", valueConstant.Name)
		out.Printlnf("if reflect.DeepEqual(%s, expected) { ok = true; break }", varName)
		out.Printlnf("}")
		out.Printlnf("if !ok {")
		out.Printlnf(`return fmt.Errorf("invalid value (expected one of %%#v): %%#v", %s, %s)`, valueConstant.Name, varName)
		out.Printlnf("}")
		out.Printlnf(`*j = %s(v)`, declType.Name)
		out.Printlnf(`return nil`)
		out.Indent(-1)
		out.Printlnf("}")

		return nil
	}
}

func (yf *yamlFormatter) addImport(out *codegen.File, declType codegen.TypeDecl) {
	out.Package.AddImport(YAMLPackage, "yaml")

	if structType, ok := declType.Type.(*codegen.StructType); ok {
		for _, f := range structType.Fields {
			if f.Name == additionalProperties {
				out.Package.AddImport("reflect", "")
				out.Package.AddImport("strings", "")
				out.Package.AddImport("github.com/go-viper/mapstructure/v2", "")

				return
			}
		}
	}
}
