package codegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/PapaCharlie/go-restli/protocol"
	. "github.com/dave/jennifer/jen"
	"github.com/pkg/errors"
)

const (
	EncodingJson  = "encoding/json"
	Unmarshal     = "Unmarshal"
	UnmarshalJSON = "UnmarshalJSON"
	Marshal       = "Marshal"
	MarshalJSON   = "MarshalJSON"

	Codec                = "codec"
	RestLiEncode         = "RestLiEncode"
	RestLiDecode         = "RestLiDecode"
	RestLiCodec          = "RestLiCodec"
	RestLiUrlEncoder     = "RestLiUrlEncoder"
	RestLiReducedEncoder = "RestLiReducedEncoder"

	PopulateDefaultValues = "populateDefaultValues"
	ValidateUnionFields   = "validateUnionFields"

	NetHttp = "net/http"

	ProtocolPackage = "github.com/PapaCharlie/go-restli/protocol"
)

var (
	PdscDirectory string
	PackagePrefix string

	CommentWrapWidth = 120

	HeaderTemplate = template.Must(template.New("header").Parse(`DO NOT EDIT

Code automatically generated by go-restli
Source file: {{.SourceFilename}}`))
)

type CodeFile struct {
	SourceFilename string
	PackagePath    string
	Filename       string
	Code           *Statement
}

func NewCodeFile(filename string, packageSegments ...string) *CodeFile {
	return &CodeFile{
		PackagePath: filepath.Join(packageSegments...),
		Filename:    filename,
		Code:        Empty(),
	}
}

func (f *CodeFile) Write(outputDir string) (filename string, err error) {
	defer func() {
		e := recover()
		if e != nil {
			err = errors.Errorf("Could not generate model: %+v", e)
		}
	}()
	file := NewFilePath(f.PackagePath)

	header := bytes.NewBuffer(nil)
	err = HeaderTemplate.Execute(header, f)
	if err != nil {
		return "", err
	}
	file.HeaderComment(header.String())

	file.Add(f.Code)
	filename = filepath.Join(outputDir, f.PackagePath, f.Filename+".go")
	err = Write(filename, file)
	return filename, err
}

func (f *CodeFile) Identifier() string {
	return f.PackagePath + "." + f.Filename
}

