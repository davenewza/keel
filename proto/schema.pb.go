// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.19.4
// source: proto/schema.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type OperationInputType int32

const (
	OperationInputType_OPERATION_INPUT_TYPE_UNKNOWN OperationInputType = 0
	// Means the input maps directly to a field on a model
	OperationInputType_OPERATION_INPUT_TYPE_FIELD  OperationInputType = 1
	OperationInputType_OPERATION_INPUT_TYPE_STRING OperationInputType = 2
	OperationInputType_OPERATION_INPUT_TYPE_BOOL   OperationInputType = 3 // etc...
)

// Enum value maps for OperationInputType.
var (
	OperationInputType_name = map[int32]string{
		0: "OPERATION_INPUT_TYPE_UNKNOWN",
		1: "OPERATION_INPUT_TYPE_FIELD",
		2: "OPERATION_INPUT_TYPE_STRING",
		3: "OPERATION_INPUT_TYPE_BOOL",
	}
	OperationInputType_value = map[string]int32{
		"OPERATION_INPUT_TYPE_UNKNOWN": 0,
		"OPERATION_INPUT_TYPE_FIELD":   1,
		"OPERATION_INPUT_TYPE_STRING":  2,
		"OPERATION_INPUT_TYPE_BOOL":    3,
	}
)

func (x OperationInputType) Enum() *OperationInputType {
	p := new(OperationInputType)
	*p = x
	return p
}

func (x OperationInputType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (OperationInputType) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_schema_proto_enumTypes[0].Descriptor()
}

func (OperationInputType) Type() protoreflect.EnumType {
	return &file_proto_schema_proto_enumTypes[0]
}

func (x OperationInputType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use OperationInputType.Descriptor instead.
func (OperationInputType) EnumDescriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{0}
}

type OperationImplementation int32

const (
	OperationImplementation_OPERATION_IMPLEMENTATION_UNKNOWN OperationImplementation = 0
	// Auto means the implementation of the operation is generated by Keel.
	OperationImplementation_OPERATION_IMPLEMENTATION_AUTO OperationImplementation = 1
	// Custom means the implementation of the operation is provided via custom code.
	// The code itself is not represented in this proto schema.
	OperationImplementation_OPERATION_IMPLEMENTATION_CUSTOM OperationImplementation = 2
)

// Enum value maps for OperationImplementation.
var (
	OperationImplementation_name = map[int32]string{
		0: "OPERATION_IMPLEMENTATION_UNKNOWN",
		1: "OPERATION_IMPLEMENTATION_AUTO",
		2: "OPERATION_IMPLEMENTATION_CUSTOM",
	}
	OperationImplementation_value = map[string]int32{
		"OPERATION_IMPLEMENTATION_UNKNOWN": 0,
		"OPERATION_IMPLEMENTATION_AUTO":    1,
		"OPERATION_IMPLEMENTATION_CUSTOM":  2,
	}
)

func (x OperationImplementation) Enum() *OperationImplementation {
	p := new(OperationImplementation)
	*p = x
	return p
}

func (x OperationImplementation) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (OperationImplementation) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_schema_proto_enumTypes[1].Descriptor()
}

func (OperationImplementation) Type() protoreflect.EnumType {
	return &file_proto_schema_proto_enumTypes[1]
}

func (x OperationImplementation) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use OperationImplementation.Descriptor instead.
func (OperationImplementation) EnumDescriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{1}
}

type OperationType int32

const (
	OperationType_OPERATION_TYPE_UNKNOWN OperationType = 0
	// Creates a new record and returns it
	OperationType_OPERATION_TYPE_CREATE OperationType = 1
	// Returns a single record by looking up on a unique field
	OperationType_OPERATION_TYPE_GET OperationType = 2
	// Lists records optionally filtering on certain fields. The response would be a
	// an object that supports pagination functionality and contains a "page" of results.
	OperationType_OPERATION_TYPE_LIST OperationType = 3
	// Update a single record by providing a unique lookup and some fields to update.
	// The resulting record is returned.
	OperationType_OPERATION_TYPE_UPDATE OperationType = 4
	// Delete a record and returns it's ID
	OperationType_OPERATION_TYPE_DELETE OperationType = 5
)

