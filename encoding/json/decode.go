/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2022 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package json

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/sema"
)

// A Decoder decodes JSON-encoded representations of Cadence values.
type Decoder struct {
	dec   *json.Decoder
	gauge common.MemoryGauge
}

// Decode returns a Cadence value decoded from its JSON-encoded representation.
//
// This function returns an error if the bytes represent JSON that is malformed
// or does not conform to the JSON Cadence specification.
func Decode(gauge common.MemoryGauge, b []byte) (cadence.Value, error) {
	r := bytes.NewReader(b)
	dec := NewDecoder(gauge, r)

	v, err := dec.Decode()
	if err != nil {
		return nil, err
	}

	return v, nil
}

// NewDecoder initializes a Decoder that will decode JSON-encoded bytes from the
// given io.Reader.
func NewDecoder(gauge common.MemoryGauge, r io.Reader) *Decoder {
	return &Decoder{
		dec:   json.NewDecoder(r),
		gauge: gauge,
	}
}

// Decode reads JSON-encoded bytes from the io.Reader and decodes them to a
// Cadence value.
//
// This function returns an error if the bytes represent JSON that is malformed
// or does not conform to the JSON Cadence specification.
func (d *Decoder) Decode() (value cadence.Value, err error) {
	jsonMap := make(map[string]interface{})

	err = d.dec.Decode(&jsonMap)
	if err != nil {
		return nil, fmt.Errorf("json-cdc: failed to decode valid JSON structure: %w", err)
	}

	// capture panics that occur during decoding
	defer func() {
		if r := recover(); r != nil {
			panicErr, isError := r.(error)
			if !isError {
				panic(r)
			}

			err = fmt.Errorf("failed to decode value: %w", panicErr)
		}
	}()

	value = d.decodeJSON(jsonMap)
	return value, nil
}

const (
	typeKey         = "type"
	kindKey         = "kind"
	valueKey        = "value"
	keyKey          = "key"
	nameKey         = "name"
	fieldsKey       = "fields"
	initializersKey = "initializers"
	idKey           = "id"
	targetPathKey   = "targetPath"
	borrowTypeKey   = "borrowType"
	domainKey       = "domain"
	identifierKey   = "identifier"
	staticTypeKey   = "staticType"
	addressKey      = "address"
	pathKey         = "path"
	authorizedKey   = "authorized"
	sizeKey         = "size"
	typeIDKey       = "typeID"
	restrictionsKey = "restrictions"
	labelKey        = "label"
	parametersKey   = "parameters"
	returnKey       = "return"
)

var ErrInvalidJSONCadence = errors.New("invalid JSON Cadence structure")

