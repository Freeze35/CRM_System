// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        v5.27.3
// source: dbservice/proto/dbchat.proto

package dbchat

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type UserId struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UserId        int64                  `protobuf:"varint,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	RoleId        int64                  `protobuf:"varint,2,opt,name=role_id,json=roleId,proto3" json:"role_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *UserId) Reset() {
	*x = UserId{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *UserId) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UserId) ProtoMessage() {}

func (x *UserId) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UserId.ProtoReflect.Descriptor instead.
func (*UserId) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{0}
}

func (x *UserId) GetUserId() int64 {
	if x != nil {
		return x.UserId
	}
	return 0
}

func (x *UserId) GetRoleId() int64 {
	if x != nil {
		return x.RoleId
	}
	return 0
}

type CreateChatRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UsersId       []*UserId              `protobuf:"bytes,1,rep,name=users_id,json=usersId,proto3" json:"users_id,omitempty"`    // ID пользователей
	ChatName      string                 `protobuf:"bytes,3,opt,name=chat_name,json=chatName,proto3" json:"chat_name,omitempty"` // Название базы данных
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateChatRequest) Reset() {
	*x = CreateChatRequest{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateChatRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateChatRequest) ProtoMessage() {}

func (x *CreateChatRequest) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateChatRequest.ProtoReflect.Descriptor instead.
func (*CreateChatRequest) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{1}
}

func (x *CreateChatRequest) GetUsersId() []*UserId {
	if x != nil {
		return x.UsersId
	}
	return nil
}

func (x *CreateChatRequest) GetChatName() string {
	if x != nil {
		return x.ChatName
	}
	return ""
}

type CreateChatResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Message       string                 `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`                       // Ответ от сервера
	ChatId        int64                  `protobuf:"varint,2,opt,name=chat_id,json=chatId,proto3" json:"chat_id,omitempty"`          // ID чата
	CreatedAt     int64                  `protobuf:"varint,3,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"` // Время создания сообщения (timestamp)
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CreateChatResponse) Reset() {
	*x = CreateChatResponse{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CreateChatResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateChatResponse) ProtoMessage() {}

func (x *CreateChatResponse) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateChatResponse.ProtoReflect.Descriptor instead.
func (*CreateChatResponse) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{2}
}

func (x *CreateChatResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *CreateChatResponse) GetChatId() int64 {
	if x != nil {
		return x.ChatId
	}
	return 0
}

func (x *CreateChatResponse) GetCreatedAt() int64 {
	if x != nil {
		return x.CreatedAt
	}
	return 0
}

type AddUsersToChatRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UsersId       []*UserId              `protobuf:"bytes,1,rep,name=UsersId,proto3" json:"UsersId,omitempty"`              // ID пользователей
	ChatId        int64                  `protobuf:"varint,2,opt,name=chat_id,json=chatId,proto3" json:"chat_id,omitempty"` // ID чата
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddUsersToChatRequest) Reset() {
	*x = AddUsersToChatRequest{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddUsersToChatRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddUsersToChatRequest) ProtoMessage() {}

func (x *AddUsersToChatRequest) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddUsersToChatRequest.ProtoReflect.Descriptor instead.
func (*AddUsersToChatRequest) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{3}
}

func (x *AddUsersToChatRequest) GetUsersId() []*UserId {
	if x != nil {
		return x.UsersId
	}
	return nil
}

func (x *AddUsersToChatRequest) GetChatId() int64 {
	if x != nil {
		return x.ChatId
	}
	return 0
}

type AddUsersToChatResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Message       string                 `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"` // Ответ от сервера
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddUsersToChatResponse) Reset() {
	*x = AddUsersToChatResponse{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddUsersToChatResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddUsersToChatResponse) ProtoMessage() {}

func (x *AddUsersToChatResponse) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddUsersToChatResponse.ProtoReflect.Descriptor instead.
func (*AddUsersToChatResponse) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{4}
}

func (x *AddUsersToChatResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type ConnectUsersToChat struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	UsersId       []*UserId              `protobuf:"bytes,1,rep,name=UsersId,proto3" json:"UsersId,omitempty"`              // ID пользователей
	ChatId        int64                  `protobuf:"varint,2,opt,name=chat_id,json=chatId,proto3" json:"chat_id,omitempty"` // ID чата
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ConnectUsersToChat) Reset() {
	*x = ConnectUsersToChat{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ConnectUsersToChat) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConnectUsersToChat) ProtoMessage() {}

func (x *ConnectUsersToChat) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConnectUsersToChat.ProtoReflect.Descriptor instead.
func (*ConnectUsersToChat) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{5}
}

func (x *ConnectUsersToChat) GetUsersId() []*UserId {
	if x != nil {
		return x.UsersId
	}
	return nil
}

func (x *ConnectUsersToChat) GetChatId() int64 {
	if x != nil {
		return x.ChatId
	}
	return 0
}

type SaveMessageRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ChatId        int64                  `protobuf:"varint,1,opt,name=chat_id,json=chatId,proto3" json:"chat_id,omitempty"` // Идентификатор чата
	Content       string                 `protobuf:"bytes,2,opt,name=content,proto3" json:"content,omitempty"`              // Содержимое сообщения
	Time          *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=time,proto3" json:"time,omitempty"`                    // Временная метка
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SaveMessageRequest) Reset() {
	*x = SaveMessageRequest{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SaveMessageRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaveMessageRequest) ProtoMessage() {}

func (x *SaveMessageRequest) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SaveMessageRequest.ProtoReflect.Descriptor instead.
func (*SaveMessageRequest) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{6}
}

func (x *SaveMessageRequest) GetChatId() int64 {
	if x != nil {
		return x.ChatId
	}
	return 0
}

func (x *SaveMessageRequest) GetContent() string {
	if x != nil {
		return x.Content
	}
	return ""
}

func (x *SaveMessageRequest) GetTime() *timestamppb.Timestamp {
	if x != nil {
		return x.Time
	}
	return nil
}

type SaveMessageResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	MessageId     int64                  `protobuf:"varint,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"` // ID сохранённого сообщения
	ChatId        int64                  `protobuf:"varint,2,opt,name=chat_id,json=chatId,proto3" json:"chat_id,omitempty"`          // ID чата
	Message       string                 `protobuf:"bytes,4,opt,name=message,proto3" json:"message,omitempty"`                       // Текст сообщения
	CreatedAt     int64                  `protobuf:"varint,5,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"` // Время создания сообщения (UNIX timestamp)
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SaveMessageResponse) Reset() {
	*x = SaveMessageResponse{}
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SaveMessageResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SaveMessageResponse) ProtoMessage() {}

func (x *SaveMessageResponse) ProtoReflect() protoreflect.Message {
	mi := &file_dbservice_proto_dbchat_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SaveMessageResponse.ProtoReflect.Descriptor instead.
func (*SaveMessageResponse) Descriptor() ([]byte, []int) {
	return file_dbservice_proto_dbchat_proto_rawDescGZIP(), []int{7}
}

func (x *SaveMessageResponse) GetMessageId() int64 {
	if x != nil {
		return x.MessageId
	}
	return 0
}

func (x *SaveMessageResponse) GetChatId() int64 {
	if x != nil {
		return x.ChatId
	}
	return 0
}

func (x *SaveMessageResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *SaveMessageResponse) GetCreatedAt() int64 {
	if x != nil {
		return x.CreatedAt
	}
	return 0
}

var File_dbservice_proto_dbchat_proto protoreflect.FileDescriptor

var file_dbservice_proto_dbchat_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x64, 0x62, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x2f, 0x64, 0x62, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x66, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3a, 0x0a, 0x06, 0x55, 0x73,
	0x65, 0x72, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x75, 0x73, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x12, 0x17, 0x0a,
	0x07, 0x72, 0x6f, 0x6c, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06,
	0x72, 0x6f, 0x6c, 0x65, 0x49, 0x64, 0x22, 0x5e, 0x0a, 0x11, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x43, 0x68, 0x61, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2c, 0x0a, 0x08, 0x75,
	0x73, 0x65, 0x72, 0x73, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x66, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x49, 0x64,
	0x52, 0x07, 0x75, 0x73, 0x65, 0x72, 0x73, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x63, 0x68, 0x61,
	0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x63, 0x68,
	0x61, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x66, 0x0a, 0x12, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x43, 0x68, 0x61, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x74, 0x5f, 0x69,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x63, 0x68, 0x61, 0x74, 0x49, 0x64, 0x12,
	0x1d, 0x0a, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x22, 0x5d,
	0x0a, 0x15, 0x61, 0x64, 0x64, 0x55, 0x73, 0x65, 0x72, 0x73, 0x54, 0x6f, 0x43, 0x68, 0x61, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2b, 0x0a, 0x07, 0x55, 0x73, 0x65, 0x72, 0x73,
	0x49, 0x64, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x66, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x49, 0x64, 0x52, 0x07, 0x55, 0x73, 0x65,
	0x72, 0x73, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x74, 0x5f, 0x69, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x63, 0x68, 0x61, 0x74, 0x49, 0x64, 0x22, 0x32, 0x0a,
	0x16, 0x61, 0x64, 0x64, 0x55, 0x73, 0x65, 0x72, 0x73, 0x54, 0x6f, 0x43, 0x68, 0x61, 0x74, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x22, 0x5a, 0x0a, 0x12, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x55, 0x73, 0x65, 0x72,
	0x73, 0x54, 0x6f, 0x43, 0x68, 0x61, 0x74, 0x12, 0x2b, 0x0a, 0x07, 0x55, 0x73, 0x65, 0x72, 0x73,
	0x49, 0x64, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x66, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x49, 0x64, 0x52, 0x07, 0x55, 0x73, 0x65,
	0x72, 0x73, 0x49, 0x64, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x74, 0x5f, 0x69, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x63, 0x68, 0x61, 0x74, 0x49, 0x64, 0x22, 0x77, 0x0a,
	0x12, 0x53, 0x61, 0x76, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x68, 0x61, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x63, 0x68, 0x61, 0x74, 0x49, 0x64, 0x12, 0x18, 0x0a, 0x07,
	0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63,
	0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x12, 0x2e, 0x0a, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x04, 0x74, 0x69, 0x6d, 0x65, 0x22, 0x86, 0x01, 0x0a, 0x13, 0x53, 0x61, 0x76, 0x65, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1d,
	0x0a, 0x0a, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x03, 0x52, 0x09, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x12, 0x17, 0x0a,
	0x07, 0x63, 0x68, 0x61, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06,
	0x63, 0x68, 0x61, 0x74, 0x49, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x05,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x32,
	0xa8, 0x01, 0x0a, 0x0d, 0x64, 0x62, 0x43, 0x68, 0x61, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x12, 0x49, 0x0a, 0x0a, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x43, 0x68, 0x61, 0x74, 0x12,
	0x1c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x66, 0x2e, 0x43, 0x72, 0x65, 0x61,
	0x74, 0x65, 0x43, 0x68, 0x61, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1d, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x66, 0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x43, 0x68, 0x61, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4c, 0x0a, 0x0b,
	0x53, 0x61, 0x76, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x1d, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x66, 0x2e, 0x53, 0x61, 0x76, 0x65, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x66, 0x2e, 0x53, 0x61, 0x76, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x12, 0x5a, 0x10, 0x2e, 0x2f,
	0x64, 0x62, 0x63, 0x68, 0x61, 0x74, 0x2f, 0x3b, 0x64, 0x62, 0x63, 0x68, 0x61, 0x74, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_dbservice_proto_dbchat_proto_rawDescOnce sync.Once
	file_dbservice_proto_dbchat_proto_rawDescData = file_dbservice_proto_dbchat_proto_rawDesc
)

