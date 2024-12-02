// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        (unknown)
// source: spacex/api/device/services/unlock/service.proto

package unlock

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type UnlockChallenge struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DeviceId        string   `protobuf:"bytes,1,opt,name=device_id,json=deviceId,proto3" json:"device_id,omitempty"`
	Nonce           []byte   `protobuf:"bytes,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	SignSpki        []byte   `protobuf:"bytes,4,opt,name=sign_spki,json=signSpki,proto3" json:"sign_spki,omitempty"`
	GrantKeydata    string   `protobuf:"bytes,5,opt,name=grant_keydata,json=grantKeydata,proto3" json:"grant_keydata,omitempty"`
	ServiceKeydata  string   `protobuf:"bytes,6,opt,name=service_keydata,json=serviceKeydata,proto3" json:"service_keydata,omitempty"`
	AuthorityGrants []string `protobuf:"bytes,7,rep,name=authority_grants,json=authorityGrants,proto3" json:"authority_grants,omitempty"`
}

func (x *UnlockChallenge) Reset() {
	*x = UnlockChallenge{}
	if protoimpl.UnsafeEnabled {
		mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UnlockChallenge) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UnlockChallenge) ProtoMessage() {}

func (x *UnlockChallenge) ProtoReflect() protoreflect.Message {
	mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UnlockChallenge.ProtoReflect.Descriptor instead.
func (*UnlockChallenge) Descriptor() ([]byte, []int) {
	return file_spacex_api_device_services_unlock_service_proto_rawDescGZIP(), []int{0}
}

func (x *UnlockChallenge) GetDeviceId() string {
	if x != nil {
		return x.DeviceId
	}
	return ""
}

func (x *UnlockChallenge) GetNonce() []byte {
	if x != nil {
		return x.Nonce
	}
	return nil
}

func (x *UnlockChallenge) GetSignSpki() []byte {
	if x != nil {
		return x.SignSpki
	}
	return nil
}

func (x *UnlockChallenge) GetGrantKeydata() string {
	if x != nil {
		return x.GrantKeydata
	}
	return ""
}

func (x *UnlockChallenge) GetServiceKeydata() string {
	if x != nil {
		return x.ServiceKeydata
	}
	return ""
}

func (x *UnlockChallenge) GetAuthorityGrants() []string {
	if x != nil {
		return x.AuthorityGrants
	}
	return nil
}

type StartUnlockRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *StartUnlockRequest) Reset() {
	*x = StartUnlockRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartUnlockRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartUnlockRequest) ProtoMessage() {}

func (x *StartUnlockRequest) ProtoReflect() protoreflect.Message {
	mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartUnlockRequest.ProtoReflect.Descriptor instead.
func (*StartUnlockRequest) Descriptor() ([]byte, []int) {
	return file_spacex_api_device_services_unlock_service_proto_rawDescGZIP(), []int{1}
}

type StartUnlockResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DeviceId string `protobuf:"bytes,1,opt,name=device_id,json=deviceId,proto3" json:"device_id,omitempty"`
	Nonce    []byte `protobuf:"bytes,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	SignSpki []byte `protobuf:"bytes,3,opt,name=sign_spki,json=signSpki,proto3" json:"sign_spki,omitempty"`
}

func (x *StartUnlockResponse) Reset() {
	*x = StartUnlockResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartUnlockResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartUnlockResponse) ProtoMessage() {}