// Enum value maps for OperationType.
var (
	OperationType_name = map[int32]string{
		0: "OPERATION_TYPE_UNKNOWN",
		1: "OPERATION_TYPE_CREATE",
		2: "OPERATION_TYPE_GET",
		3: "OPERATION_TYPE_LIST",
		4: "OPERATION_TYPE_UPDATE",
		5: "OPERATION_TYPE_DELETE",
	}
	OperationType_value = map[string]int32{
		"OPERATION_TYPE_UNKNOWN": 0,
		"OPERATION_TYPE_CREATE":  1,
		"OPERATION_TYPE_GET":     2,
		"OPERATION_TYPE_LIST":    3,
		"OPERATION_TYPE_UPDATE":  4,
		"OPERATION_TYPE_DELETE":  5,
	}
)

func (x OperationType) Enum() *OperationType {
	p := new(OperationType)
	*p = x
	return p
}

func (x OperationType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (OperationType) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_schema_proto_enumTypes[2].Descriptor()
}

func (OperationType) Type() protoreflect.EnumType {
	return &file_proto_schema_proto_enumTypes[2]
}

func (x OperationType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use OperationType.Descriptor instead.
func (OperationType) EnumDescriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{2}
}

type FieldType int32

const (
	FieldType_FIELD_TYPE_UNKNOWN      FieldType = 0
	FieldType_FIELD_TYPE_STRING       FieldType = 1
	FieldType_FIELD_TYPE_BOOL         FieldType = 2
	FieldType_FIELD_TYPE_INT          FieldType = 3
	FieldType_FIELD_TYPE_TIMESTAMP    FieldType = 4
	FieldType_FIELD_TYPE_DATE         FieldType = 5
	FieldType_FIELD_TYPE_ID           FieldType = 6
	FieldType_FIELD_TYPE_RELATIONSHIP FieldType = 7 // etc...
)

// Enum value maps for FieldType.
var (
	FieldType_name = map[int32]string{
		0: "FIELD_TYPE_UNKNOWN",
		1: "FIELD_TYPE_STRING",
		2: "FIELD_TYPE_BOOL",
		3: "FIELD_TYPE_INT",
		4: "FIELD_TYPE_TIMESTAMP",
		5: "FIELD_TYPE_DATE",
		6: "FIELD_TYPE_ID",
		7: "FIELD_TYPE_RELATIONSHIP",
	}
	FieldType_value = map[string]int32{
		"FIELD_TYPE_UNKNOWN":      0,
		"FIELD_TYPE_STRING":       1,
		"FIELD_TYPE_BOOL":         2,
		"FIELD_TYPE_INT":          3,
		"FIELD_TYPE_TIMESTAMP":    4,
		"FIELD_TYPE_DATE":         5,
		"FIELD_TYPE_ID":           6,
		"FIELD_TYPE_RELATIONSHIP": 7,
	}
)

func (x FieldType) Enum() *FieldType {
	p := new(FieldType)
	*p = x
	return p
}

func (x FieldType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (FieldType) Descriptor() protoreflect.EnumDescriptor {
	return file_proto_schema_proto_enumTypes[3].Descriptor()
}

func (FieldType) Type() protoreflect.EnumType {
	return &file_proto_schema_proto_enumTypes[3]
}

func (x FieldType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use FieldType.Descriptor instead.
func (FieldType) EnumDescriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{3}
}

type Schema struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Models []*Model `protobuf:"bytes,1,rep,name=models,proto3" json:"models,omitempty"`
}

func (x *Schema) Reset() {
	*x = Schema{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_schema_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Schema) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Schema) ProtoMessage() {}

func (x *Schema) ProtoReflect() protoreflect.Message {
	mi := &file_proto_schema_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Schema.ProtoReflect.Descriptor instead.
func (*Schema) Descriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{0}
}

func (x *Schema) GetModels() []*Model {
	if x != nil {
		return x.Models
	}
	return nil
}

type Model struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the model. Must be in PascalCase and be unique within the schema.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The fields this model contains
	Fields []*Field `protobuf:"bytes,2,rep,name=fields,proto3" json:"fields,omitempty"`
	// The operations this model defines. Contains both operations that will be auto
	// generated and also custom functions
	Operations []*Operation `protobuf:"bytes,3,rep,name=operations,proto3" json:"operations,omitempty"`
}

func (x *Model) Reset() {
	*x = Model{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_schema_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Model) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Model) ProtoMessage() {}

func (x *Model) ProtoReflect() protoreflect.Message {
	mi := &file_proto_schema_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Model.ProtoReflect.Descriptor instead.
func (*Model) Descriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{1}
}