func (d *Decoder) decodeJSON(v interface{}) cadence.Value {
	obj := toObject(v)

	typeStr := obj.GetString(typeKey)

	// void is a special case, does not have "value" field
	if typeStr == voidTypeStr {
		return d.decodeVoid(obj)
	}

	// object should only contain two keys: "type", "value"
	if len(obj) != 2 {
		panic(ErrInvalidJSONCadence)
	}

	valueJSON := obj.Get(valueKey)

	switch typeStr {
	case optionalTypeStr:
		return d.decodeOptional(valueJSON)
	case boolTypeStr:
		return d.decodeBool(valueJSON)
	case characterTypeStr:
		return d.decodeCharacter(valueJSON)
	case stringTypeStr:
		return d.decodeString(valueJSON)
	case addressTypeStr:
		return d.decodeAddress(valueJSON)
	case intTypeStr:
		return d.decodeInt(valueJSON)
	case int8TypeStr:
		return d.decodeInt8(valueJSON)
	case int16TypeStr:
		return d.decodeInt16(valueJSON)
	case int32TypeStr:
		return d.decodeInt32(valueJSON)
	case int64TypeStr:
		return d.decodeInt64(valueJSON)
	case int128TypeStr:
		return d.decodeInt128(valueJSON)
	case int256TypeStr:
		return d.decodeInt256(valueJSON)
	case uintTypeStr:
		return d.decodeUInt(valueJSON)
	case uint8TypeStr:
		return d.decodeUInt8(valueJSON)
	case uint16TypeStr:
		return d.decodeUInt16(valueJSON)
	case uint32TypeStr:
		return d.decodeUInt32(valueJSON)
	case uint64TypeStr:
		return d.decodeUInt64(valueJSON)
	case uint128TypeStr:
		return d.decodeUInt128(valueJSON)
	case uint256TypeStr:
		return d.decodeUInt256(valueJSON)
	case word8TypeStr:
		return d.decodeWord8(valueJSON)
	case word16TypeStr:
		return d.decodeWord16(valueJSON)
	case word32TypeStr:
		return d.decodeWord32(valueJSON)
	case word64TypeStr:
		return d.decodeWord64(valueJSON)
	case fix64TypeStr:
		return d.decodeFix64(valueJSON)
	case ufix64TypeStr:
		return d.decodeUFix64(valueJSON)
	case arrayTypeStr:
		return d.decodeArray(valueJSON)
	case dictionaryTypeStr:
		return d.decodeDictionary(valueJSON)
	case resourceTypeStr:
		return d.decodeResource(valueJSON)
	case structTypeStr:
		return d.decodeStruct(valueJSON)
	case eventTypeStr:
		return d.decodeEvent(valueJSON)
	case contractTypeStr:
		return d.decodeContract(valueJSON)
	case linkTypeStr:
		return d.decodeLink(valueJSON)
	case pathTypeStr:
		return d.decodePath(valueJSON)
	case typeTypeStr:
		return d.decodeTypeValue(valueJSON)
	case capabilityTypeStr:
		return d.decodeCapability(valueJSON)
	case enumTypeStr:
		return d.decodeEnum(valueJSON)
	}

	panic(ErrInvalidJSONCadence)
}

func (d *Decoder) decodeVoid(m map[string]interface{}) cadence.Void {
	// object should not contain fields other than "type"
	if len(m) != 1 {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewVoid(d.gauge)
}

func (d *Decoder) decodeOptional(valueJSON interface{}) cadence.Optional {
	if valueJSON == nil {
		return cadence.NewOptional(d.gauge, nil)
	}

	return cadence.NewOptional(d.gauge, d.decodeJSON(valueJSON))
}

func (d *Decoder) decodeBool(valueJSON interface{}) cadence.Bool {
	return cadence.NewBool(d.gauge, toBool(valueJSON))
}

func (d *Decoder) decodeCharacter(valueJSON interface{}) cadence.Character {
	asString := toString(valueJSON)
	char, err := cadence.NewCharacter(
		d.gauge,
		common.NewCadenceCharacterMemoryUsage(len(asString)),
		func() string {
			return asString
		})
	if err != nil {
		panic(err)
	}
	return char
}

func (d *Decoder) decodeString(valueJSON interface{}) cadence.String {
	asString := toString(valueJSON)
	str, err := cadence.NewString(
		d.gauge,
		common.NewCadenceStringMemoryUsage(len(asString)),
		func() string {
			return asString
		},
	)
	if err != nil {
		panic(err)
	}
	return str
}

func (d *Decoder) decodeAddress(valueJSON interface{}) cadence.Address {
	v := toString(valueJSON)

	// must include 0x prefix
	if v[:2] != "0x" {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	b, err := hex.DecodeString(v[2:])
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.BytesToUnmeteredAddress(b)
}

func (d *Decoder) decodeBigInt(valueJSON interface{}) *big.Int {
	v := toString(valueJSON)

	i := new(big.Int)
	i, ok := i.SetString(v, 10)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return i
}

func (d *Decoder) decodeInt(valueJSON interface{}) cadence.Int {
	bigInt := d.decodeBigInt(valueJSON)
	return cadence.NewIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	)
}

func (d *Decoder) decodeInt8(valueJSON interface{}) cadence.Int8 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt8(d.gauge, int8(i))
}

func (d *Decoder) decodeInt16(valueJSON interface{}) cadence.Int16 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt16(d.gauge, int16(i))
}

func (d *Decoder) decodeInt32(valueJSON interface{}) cadence.Int32 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt32(d.gauge, int32(i))
}