func Write(filename string, file *File) error {
	b := bytes.NewBuffer(nil)
	if err := file.Render(b); err != nil {
		return errors.WithStack(err)
	}

	if err := os.MkdirAll(filepath.Dir(filename), os.ModePerm); err != nil {
		return errors.WithStack(err)
	}

	_ = os.Remove(filename)

	if _, err := os.Stat(filename); err == nil {
		if removeErr := os.Remove(filename); removeErr != nil {
			return errors.WithMessagef(removeErr, "Could not delete %s", filename)
		}
	} else {
		if !os.IsNotExist(err) {
			return errors.WithStack(err)
		}
	}

	if err := ioutil.WriteFile(filename, b.Bytes(), os.FileMode(0555)); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func AddWordWrappedComment(code *Statement, comment string) *Statement {
	if comment != "" {
		code.Comment(comment)
		return code
	} else {
		return code
	}

	// WIP: RestLi comments are not behaving quite as expected, so comments get added as is, without being wrapped
	for len(comment) > CommentWrapWidth {
		if newline := strings.Index(comment[:CommentWrapWidth], "\n"); newline != -1 {
			code.Comment(comment[:newline]).Line()
			comment = comment[newline+1:]
			continue
		}

		if index := strings.LastIndexFunc(comment[:CommentWrapWidth], unicode.IsSpace); index > 0 {
			code.Comment(comment[:index]).Line()
			comment = comment[index+1:]
		} else {
			break
		}
	}

	code.Comment(comment)

	return code
}

func ExportedIdentifier(identifier string) string {
	return strings.ToUpper(identifier[:1]) + identifier[1:]
}

func PrivateIdentifier(identifier string) string {
	return strings.ToLower(identifier[:1]) + identifier[1:]
}

func ReceiverName(typeName string) string {
	return PrivateIdentifier(typeName[:1])
}

func AddFuncOnReceiver(def *Statement, receiver, typeName, funcName string) *Statement {
	return def.Func().
		Params(Id(receiver).Op("*").Id(typeName)).
		Id(funcName)
}

func AddMarshalJSON(def *Statement, receiver, typeName string, f func(def *Group)) *Statement {
	return AddFuncOnReceiver(def, receiver, typeName, MarshalJSON).
		Params().
		Params(Id("data").Index().Byte(), Err().Error()).
		BlockFunc(f)
}

func AddUnmarshalJSON(def *Statement, receiver, typeName string, f func(def *Group)) *Statement {
	return AddFuncOnReceiver(def, receiver, typeName, UnmarshalJSON).
		Params(Id("data").Index().Byte()).
		Params(Err().Error()).
		BlockFunc(f)
}

func AddRestLiEncode(def *Statement, receiver, typeName string, f func(def *Group)) *Statement {
	return AddFuncOnReceiver(def, receiver, typeName, RestLiEncode).
		Params(Id(Codec).Qual(ProtocolPackage, RestLiCodec)).
		Params(Id("data").String(), Err().Error()).
		BlockFunc(f)
}

func AddRestLiDecode(def *Statement, receiver, typeName string, f func(def *Group)) *Statement {
	return AddFuncOnReceiver(def, receiver, typeName, RestLiDecode).
		Params(Id(Codec).Qual(ProtocolPackage, RestLiCodec), Id("data").String()).
		Params(Err().Error()).
		BlockFunc(f)
}

func AddStringer(def *Statement, receiver, typeName string, f func(def *Group)) *Statement {
	return AddFuncOnReceiver(def, receiver, typeName, "String").
		Params().
		String().
		BlockFunc(f)
}

func IfErrReturn(def *Group, results ...Code) *Group {
	def.If(Err().Op("!=").Nil()).Block(Return(results...))
	return def
}

func Bytes() *Statement {
	return Qual(ProtocolPackage, "Bytes")
}

type FieldTag struct {
	Json struct {
		Name     string
		Optional bool
	}
}

func (f *FieldTag) ToMap() map[string]string {
	tags := map[string]string{}
	if f.Json.Name != "" {
		tags["json"] = f.Json.Name
		if f.Json.Optional {
			tags["json"] += ",omitempty"
		}
	}

	return tags
}

func RestLiMethod(method protocol.RestLiMethod) *Statement {
	return Qual(ProtocolPackage, "Method_"+method.String())
}

func DeduplicateFiles(files []*CodeFile) []*CodeFile {
	idToFile := make(map[string]*CodeFile)

	renderCode := func(s *Statement) []byte {
		b := bytes.NewBuffer(nil)
		if err := s.Render(b); err != nil {
			log.Panicln(err)
		}
		return b.Bytes()
	}

	for _, file := range files {
		id := file.Identifier()
		if existingFile, ok := idToFile[id]; ok {
			existingCode := renderCode(existingFile.Code)
			code := renderCode(file.Code)
			if !bytes.Equal(existingCode, code) {
				log.Fatalf("Conflicting defitions of %s: %s\n\n-----------\n\n%s",
					id, string(existingCode), string(code))
			}
		} else {
			idToFile[id] = file
		}
	}

	identifiers := make([]string, 0, len(idToFile))
	for id := range idToFile {
		identifiers = append(identifiers, id)
	}
	sort.Strings(identifiers)

	uniqueCodeFiles := make([]*CodeFile, 0, len(idToFile))
	for _, id := range identifiers {
		uniqueCodeFiles = append(uniqueCodeFiles, idToFile[id])
	}

	return uniqueCodeFiles
}

func GenerateAllImportsFile(outputDir string, codeFiles []*CodeFile) {
	imports := make(map[string]bool)
	for _, code := range codeFiles {
		imports[code.PackagePath] = true
	}
	f := NewFile("main")
	for p := range imports {
		f.Anon(p)
	}
	f.Func().Id("TestAllImports").Params(Op("*").Qual("testing", "T")).Block()

	err := Write(filepath.Join(outputDir, PackagePrefix, "all_imports_test.go"), f)
	if err != nil {
		log.Panicf("Could not write all imports file: %+v", err)
	}
}

type MalformedPdscFileError struct {
	Filename    string
	SyntaxError *json.SyntaxError
}

func (m *MalformedPdscFileError) Error() string {
	return fmt.Sprintf("malformed .pdsc file %s at %d: %+v", m.Filename, m.SyntaxError.Offset, m.SyntaxError)
}

func ReadJSONFromFile(filename string, s interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.WithStack(err)
	}

	// Apparently the rest.li spec allows C-style comments in pdsc files. We can gracefully handle SyntaxErrors caused
	// by such comments by removing the comment and retrying the file
	for {
		err = json.Unmarshal(data, s)

		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			// Special case, the zero-valued Offset is an unrecoverable SyntaxError
			if syntaxErr.Offset == 0 {
				return syntaxErr
			}
			// The SyntaxError gives the offset+1 otherwise, likely to indicate the difference between the zero-value
			// and an actual offset?
			offset := int(syntaxErr.Offset - 1)

			var endOfComment string
			if data[offset] == '/' {
				switch data[offset+1] {
				case '/':
					endOfComment = "\n"
				case '*':
					endOfComment = "*/"
				default:
					return errors.WithStack(&MalformedPdscFileError{Filename: filename, SyntaxError: syntaxErr})
				}
				eol := bytes.Index(data[offset:], []byte(endOfComment))
				if eol == -1 {
					return errors.WithStack(&MalformedPdscFileError{Filename: filename, SyntaxError: syntaxErr})
				}
				data = append(data[:offset], data[offset+eol+len(endOfComment):]...)
				continue
			}
		}

		return errors.WithStack(err)
	}
}