func (x *Model) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Model) GetFields() []*Field {
	if x != nil {
		return x.Fields
	}
	return nil
}

func (x *Model) GetOperations() []*Operation {
	if x != nil {
		return x.Operations
	}
	return nil
}

type Field struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the model this field belongs to.
	ModelName string `protobuf:"bytes,1,opt,name=model_name,json=modelName,proto3" json:"model_name,omitempty"`
	// The name of the field. Must be in lowerCamelCase and be unique within the model.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// The type of the field.
	Type FieldType `protobuf:"varint,3,opt,name=type,proto3,enum=proto.FieldType" json:"type,omitempty"`
	// If true then this field is allowed to be null
	Optional bool `protobuf:"varint,4,opt,name=optional,proto3" json:"optional,omitempty"`
	// If true then this field will have a unique constraint added to it meaning
	// a given value can only exist in a given row
	// Cannot be true if `repeated` is true
	Unique bool `protobuf:"varint,5,opt,name=unique,proto3" json:"unique,omitempty"`
	// If true then this field can contain multiple values.
	// Cannot be true if `unique` is true.
	Repeated bool `protobuf:"varint,6,opt,name=repeated,proto3" json:"repeated,omitempty"`
}

func (x *Field) Reset() {
	*x = Field{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_schema_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Field) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Field) ProtoMessage() {}

func (x *Field) ProtoReflect() protoreflect.Message {
	mi := &file_proto_schema_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Field.ProtoReflect.Descriptor instead.
func (*Field) Descriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{2}
}

func (x *Field) GetModelName() string {
	if x != nil {
		return x.ModelName
	}
	return ""
}

func (x *Field) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Field) GetType() FieldType {
	if x != nil {
		return x.Type
	}
	return FieldType_FIELD_TYPE_UNKNOWN
}

func (x *Field) GetOptional() bool {
	if x != nil {
		return x.Optional
	}
	return false
}

func (x *Field) GetUnique() bool {
	if x != nil {
		return x.Unique
	}
	return false
}

func (x *Field) GetRepeated() bool {
	if x != nil {
		return x.Repeated
	}
	return false
}

type Operation struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The name of the model this operation belongs to.
	ModelName string `protobuf:"bytes,1,opt,name=model_name,json=modelName,proto3" json:"model_name,omitempty"`
	// The name of the operation. Must be in lowerCamelCase and be unique across all operations
	// across all models within the schema. This is because in both RPC and GraphQL operations
	// are top-level and so two different models can't both define an operation with the same name.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// The type of this operation.
	Type OperationType `protobuf:"varint,3,opt,name=type,proto3,enum=proto.OperationType" json:"type,omitempty"`
	// Whether this operation will be auto-generated by Keel or implemented with a custom function.
	Implementation OperationImplementation `protobuf:"varint,4,opt,name=implementation,proto3,enum=proto.OperationImplementation" json:"implementation,omitempty"`
	// The inputs this operation accepts.
	Inputs []*OperationInput `protobuf:"bytes,5,rep,name=inputs,proto3" json:"inputs,omitempty"`
}

func (x *Operation) Reset() {
	*x = Operation{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_schema_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Operation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Operation) ProtoMessage() {}

func (x *Operation) ProtoReflect() protoreflect.Message {
	mi := &file_proto_schema_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Operation.ProtoReflect.Descriptor instead.
func (*Operation) Descriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{3}
}

func (x *Operation) GetModelName() string {
	if x != nil {
		return x.ModelName
	}
	return ""
}

func (x *Operation) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Operation) GetType() OperationType {
	if x != nil {
		return x.Type
	}
	return OperationType_OPERATION_TYPE_UNKNOWN
}

func (x *Operation) GetImplementation() OperationImplementation {
	if x != nil {
		return x.Implementation
	}
	return OperationImplementation_OPERATION_IMPLEMENTATION_UNKNOWN
}

func (x *Operation) GetInputs() []*OperationInput {
	if x != nil {
		return x.Inputs
	}
	return nil
}