func (d *Decoder) decodeInt64(valueJSON interface{}) cadence.Int64 {
	v := toString(valueJSON)

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewInt64(d.gauge, i)
}

func (d *Decoder) decodeInt128(valueJSON interface{}) cadence.Int128 {
	bigInt := d.decodeBigInt(valueJSON)

	value, err := cadence.NewInt128FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeInt256(valueJSON interface{}) cadence.Int256 {
	bigInt := d.decodeBigInt(valueJSON)

	value, err := cadence.NewInt256FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeUInt(valueJSON interface{}) cadence.UInt {
	bigInt := d.decodeBigInt(valueJSON)
	value, err := cadence.NewUIntFromBig(
		d.gauge,
		common.NewCadenceIntMemoryUsage(
			common.BigIntByteLength(bigInt),
		),
		func() *big.Int {
			return bigInt
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeUInt8(valueJSON interface{}) cadence.UInt8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt8(d.gauge, uint8(i))
}

func (d *Decoder) decodeUInt16(valueJSON interface{}) cadence.UInt16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt16(d.gauge, uint16(i))
}

func (d *Decoder) decodeUInt32(valueJSON interface{}) cadence.UInt32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt32(d.gauge, uint32(i))
}

func (d *Decoder) decodeUInt64(valueJSON interface{}) cadence.UInt64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewUInt64(d.gauge, i)
}

func (d *Decoder) decodeUInt128(valueJSON interface{}) cadence.UInt128 {
	bigInt := d.decodeBigInt(valueJSON)
	value, err := cadence.NewUInt128FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeUInt256(valueJSON interface{}) cadence.UInt256 {
	bigInt := d.decodeBigInt(valueJSON)
	value, err := cadence.NewUInt256FromBig(
		d.gauge,
		func() *big.Int {
			return bigInt
		},
	)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeWord8(valueJSON interface{}) cadence.Word8 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord8(d.gauge, uint8(i))
}

func (d *Decoder) decodeWord16(valueJSON interface{}) cadence.Word16 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 16)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord16(d.gauge, uint16(i))
}

func (d *Decoder) decodeWord32(valueJSON interface{}) cadence.Word32 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord32(d.gauge, uint32(i))
}

func (d *Decoder) decodeWord64(valueJSON interface{}) cadence.Word64 {
	v := toString(valueJSON)

	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewWord64(d.gauge, i)
}

func (d *Decoder) decodeFix64(valueJSON interface{}) cadence.Fix64 {
	v, err := cadence.NewFix64(d.gauge, func() (int64, error) {
		return cadence.ParseFix64(toString(valueJSON))
	})
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return v
}

func (d *Decoder) decodeUFix64(valueJSON interface{}) cadence.UFix64 {
	v, err := cadence.NewUFix64(d.gauge, func() (uint64, error) {
		return cadence.ParseUFix64(toString(valueJSON))
	})
	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return v
}