func (x *StartUnlockResponse) ProtoReflect() protoreflect.Message {
	mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartUnlockResponse.ProtoReflect.Descriptor instead.
func (*StartUnlockResponse) Descriptor() ([]byte, []int) {
	return file_spacex_api_device_services_unlock_service_proto_rawDescGZIP(), []int{2}
}

func (x *StartUnlockResponse) GetDeviceId() string {
	if x != nil {
		return x.DeviceId
	}
	return ""
}

func (x *StartUnlockResponse) GetNonce() []byte {
	if x != nil {
		return x.Nonce
	}
	return nil
}

func (x *StartUnlockResponse) GetSignSpki() []byte {
	if x != nil {
		return x.SignSpki
	}
	return nil
}

type FinishUnlockRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Challenge []byte `protobuf:"bytes,1,opt,name=challenge,proto3" json:"challenge,omitempty"`
	Signature []byte `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (x *FinishUnlockRequest) Reset() {
	*x = FinishUnlockRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FinishUnlockRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FinishUnlockRequest) ProtoMessage() {}

func (x *FinishUnlockRequest) ProtoReflect() protoreflect.Message {
	mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FinishUnlockRequest.ProtoReflect.Descriptor instead.
func (*FinishUnlockRequest) Descriptor() ([]byte, []int) {
	return file_spacex_api_device_services_unlock_service_proto_rawDescGZIP(), []int{3}
}

func (x *FinishUnlockRequest) GetChallenge() []byte {
	if x != nil {
		return x.Challenge
	}
	return nil
}

func (x *FinishUnlockRequest) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

type FinishUnlockResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status uint32 `protobuf:"varint,1,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *FinishUnlockResponse) Reset() {
	*x = FinishUnlockResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FinishUnlockResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FinishUnlockResponse) ProtoMessage() {}

