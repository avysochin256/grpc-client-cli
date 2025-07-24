package caller

import (
	"errors"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoiface"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var customAnyDescr protoreflect.MessageDescriptor

func init() {
	md := &descriptorpb.DescriptorProto{
		Name: proto.String("any"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("err"),
				Number: proto.Int32(1),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			},
		},
	}
	fd := &descriptorpb.FileDescriptorProto{
		Syntax:      proto.String("proto3"),
		Name:        proto.String("any.proto"),
		MessageType: []*descriptorpb.DescriptorProto{md},
	}
	f, err := protodesc.NewFile(fd, nil)
	if err != nil {
		panic(err)
	}
	customAnyDescr = f.Messages().Get(0)
}

type protoResolver struct {
	*dynamicpb.Types
}

func newResolver() *protoResolver {
	return &protoResolver{dynamicpb.NewTypes(protoregistry.GlobalFiles)}
}

// FindMessageByURL is being called when Any @type needs to be resolved
func (t *protoResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	mt, err := t.Types.FindMessageByURL(url)
	if err != nil {
		if errors.Is(err, protoregistry.NotFound) {
			msg := dynamicpb.NewMessage(customAnyDescr)
			return &unknownMsgType{msg}, nil
		}
		return mt, err
	}
	return mt, nil
}

// unknownMsgType represents protoreflect.MessageType of unknownMsg
type unknownMsgType struct {
	m *dynamicpb.Message
}

func (u *unknownMsgType) New() protoreflect.Message {
	return &unknownMsg{u.m}
}

func (u *unknownMsgType) Zero() protoreflect.Message {
	return &unknownMsg{u.m}
}

func (u *unknownMsgType) Descriptor() protoreflect.MessageDescriptor {
	return u.m.Descriptor()
}

// unknownMsg is used when underlying type of google.protobuf.Any type cannot be resolved
type unknownMsg struct {
	*dynamicpb.Message
}

func (m *unknownMsg) Interface() protoreflect.ProtoMessage {
	return m
}

func (m *unknownMsg) ProtoReflect() protoreflect.Message {
	return m
}

func (a *unknownMsg) ProtoMethods() *protoiface.Methods {
	return &protoiface.Methods{
		Unmarshal: func(in protoiface.UnmarshalInput) (protoiface.UnmarshalOutput, error) {
			if msg, ok := in.Message.(*unknownMsg); ok {
				msg.Set(
					msg.Descriptor().Fields().ByName("err"),
					protoreflect.ValueOfString("type not found"))
			}
			return protoiface.UnmarshalOutput{}, nil
		},
	}
}