func (d *Decoder) decodeArray(valueJSON interface{}) cadence.Array {
	v := toSlice(valueJSON)

	value, err := cadence.NewArray(
		d.gauge,
		len(v),
		func() ([]cadence.Value, error) {
			values := make([]cadence.Value, len(v))
			for i, val := range v {
				values[i] = d.decodeJSON(val)
			}
			return values, nil
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}
	return value
}

func (d *Decoder) decodeDictionary(valueJSON interface{}) cadence.Dictionary {
	v := toSlice(valueJSON)

	value, err := cadence.NewDictionary(
		d.gauge,
		len(v),
		func() ([]cadence.KeyValuePair, error) {
			pairs := make([]cadence.KeyValuePair, len(v))

			for i, val := range v {
				pairs[i] = d.decodeKeyValuePair(val)
			}

			return pairs, nil
		},
	)

	if err != nil {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return value
}

func (d *Decoder) decodeKeyValuePair(valueJSON interface{}) cadence.KeyValuePair {
	obj := toObject(valueJSON)

	key := obj.GetValue(d, keyKey)
	value := obj.GetValue(d, valueKey)

	return cadence.NewKeyValuePair(
		d.gauge,
		key,
		value,
	)
}

type composite struct {
	location            common.Location
	qualifiedIdentifier string
	fieldValues         []cadence.Value
	fieldTypes          []cadence.Field
}

func (d *Decoder) decodeComposite(valueJSON interface{}) composite {
	obj := toObject(valueJSON)

	typeID := obj.GetString(idKey)
	location, qualifiedIdentifier, err := common.DecodeTypeID(d.gauge, typeID)

	if err != nil ||
		location == nil && sema.NativeCompositeTypes[typeID] == nil {

		// If the location is nil, and there is no native composite type with this ID, then its an invalid type.
		// Note: This is moved out from the common.DecodeTypeID() to avoid the circular dependency.
		panic(fmt.Errorf("%s. invalid type ID: `%s`", ErrInvalidJSONCadence, typeID))
	}

	fields := obj.GetSlice(fieldsKey)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: uint64(len(fields)),
	})

	fieldValues := make([]cadence.Value, len(fields))
	fieldTypes := make([]cadence.Field, len(fields))

	for i, field := range fields {
		value, fieldType := d.decodeCompositeField(field)

		fieldValues[i] = value
		fieldTypes[i] = fieldType
	}

	return composite{
		location:            location,
		qualifiedIdentifier: qualifiedIdentifier,
		fieldValues:         fieldValues,
		fieldTypes:          fieldTypes,
	}
}

func (d *Decoder) decodeCompositeField(valueJSON interface{}) (cadence.Value, cadence.Field) {
	obj := toObject(valueJSON)

	name := obj.GetString(nameKey)
	value := obj.GetValue(d, valueKey)

	// Unmetered because decodeCompositeField is metered in decodeComposite and called nowhere else
	field := cadence.NewUnmeteredField(name, value.Type(d.gauge))

	return value, field
}