type OperationInput struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Name of the input. Must be lowerCamelCase and unique within the parent operation.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The type of this input. If type is OPERATION_INPUT_TYPE_FIELD then `model_name` and `field_name`
	// must also be populated.
	Type OperationInputType `protobuf:"varint,2,opt,name=type,proto3,enum=proto.OperationInputType" json:"type,omitempty"`
	// Set to true if this input field should accept a list of values. Only applies if `type` is not
	// set to OPERATION_INPUT_TYPE_FIELD
	Repeated bool `protobuf:"varint,3,opt,name=repeated,proto3" json:"repeated,omitempty"`
	// Set to true if this input field is optional. Only applies if `type` is not
	// set to OPERATION_INPUT_TYPE_FIELD
	Optional bool `protobuf:"varint,4,opt,name=optional,proto3" json:"optional,omitempty"`
	// The name of the model this input field is referring to. Should only be set if `type` is
	// set to OPERATION_INPUT_TYPE_FIELD
	ModelName *wrapperspb.StringValue `protobuf:"bytes,5,opt,name=model_name,json=modelName,proto3" json:"model_name,omitempty"`
	// The name of the field inside `model_name` that this input type refers to. Should only
	// be set if `type` is set to OPERATION_INPUT_TYPE_FIELD
	FieldName *wrapperspb.StringValue `protobuf:"bytes,6,opt,name=field_name,json=fieldName,proto3" json:"field_name,omitempty"`
}

func (x *OperationInput) Reset() {
	*x = OperationInput{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_schema_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OperationInput) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OperationInput) ProtoMessage() {}

func (x *OperationInput) ProtoReflect() protoreflect.Message {
	mi := &file_proto_schema_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OperationInput.ProtoReflect.Descriptor instead.
func (*OperationInput) Descriptor() ([]byte, []int) {
	return file_proto_schema_proto_rawDescGZIP(), []int{4}
}

func (x *OperationInput) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *OperationInput) GetType() OperationInputType {
	if x != nil {
		return x.Type
	}
	return OperationInputType_OPERATION_INPUT_TYPE_UNKNOWN
}

func (x *OperationInput) GetRepeated() bool {
	if x != nil {
		return x.Repeated
	}
	return false
}

func (x *OperationInput) GetOptional() bool {
	if x != nil {
		return x.Optional
	}
	return false
}

func (x *OperationInput) GetModelName() *wrapperspb.StringValue {
	if x != nil {
		return x.ModelName
	}
	return nil
}

func (x *OperationInput) GetFieldName() *wrapperspb.StringValue {
	if x != nil {
		return x.FieldName
	}
	return nil
}

var File_proto_schema_proto protoreflect.FileDescriptor