func file_dbservice_proto_dbchat_proto_rawDescGZIP() []byte {
	file_dbservice_proto_dbchat_proto_rawDescOnce.Do(func() {
		file_dbservice_proto_dbchat_proto_rawDescData = protoimpl.X.CompressGZIP(file_dbservice_proto_dbchat_proto_rawDescData)
	})
	return file_dbservice_proto_dbchat_proto_rawDescData
}

var file_dbservice_proto_dbchat_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_dbservice_proto_dbchat_proto_goTypes = []any{
	(*UserId)(nil),                 // 0: protobuff.UserId
	(*CreateChatRequest)(nil),      // 1: protobuff.CreateChatRequest
	(*CreateChatResponse)(nil),     // 2: protobuff.CreateChatResponse
	(*AddUsersToChatRequest)(nil),  // 3: protobuff.addUsersToChatRequest
	(*AddUsersToChatResponse)(nil), // 4: protobuff.addUsersToChatResponse
	(*ConnectUsersToChat)(nil),     // 5: protobuff.ConnectUsersToChat
	(*SaveMessageRequest)(nil),     // 6: protobuff.SaveMessageRequest
	(*SaveMessageResponse)(nil),    // 7: protobuff.SaveMessageResponse
	(*timestamppb.Timestamp)(nil),  // 8: google.protobuf.Timestamp
}
var file_dbservice_proto_dbchat_proto_depIdxs = []int32{
	0, // 0: protobuff.CreateChatRequest.users_id:type_name -> protobuff.UserId
	0, // 1: protobuff.addUsersToChatRequest.UsersId:type_name -> protobuff.UserId
	0, // 2: protobuff.ConnectUsersToChat.UsersId:type_name -> protobuff.UserId
	8, // 3: protobuff.SaveMessageRequest.time:type_name -> google.protobuf.Timestamp
	1, // 4: protobuff.dbChatService.CreateChat:input_type -> protobuff.CreateChatRequest
	6, // 5: protobuff.dbChatService.SaveMessage:input_type -> protobuff.SaveMessageRequest
	2, // 6: protobuff.dbChatService.CreateChat:output_type -> protobuff.CreateChatResponse
	7, // 7: protobuff.dbChatService.SaveMessage:output_type -> protobuff.SaveMessageResponse
	6, // [6:8] is the sub-list for method output_type
	4, // [4:6] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_dbservice_proto_dbchat_proto_init() }
func file_dbservice_proto_dbchat_proto_init() {
	if File_dbservice_proto_dbchat_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_dbservice_proto_dbchat_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_dbservice_proto_dbchat_proto_goTypes,
		DependencyIndexes: file_dbservice_proto_dbchat_proto_depIdxs,
		MessageInfos:      file_dbservice_proto_dbchat_proto_msgTypes,
	}.Build()
	File_dbservice_proto_dbchat_proto = out.File
	file_dbservice_proto_dbchat_proto_rawDesc = nil
	file_dbservice_proto_dbchat_proto_goTypes = nil
	file_dbservice_proto_dbchat_proto_depIdxs = nil
}
