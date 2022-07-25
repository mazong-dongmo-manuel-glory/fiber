package binder

import (
	"reflect"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3/internal/schema"
	"github.com/gofiber/fiber/v3/utils"
	"github.com/valyala/bytebufferpool"
)

// Some constants for BodyParser, QueryParser and ReqHeaderParser.
const (
	queryTag     = "query"
	reqHeaderTag = "reqHeader"
	bodyTag      = "form"
)

// ParserConfig form decoder config for SetParserDecoder
type ParserConfig struct {
	IgnoreUnknownKeys bool
	SetAliasTag       string
	ParserType        []ParserType
	ZeroEmpty         bool
}

// ParserType require two element, type and converter for register.
// Use ParserType with BodyParser for parsing custom type in form data.
type ParserType struct {
	Customtype any
	Converter  func(string) reflect.Value
}

// decoderPool helps to improve BodyParser's, QueryParser's and ReqHeaderParser's performance
var decoderPool = &sync.Pool{New: func() any {
	return decoderBuilder(ParserConfig{
		IgnoreUnknownKeys: true,
		ZeroEmpty:         true,
	})
}}

// SetParserDecoder allow globally change the option of form decoder, update decoderPool
func SetParserDecoder(parserConfig ParserConfig) {
	decoderPool = &sync.Pool{New: func() any {
		return decoderBuilder(parserConfig)
	}}
}

func decoderBuilder(parserConfig ParserConfig) any {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(parserConfig.IgnoreUnknownKeys)
	if parserConfig.SetAliasTag != "" {
		decoder.SetAliasTag(parserConfig.SetAliasTag)
	}
	for _, v := range parserConfig.ParserType {
		decoder.RegisterConverter(reflect.ValueOf(v.Customtype).Interface(), v.Converter)
	}
	decoder.ZeroEmpty(parserConfig.ZeroEmpty)
	return decoder
}

func parseToStruct(aliasTag string, out any, data map[string][]string) error {
	ptrVal := reflect.ValueOf(out)

	if ptrVal.Kind() == reflect.Map && ptrVal.Type().Key().Kind() == reflect.String {
		//
	}

	// Get decoder from pool
	schemaDecoder := decoderPool.Get().(*schema.Decoder)
	defer decoderPool.Put(schemaDecoder)

	// Set alias tag
	schemaDecoder.SetAliasTag(aliasTag)

	return schemaDecoder.Decode(out, data)
}

func parseParamSquareBrackets(k string) (string, error) {
	bb := bytebufferpool.Get()
	defer bytebufferpool.Put(bb)

	kbytes := []byte(k)

	for i, b := range kbytes {

		if b == '[' && kbytes[i+1] != ']' {
			if err := bb.WriteByte('.'); err != nil {
				return "", err
			}
		}

		if b == '[' || b == ']' {
			continue
		}

		if err := bb.WriteByte(b); err != nil {
			return "", err
		}
	}

	return bb.String(), nil
}

func equalFieldType(out any, kind reflect.Kind, key string) bool {
	// Get type of interface
	outTyp := reflect.TypeOf(out).Elem()
	key = utils.ToLower(key)
	// Must be a struct to match a field
	if outTyp.Kind() != reflect.Struct {
		return false
	}
	// Copy interface to an value to be used
	outVal := reflect.ValueOf(out).Elem()
	// Loop over each field
	for i := 0; i < outTyp.NumField(); i++ {
		// Get field value data
		structField := outVal.Field(i)
		// Can this field be changed?
		if !structField.CanSet() {
			continue
		}
		// Get field key data
		typeField := outTyp.Field(i)
		// Get type of field key
		structFieldKind := structField.Kind()
		// Does the field type equals input?
		if structFieldKind != kind {
			continue
		}
		// Get tag from field if exist
		inputFieldName := typeField.Tag.Get(queryTag)
		if inputFieldName == "" {
			inputFieldName = typeField.Name
		} else {
			inputFieldName = strings.Split(inputFieldName, ",")[0]
		}
		// Compare field/tag with provided key
		if utils.ToLower(inputFieldName) == key {
			return true
		}
	}
	return false
}

func FilterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}