var file_proto_schema_proto_rawDesc = []byte{
	0x0a, 0x12, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x73, 0x63, 0x68, 0x65, 0x6d, 0x61, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x77, 0x72, 0x61,
	0x70, 0x70, 0x65, 0x72, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x2e, 0x0a, 0x06, 0x53,
	0x63, 0x68, 0x65, 0x6d, 0x61, 0x12, 0x24, 0x0a, 0x06, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4d, 0x6f,
	0x64, 0x65, 0x6c, 0x52, 0x06, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x73, 0x22, 0x73, 0x0a, 0x05, 0x4d,
	0x6f, 0x64, 0x65, 0x6c, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x24, 0x0a, 0x06, 0x66, 0x69, 0x65, 0x6c,
	0x64, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x52, 0x06, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x73, 0x12, 0x30,
	0x0a, 0x0a, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x03, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x10, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4f, 0x70, 0x65, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0a, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x22, 0xb0, 0x01, 0x0a, 0x05, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x6f,
	0x64, 0x65, 0x6c, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09,
	0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x24, 0x0a,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x10, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74,
	0x79, 0x70, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x12,
	0x16, 0x0a, 0x06, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52,
	0x06, 0x75, 0x6e, 0x69, 0x71, 0x75, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x70, 0x65, 0x61,
	0x74, 0x65, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x72, 0x65, 0x70, 0x65, 0x61,
	0x74, 0x65, 0x64, 0x22, 0xdf, 0x01, 0x0a, 0x09, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x6e, 0x61, 0x6d, 0x65, 0x12, 0x28, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x14, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4f, 0x70, 0x65, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x46,
	0x0a, 0x0e, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4f,
	0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6d, 0x70, 0x6c, 0x65, 0x6d, 0x65, 0x6e,
	0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0e, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x6d, 0x65, 0x6e,
	0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x2d, 0x0a, 0x06, 0x69, 0x6e, 0x70, 0x75, 0x74, 0x73,
	0x18, 0x05, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4f,
	0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x52, 0x06, 0x69,
	0x6e, 0x70, 0x75, 0x74, 0x73, 0x22, 0x85, 0x02, 0x0a, 0x0e, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2d, 0x0a, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x19, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75,
	0x74, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x72,
	0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x72,
	0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x6f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x61, 0x6c, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x08, 0x6f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x61, 0x6c, 0x12, 0x3b, 0x0a, 0x0a, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x52, 0x09, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x3b, 0x0a, 0x0a, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x06,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x52, 0x09, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x2a, 0x96, 0x01,
	0x0a, 0x12, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6e, 0x70, 0x75, 0x74,
	0x54, 0x79, 0x70, 0x65, 0x12, 0x20, 0x0a, 0x1c, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54, 0x49, 0x4f,
	0x4e, 0x5f, 0x49, 0x4e, 0x50, 0x55, 0x54, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x4e, 0x4b,
	0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x1e, 0x0a, 0x1a, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54,
	0x49, 0x4f, 0x4e, 0x5f, 0x49, 0x4e, 0x50, 0x55, 0x54, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x46,
	0x49, 0x45, 0x4c, 0x44, 0x10, 0x01, 0x12, 0x1f, 0x0a, 0x1b, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54,
	0x49, 0x4f, 0x4e, 0x5f, 0x49, 0x4e, 0x50, 0x55, 0x54, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x53,
	0x54, 0x52, 0x49, 0x4e, 0x47, 0x10, 0x02, 0x12, 0x1d, 0x0a, 0x19, 0x4f, 0x50, 0x45, 0x52, 0x41,
	0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x49, 0x4e, 0x50, 0x55, 0x54, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f,
	0x42, 0x4f, 0x4f, 0x4c, 0x10, 0x03, 0x2a, 0x87, 0x01, 0x0a, 0x17, 0x4f, 0x70, 0x65, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x49, 0x6d, 0x70, 0x6c, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x12, 0x24, 0x0a, 0x20, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f,
	0x49, 0x4d, 0x50, 0x4c, 0x45, 0x4d, 0x45, 0x4e, 0x54, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x55,
	0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x21, 0x0a, 0x1d, 0x4f, 0x50, 0x45, 0x52,
	0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x49, 0x4d, 0x50, 0x4c, 0x45, 0x4d, 0x45, 0x4e, 0x54, 0x41,
	0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x41, 0x55, 0x54, 0x4f, 0x10, 0x01, 0x12, 0x23, 0x0a, 0x1f, 0x4f,
	0x50, 0x45, 0x52, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x49, 0x4d, 0x50, 0x4c, 0x45, 0x4d, 0x45,
	0x4e, 0x54, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x43, 0x55, 0x53, 0x54, 0x4f, 0x4d, 0x10, 0x02,
	0x2a, 0xad, 0x01, 0x0a, 0x0d, 0x4f, 0x70, 0x65, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x79,
	0x70, 0x65, 0x12, 0x1a, 0x0a, 0x16, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f,
	0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x19,
	0x0a, 0x15, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x54, 0x59, 0x50, 0x45,
	0x5f, 0x43, 0x52, 0x45, 0x41, 0x54, 0x45, 0x10, 0x01, 0x12, 0x16, 0x0a, 0x12, 0x4f, 0x50, 0x45,
	0x52, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x47, 0x45, 0x54, 0x10,
	0x02, 0x12, 0x17, 0x0a, 0x13, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x54,
	0x59, 0x50, 0x45, 0x5f, 0x4c, 0x49, 0x53, 0x54, 0x10, 0x03, 0x12, 0x19, 0x0a, 0x15, 0x4f, 0x50,
	0x45, 0x52, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x50, 0x44,
	0x41, 0x54, 0x45, 0x10, 0x04, 0x12, 0x19, 0x0a, 0x15, 0x4f, 0x50, 0x45, 0x52, 0x41, 0x54, 0x49,
	0x4f, 0x4e, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x44, 0x45, 0x4c, 0x45, 0x54, 0x45, 0x10, 0x05,
	0x2a, 0xc2, 0x01, 0x0a, 0x09, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x54, 0x79, 0x70, 0x65, 0x12, 0x16,
	0x0a, 0x12, 0x46, 0x49, 0x45, 0x4c, 0x44, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x55, 0x4e, 0x4b,
	0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x15, 0x0a, 0x11, 0x46, 0x49, 0x45, 0x4c, 0x44, 0x5f,
	0x54, 0x59, 0x50, 0x45, 0x5f, 0x53, 0x54, 0x52, 0x49, 0x4e, 0x47, 0x10, 0x01, 0x12, 0x13, 0x0a,
	0x0f, 0x46, 0x49, 0x45, 0x4c, 0x44, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x42, 0x4f, 0x4f, 0x4c,
	0x10, 0x02, 0x12, 0x12, 0x0a, 0x0e, 0x46, 0x49, 0x45, 0x4c, 0x44, 0x5f, 0x54, 0x59, 0x50, 0x45,
	0x5f, 0x49, 0x4e, 0x54, 0x10, 0x03, 0x12, 0x18, 0x0a, 0x14, 0x46, 0x49, 0x45, 0x4c, 0x44, 0x5f,
	0x54, 0x59, 0x50, 0x45, 0x5f, 0x54, 0x49, 0x4d, 0x45, 0x53, 0x54, 0x41, 0x4d, 0x50, 0x10, 0x04,
	0x12, 0x13, 0x0a, 0x0f, 0x46, 0x49, 0x45, 0x4c, 0x44, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x44,
	0x41, 0x54, 0x45, 0x10, 0x05, 0x12, 0x11, 0x0a, 0x0d, 0x46, 0x49, 0x45, 0x4c, 0x44, 0x5f, 0x54,
	0x59, 0x50, 0x45, 0x5f, 0x49, 0x44, 0x10, 0x06, 0x12, 0x1b, 0x0a, 0x17, 0x46, 0x49, 0x45, 0x4c,
	0x44, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x52, 0x45, 0x4c, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x53,
	0x48, 0x49, 0x50, 0x10, 0x07, 0x42, 0x1b, 0x5a, 0x19, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x74, 0x65, 0x61, 0x6d, 0x6b, 0x65, 0x65, 0x6c, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_schema_proto_rawDescOnce sync.Once
	file_proto_schema_proto_rawDescData = file_proto_schema_proto_rawDesc
)