func (x *FinishUnlockResponse) ProtoReflect() protoreflect.Message {
	mi := &file_spacex_api_device_services_unlock_service_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FinishUnlockResponse.ProtoReflect.Descriptor instead.
func (*FinishUnlockResponse) Descriptor() ([]byte, []int) {
	return file_spacex_api_device_services_unlock_service_proto_rawDescGZIP(), []int{4}
}

func (x *FinishUnlockResponse) GetStatus() uint32 {
	if x != nil {
		return x.Status
	}
	return 0
}

var File_spacex_api_device_services_unlock_service_proto protoreflect.FileDescriptor

var file_spacex_api_device_services_unlock_service_proto_rawDesc = []byte{
	0x0a, 0x2f, 0x73, 0x70, 0x61, 0x63, 0x65, 0x78, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x64, 0x65, 0x76,
	0x69, 0x63, 0x65, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2f, 0x75, 0x6e, 0x6c,
	0x6f, 0x63, 0x6b, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x21, 0x53, 0x70, 0x61, 0x63, 0x65, 0x58, 0x2e, 0x41, 0x50, 0x49, 0x2e, 0x44, 0x65,
	0x76, 0x69, 0x63, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x55, 0x6e,
	0x6c, 0x6f, 0x63, 0x6b, 0x22, 0xec, 0x01, 0x0a, 0x0f, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x43,
	0x68, 0x61, 0x6c, 0x6c, 0x65, 0x6e, 0x67, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x64, 0x65, 0x76, 0x69,
	0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x64, 0x65, 0x76,
	0x69, 0x63, 0x65, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x73,
	0x69, 0x67, 0x6e, 0x5f, 0x73, 0x70, 0x6b, 0x69, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x08,
	0x73, 0x69, 0x67, 0x6e, 0x53, 0x70, 0x6b, 0x69, 0x12, 0x23, 0x0a, 0x0d, 0x67, 0x72, 0x61, 0x6e,
	0x74, 0x5f, 0x6b, 0x65, 0x79, 0x64, 0x61, 0x74, 0x61, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0c, 0x67, 0x72, 0x61, 0x6e, 0x74, 0x4b, 0x65, 0x79, 0x64, 0x61, 0x74, 0x61, 0x12, 0x27, 0x0a,
	0x0f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x6b, 0x65, 0x79, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4b,
	0x65, 0x79, 0x64, 0x61, 0x74, 0x61, 0x12, 0x29, 0x0a, 0x10, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72,
	0x69, 0x74, 0x79, 0x5f, 0x67, 0x72, 0x61, 0x6e, 0x74, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x0f, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79, 0x47, 0x72, 0x61, 0x6e, 0x74,
	0x73, 0x4a, 0x04, 0x08, 0x03, 0x10, 0x04, 0x52, 0x0a, 0x67, 0x72, 0x61, 0x6e, 0x74, 0x5f, 0x73,
	0x70, 0x6b, 0x69, 0x22, 0x14, 0x0a, 0x12, 0x53, 0x74, 0x61, 0x72, 0x74, 0x55, 0x6e, 0x6c, 0x6f,
	0x63, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x65, 0x0a, 0x13, 0x53, 0x74, 0x61,
	0x72, 0x74, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x1b, 0x0a, 0x09, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x08, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x49, 0x64, 0x12, 0x14, 0x0a,
	0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x6e, 0x6f,
	0x6e, 0x63, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x5f, 0x73, 0x70, 0x6b, 0x69,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x73, 0x69, 0x67, 0x6e, 0x53, 0x70, 0x6b, 0x69,
	0x22, 0x51, 0x0a, 0x13, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1c, 0x0a, 0x09, 0x63, 0x68, 0x61, 0x6c, 0x6c,
	0x65, 0x6e, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x63, 0x68, 0x61, 0x6c,
	0x6c, 0x65, 0x6e, 0x67, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75,
	0x72, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x22, 0x2e, 0x0a, 0x14, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x55, 0x6e, 0x6c,
	0x6f, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x06, 0x73, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x32, 0x8e, 0x02, 0x0a, 0x0d, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x7c, 0x0a, 0x0b, 0x53, 0x74, 0x61, 0x72, 0x74, 0x55, 0x6e,
	0x6c, 0x6f, 0x63, 0x6b, 0x12, 0x35, 0x2e, 0x53, 0x70, 0x61, 0x63, 0x65, 0x58, 0x2e, 0x41, 0x50,
	0x49, 0x2e, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x73, 0x2e, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x55, 0x6e,
	0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x36, 0x2e, 0x53, 0x70,
	0x61, 0x63, 0x65, 0x58, 0x2e, 0x41, 0x50, 0x49, 0x2e, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x2e,
	0x53, 0x74, 0x61, 0x72, 0x74, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x12, 0x7f, 0x0a, 0x0c, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x55, 0x6e, 0x6c,
	0x6f, 0x63, 0x6b, 0x12, 0x36, 0x2e, 0x53, 0x70, 0x61, 0x63, 0x65, 0x58, 0x2e, 0x41, 0x50, 0x49,
	0x2e, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73,
	0x2e, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x2e, 0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x55, 0x6e,
	0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x37, 0x2e, 0x53, 0x70,
	0x61, 0x63, 0x65, 0x58, 0x2e, 0x41, 0x50, 0x49, 0x2e, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x2e,
	0x46, 0x69, 0x6e, 0x69, 0x73, 0x68, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x42, 0xc8, 0x02, 0x0a, 0x25, 0x63, 0x6f, 0x6d, 0x2e, 0x53, 0x70, 0x61,
	0x63, 0x65, 0x58, 0x2e, 0x41, 0x50, 0x49, 0x2e, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x53,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x42, 0x0c,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x67,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x61, 0x6b, 0x69, 0x6e,
	0x73, 0x2f, 0x6f, 0x74, 0x65, 0x6c, 0x2d, 0x63, 0x6f, 0x6c, 0x6c, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x2d, 0x62, 0x75, 0x6e, 0x64, 0x6c, 0x65, 0x2f, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65, 0x72,
	0x2f, 0x73, 0x74, 0x61, 0x72, 0x6c, 0x69, 0x6e, 0x6b, 0x72, 0x65, 0x63, 0x65, 0x69, 0x76, 0x65,
	0x72, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x73, 0x70, 0x61, 0x63, 0x65, 0x78, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73,
	0x2f, 0x75, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0xa2, 0x02, 0x05, 0x53, 0x41, 0x44, 0x53, 0x55, 0xaa,
	0x02, 0x21, 0x53, 0x70, 0x61, 0x63, 0x65, 0x58, 0x2e, 0x41, 0x50, 0x49, 0x2e, 0x44, 0x65, 0x76,
	0x69, 0x63, 0x65, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x2e, 0x55, 0x6e, 0x6c,
	0x6f, 0x63, 0x6b, 0xca, 0x02, 0x21, 0x53, 0x70, 0x61, 0x63, 0x65, 0x58, 0x5c, 0x41, 0x50, 0x49,
	0x5c, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5c, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73,
	0x5c, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0xe2, 0x02, 0x2d, 0x53, 0x70, 0x61, 0x63, 0x65, 0x58,
	0x5c, 0x41, 0x50, 0x49, 0x5c, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x5c, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x73, 0x5c, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x5c, 0x47, 0x50, 0x42, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x25, 0x53, 0x70, 0x61, 0x63, 0x65, 0x58,
	0x3a, 0x3a, 0x41, 0x50, 0x49, 0x3a, 0x3a, 0x44, 0x65, 0x76, 0x69, 0x63, 0x65, 0x3a, 0x3a, 0x53,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x73, 0x3a, 0x3a, 0x55, 0x6e, 0x6c, 0x6f, 0x63, 0x6b, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_spacex_api_device_services_unlock_service_proto_rawDescOnce sync.Once
	file_spacex_api_device_services_unlock_service_proto_rawDescData = file_spacex_api_device_services_unlock_service_proto_rawDesc
)

func file_spacex_api_device_services_unlock_service_proto_rawDescGZIP() []byte {
	file_spacex_api_device_services_unlock_service_proto_rawDescOnce.Do(func() {
		file_spacex_api_device_services_unlock_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_spacex_api_device_services_unlock_service_proto_rawDescData)
	})
	return file_spacex_api_device_services_unlock_service_proto_rawDescData
}