func (d *Decoder) decodeStruct(valueJSON interface{}) cadence.Struct {
	comp := d.decodeComposite(valueJSON)

	structure, err := cadence.NewStruct(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return structure.WithType(cadence.NewStructType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeResource(valueJSON interface{}) cadence.Resource {
	comp := d.decodeComposite(valueJSON)

	resource, err := cadence.NewResource(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}
	return resource.WithType(cadence.NewResourceType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeEvent(valueJSON interface{}) cadence.Event {
	comp := d.decodeComposite(valueJSON)

	event, err := cadence.NewEvent(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return event.WithType(cadence.NewEventType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeContract(valueJSON interface{}) cadence.Contract {
	comp := d.decodeComposite(valueJSON)

	contract, err := cadence.NewContract(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return contract.WithType(cadence.NewContractType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeEnum(valueJSON interface{}) cadence.Enum {
	comp := d.decodeComposite(valueJSON)

	enum, err := cadence.NewEnum(
		d.gauge,
		len(comp.fieldValues),
		func() ([]cadence.Value, error) {
			return comp.fieldValues, nil
		},
	)

	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	return enum.WithType(cadence.NewEnumType(
		d.gauge,
		comp.location,
		comp.qualifiedIdentifier,
		nil,
		comp.fieldTypes,
		nil,
	))
}

func (d *Decoder) decodeLink(valueJSON interface{}) cadence.Link {
	obj := toObject(valueJSON)

	targetPath, ok := d.decodeJSON(obj.Get(targetPathKey)).(cadence.Path)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	borrowType := obj.GetString(borrowTypeKey)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// no need to add 1 to account for empty string: string is metered in Link struct
		Amount: uint64(len(borrowType)),
	})

	return cadence.NewLink(
		d.gauge,
		targetPath,
		borrowType,
	)
}

func (d *Decoder) decodePath(valueJSON interface{}) cadence.Path {
	obj := toObject(valueJSON)

	domain := obj.GetString(domainKey)

	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// no need to add 1 to account for empty string: string is metered in Path struct
		Amount: uint64(len(domain)),
	})

	identifier := obj.GetString(identifierKey)
	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind: common.MemoryKindRawString,
		// no need to add 1 to account for empty string: string is metered in Path struct
		Amount: uint64(len(identifier)),
	})

	return cadence.NewPath(
		d.gauge,
		domain,
		identifier,
	)
}

func (d *Decoder) decodeParamType(valueJSON interface{}) cadence.Parameter {
	obj := toObject(valueJSON)
	// Unmetered because decodeParamType is metered in decodeParamTypes and called nowhere else
	return cadence.NewUnmeteredParameter(
		toString(obj.Get(labelKey)),
		toString(obj.Get(idKey)),
		d.decodeType(obj.Get(typeKey)),
	)
}

func (d *Decoder) decodeParamTypes(params []interface{}) []cadence.Parameter {
	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceParameter,
		Amount: uint64(len(params)),
	})
	parameters := make([]cadence.Parameter, 0, len(params))

	for _, param := range params {
		parameters = append(parameters, d.decodeParamType(param))
	}

	return parameters
}

func (d *Decoder) decodeFieldTypes(fs []interface{}) []cadence.Field {
	common.UseMemory(d.gauge, common.MemoryUsage{
		Kind:   common.MemoryKindCadenceField,
		Amount: uint64(len(fs)),
	})

	fields := make([]cadence.Field, 0, len(fs))

	for _, field := range fs {
		fields = append(fields, d.decodeFieldType(field))
	}

	return fields
}

func (d *Decoder) decodeFieldType(valueJSON interface{}) cadence.Field {
	obj := toObject(valueJSON)
	// Unmetered because decodeFieldType is metered in decodeFieldTypes and called nowhere else
	return cadence.NewUnmeteredField(
		toString(obj.Get(idKey)),
		d.decodeType(obj.Get(typeKey)),
	)
}

func (d *Decoder) decodeFunctionType(returnValue, parametersValue, id interface{}) cadence.Type {
	parameters := d.decodeParamTypes(toSlice(parametersValue))
	returnType := d.decodeType(returnValue)

	return cadence.NewFunctionType(
		d.gauge,
		"",
		parameters,
		returnType,
	).WithID(toString(id))
}

func (d *Decoder) decodeNominalType(obj jsonObject, kind, typeID string, fs, initializers []interface{}) cadence.Type {
	fields := d.decodeFieldTypes(fs)

	// Unmetered because this is created as an array of nil arrays, not Parameter structs
	inits := make([][]cadence.Parameter, 0, len(initializers))
	for _, params := range initializers {
		inits = append(inits, d.decodeParamTypes(toSlice(params)))
	}

	location, id, err := common.DecodeTypeID(d.gauge, typeID)
	if err != nil {
		panic(ErrInvalidJSONCadence)
	}

	switch kind {
	case "Struct":
		return cadence.NewStructType(
			d.gauge,
			location,
			id,
			fields,
			inits,
		)
	case "Resource":
		return cadence.NewResourceType(
			d.gauge,
			location,
			id,
			fields,
			inits,
		)
	case "Event":
		return cadence.NewEventType(
			d.gauge,
			location,
			id,
			fields,
			inits[0],
		)
	case "Contract":
		return cadence.NewContractType(
			d.gauge,
			location,
			id,
			fields,
			inits,
		)
	case "StructInterface":
		return cadence.NewStructInterfaceType(
			d.gauge,
			location,
			id,
			fields,
			inits,
		)
	case "ResourceInterface":
		return cadence.NewResourceInterfaceType(
			d.gauge,
			location,
			id,
			fields,
			inits,
		)
	case "ContractInterface":
		return cadence.NewContractInterfaceType(
			d.gauge,
			location,
			id,
			fields,
			inits,
		)
	case "Enum":
		return cadence.NewEnumType(
			d.gauge,
			location,
			id,
			d.decodeType(obj.Get(typeKey)),
			fields,
			inits,
		)
	}

	panic(ErrInvalidJSONCadence)
}

func (d *Decoder) decodeRestrictedType(
	typeValue interface{},
	restrictionsValue []interface{},
	typeIDValue string,
) cadence.Type {
	typ := d.decodeType(typeValue)
	restrictions := make([]cadence.Type, 0, len(restrictionsValue))
	for _, restriction := range restrictionsValue {
		restrictions = append(restrictions, d.decodeType(restriction))
	}

	return cadence.NewRestrictedType(
		d.gauge,
		"",
		typ,
		restrictions,
	).WithID(typeIDValue)
}

func (d *Decoder) decodeType(valueJSON interface{}) cadence.Type {
	if valueJSON == "" {
		return nil
	}
	obj := toObject(valueJSON)
	kindValue := toString(obj.Get(kindKey))

	switch kindValue {
	case "Function":
		returnValue := obj.Get(returnKey)
		parametersValue := obj.Get(parametersKey)
		idValue := obj.Get(typeIDKey)
		return d.decodeFunctionType(returnValue, parametersValue, idValue)
	case "Restriction":
		restrictionsValue := obj.Get(restrictionsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		typeValue := obj.Get(typeKey)
		return d.decodeRestrictedType(typeValue, toSlice(restrictionsValue), typeIDValue)
	case "Optional":
		return cadence.NewOptionalType(
			d.gauge,
			d.decodeType(obj.Get(typeKey)),
		)
	case "VariableSizedArray":
		return cadence.NewVariableSizedArrayType(
			d.gauge,
			d.decodeType(obj.Get(typeKey)),
		)
	case "Capability":
		return cadence.NewCapabilityType(
			d.gauge,
			d.decodeType(obj.Get(typeKey)),
		)
	case "Dictionary":
		return cadence.NewDictionaryType(
			d.gauge,
			d.decodeType(obj.Get(keyKey)),
			d.decodeType(obj.Get(valueKey)),
		)
	case "ConstantSizedArray":
		size := toUInt(obj.Get(sizeKey))
		return cadence.NewConstantSizedArrayType(
			d.gauge,
			size,
			d.decodeType(obj.Get(typeKey)),
		)
	case "Reference":
		auth := toBool(obj.Get(authorizedKey))
		return cadence.NewReferenceType(
			d.gauge,
			auth,
			d.decodeType(obj.Get(typeKey)),
		)
	case "Any":
		return cadence.NewAnyType(d.gauge)
	case "AnyStruct":
		return cadence.NewAnyStructType(d.gauge)
	case "AnyResource":
		return cadence.NewAnyResourceType(d.gauge)
	case "Type":
		return cadence.NewMetaType(d.gauge)
	case "Void":
		return cadence.NewVoidType(d.gauge)
	case "Never":
		return cadence.NewNeverType(d.gauge)
	case "Bool":
		return cadence.NewBoolType(d.gauge)
	case "String":
		return cadence.NewStringType(d.gauge)
	case "Character":
		return cadence.NewCharacterType(d.gauge)
	case "Bytes":
		return cadence.NewBytesType(d.gauge)
	case "Address":
		return cadence.NewAddressType(d.gauge)
	case "Number":
		return cadence.NewNumberType(d.gauge)
	case "SignedNumber":
		return cadence.NewSignedNumberType(d.gauge)
	case "Integer":
		return cadence.NewIntegerType(d.gauge)
	case "SignedInteger":
		return cadence.NewSignedIntegerType(d.gauge)
	case "FixedPoint":
		return cadence.NewFixedPointType(d.gauge)
	case "SignedFixedPoint":
		return cadence.NewSignedFixedPointType(d.gauge)
	case "Int":
		return cadence.NewIntType(d.gauge)
	case "Int8":
		return cadence.NewInt8Type(d.gauge)
	case "Int16":
		return cadence.NewInt16Type(d.gauge)
	case "Int32":
		return cadence.NewInt32Type(d.gauge)
	case "Int64":
		return cadence.NewInt64Type(d.gauge)
	case "Int128":
		return cadence.NewInt128Type(d.gauge)
	case "Int256":
		return cadence.NewInt256Type(d.gauge)
	case "UInt":
		return cadence.NewUIntType(d.gauge)
	case "UInt8":
		return cadence.NewUInt8Type(d.gauge)
	case "UInt16":
		return cadence.NewUInt16Type(d.gauge)
	case "UInt32":
		return cadence.NewUInt32Type(d.gauge)
	case "UInt64":
		return cadence.NewUInt64Type(d.gauge)
	case "UInt128":
		return cadence.NewUInt128Type(d.gauge)
	case "UInt256":
		return cadence.NewUInt256Type(d.gauge)
	case "Word8":
		return cadence.NewWord8Type(d.gauge)
	case "Word16":
		return cadence.NewWord16Type(d.gauge)
	case "Word32":
		return cadence.NewWord32Type(d.gauge)
	case "Word64":
		return cadence.NewWord64Type(d.gauge)
	case "Fix64":
		return cadence.NewFix64Type(d.gauge)
	case "UFix64":
		return cadence.NewUFix64Type(d.gauge)
	case "Path":
		return cadence.NewPathType(d.gauge)
	case "CapabilityPath":
		return cadence.NewCapabilityPathType(d.gauge)
	case "StoragePath":
		return cadence.NewStoragePathType(d.gauge)
	case "PublicPath":
		return cadence.NewPublicPathType(d.gauge)
	case "PrivatePath":
		return cadence.NewPrivatePathType(d.gauge)
	case "AuthAccount":
		return cadence.NewAuthAccountType(d.gauge)
	case "PublicAccount":
		return cadence.NewPublicAccountType(d.gauge)
	case "AuthAccount.Keys":
		return cadence.NewAuthAccountKeysType(d.gauge)
	case "PublicAccount.Keys":
		return cadence.NewPublicAccountKeysType(d.gauge)
	case "AuthAccount.Contracts":
		return cadence.NewAuthAccountContractsType(d.gauge)
	case "PublicAccount.Contracts":
		return cadence.NewPublicAccountContractsType(d.gauge)
	case "DeployedContract":
		return cadence.NewDeployedContractType(d.gauge)
	case "AccountKey":
		return cadence.NewAccountKeyType(d.gauge)
	default:
		fieldsValue := obj.Get(fieldsKey)
		typeIDValue := toString(obj.Get(typeIDKey))
		initValue := obj.Get(initializersKey)
		return d.decodeNominalType(obj, kindValue, typeIDValue, toSlice(fieldsValue), toSlice(initValue))
	}
}

func (d *Decoder) decodeTypeValue(valueJSON interface{}) cadence.TypeValue {
	obj := toObject(valueJSON)

	return cadence.NewTypeValue(
		d.gauge,
		d.decodeType(obj.Get(staticTypeKey)),
	)
}

func (d *Decoder) decodeCapability(valueJSON interface{}) cadence.Capability {
	obj := toObject(valueJSON)

	path, ok := d.decodeJSON(obj.Get(pathKey)).(cadence.Path)
	if !ok {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return cadence.NewCapability(
		d.gauge,
		path,
		d.decodeAddress(obj.Get(addressKey)),
		d.decodeType(obj.Get(borrowTypeKey)),
	)
}

// JSON types

type jsonObject map[string]interface{}

func (obj jsonObject) Get(key string) interface{} {
	v, hasKey := obj[key]
	if !hasKey {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func (obj jsonObject) GetBool(key string) bool {
	v := obj.Get(key)
	return toBool(v)
}

func (obj jsonObject) GetString(key string) string {
	v := obj.Get(key)
	return toString(v)
}

func (obj jsonObject) GetSlice(key string) []interface{} {
	v := obj.Get(key)
	return toSlice(v)
}

func (obj jsonObject) GetValue(d *Decoder, key string) cadence.Value {
	v := obj.Get(key)
	return d.decodeJSON(v)
}

// JSON conversion helpers

func toBool(valueJSON interface{}) bool {
	v, isBool := valueJSON.(bool)
	if !isBool {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toUInt(valueJSON interface{}) uint {
	v, isNum := valueJSON.(float64)
	if !isNum {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return uint(v)
}

func toString(valueJSON interface{}) string {
	v, isString := valueJSON.(string)
	if !isString {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toSlice(valueJSON interface{}) []interface{} {
	v, isSlice := valueJSON.([]interface{})
	if !isSlice {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}

func toObject(valueJSON interface{}) jsonObject {
	v, isMap := valueJSON.(map[string]interface{})
	if !isMap {
		// TODO: improve error message
		panic(ErrInvalidJSONCadence)
	}

	return v
}