func file_proto_schema_proto_rawDescGZIP() []byte {
	file_proto_schema_proto_rawDescOnce.Do(func() {
		file_proto_schema_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_schema_proto_rawDescData)
	})
	return file_proto_schema_proto_rawDescData
}

var file_proto_schema_proto_enumTypes = make([]protoimpl.EnumInfo, 4)
var file_proto_schema_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_proto_schema_proto_goTypes = []interface{}{
	(OperationInputType)(0),        // 0: proto.OperationInputType
	(OperationImplementation)(0),   // 1: proto.OperationImplementation
	(OperationType)(0),             // 2: proto.OperationType
	(FieldType)(0),                 // 3: proto.FieldType
	(*Schema)(nil),                 // 4: proto.Schema
	(*Model)(nil),                  // 5: proto.Model
	(*Field)(nil),                  // 6: proto.Field
	(*Operation)(nil),              // 7: proto.Operation
	(*OperationInput)(nil),         // 8: proto.OperationInput
	(*wrapperspb.StringValue)(nil), // 9: google.protobuf.StringValue
}
var file_proto_schema_proto_depIdxs = []int32{
	5,  // 0: proto.Schema.models:type_name -> proto.Model
	6,  // 1: proto.Model.fields:type_name -> proto.Field
	7,  // 2: proto.Model.operations:type_name -> proto.Operation
	3,  // 3: proto.Field.type:type_name -> proto.FieldType
	2,  // 4: proto.Operation.type:type_name -> proto.OperationType
	1,  // 5: proto.Operation.implementation:type_name -> proto.OperationImplementation
	8,  // 6: proto.Operation.inputs:type_name -> proto.OperationInput
	0,  // 7: proto.OperationInput.type:type_name -> proto.OperationInputType
	9,  // 8: proto.OperationInput.model_name:type_name -> google.protobuf.StringValue
	9,  // 9: proto.OperationInput.field_name:type_name -> google.protobuf.StringValue
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_proto_schema_proto_init() }
func file_proto_schema_proto_init() {
	if File_proto_schema_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_schema_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Schema); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_schema_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Model); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_schema_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Field); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_schema_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Operation); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_proto_schema_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OperationInput); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_schema_proto_rawDesc,
			NumEnums:      4,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_proto_schema_proto_goTypes,
		DependencyIndexes: file_proto_schema_proto_depIdxs,
		EnumInfos:         file_proto_schema_proto_enumTypes,
		MessageInfos:      file_proto_schema_proto_msgTypes,
	}.Build()
	File_proto_schema_proto = out.File
	file_proto_schema_proto_rawDesc = nil
	file_proto_schema_proto_goTypes = nil
	file_proto_schema_proto_depIdxs = nil
}