var file_spacex_api_device_services_unlock_service_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_spacex_api_device_services_unlock_service_proto_goTypes = []any{
	(*UnlockChallenge)(nil),      // 0: SpaceX.API.Device.Services.Unlock.UnlockChallenge
	(*StartUnlockRequest)(nil),   // 1: SpaceX.API.Device.Services.Unlock.StartUnlockRequest
	(*StartUnlockResponse)(nil),  // 2: SpaceX.API.Device.Services.Unlock.StartUnlockResponse
	(*FinishUnlockRequest)(nil),  // 3: SpaceX.API.Device.Services.Unlock.FinishUnlockRequest
	(*FinishUnlockResponse)(nil), // 4: SpaceX.API.Device.Services.Unlock.FinishUnlockResponse
}
var file_spacex_api_device_services_unlock_service_proto_depIdxs = []int32{
	1, // 0: SpaceX.API.Device.Services.Unlock.UnlockService.StartUnlock:input_type -> SpaceX.API.Device.Services.Unlock.StartUnlockRequest
	3, // 1: SpaceX.API.Device.Services.Unlock.UnlockService.FinishUnlock:input_type -> SpaceX.API.Device.Services.Unlock.FinishUnlockRequest
	2, // 2: SpaceX.API.Device.Services.Unlock.UnlockService.StartUnlock:output_type -> SpaceX.API.Device.Services.Unlock.StartUnlockResponse
	4, // 3: SpaceX.API.Device.Services.Unlock.UnlockService.FinishUnlock:output_type -> SpaceX.API.Device.Services.Unlock.FinishUnlockResponse
	2, // [2:4] is the sub-list for method output_type
	0, // [0:2] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_spacex_api_device_services_unlock_service_proto_init() }
func file_spacex_api_device_services_unlock_service_proto_init() {
	if File_spacex_api_device_services_unlock_service_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_spacex_api_device_services_unlock_service_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*UnlockChallenge); i {
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
		file_spacex_api_device_services_unlock_service_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*StartUnlockRequest); i {
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
		file_spacex_api_device_services_unlock_service_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*StartUnlockResponse); i {
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
		file_spacex_api_device_services_unlock_service_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*FinishUnlockRequest); i {
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
		file_spacex_api_device_services_unlock_service_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*FinishUnlockResponse); i {
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
			RawDescriptor: file_spacex_api_device_services_unlock_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_spacex_api_device_services_unlock_service_proto_goTypes,
		DependencyIndexes: file_spacex_api_device_services_unlock_service_proto_depIdxs,
		MessageInfos:      file_spacex_api_device_services_unlock_service_proto_msgTypes,
	}.Build()
	File_spacex_api_device_services_unlock_service_proto = out.File
	file_spacex_api_device_services_unlock_service_proto_rawDesc = nil
	file_spacex_api_device_services_unlock_service_proto_goTypes = nil
	file_spacex_api_device_services_unlock_service_proto_depIdxs = nil
}
