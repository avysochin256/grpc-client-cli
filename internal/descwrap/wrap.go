package descwrap

import "google.golang.org/protobuf/reflect/protoreflect"

// FileDescriptor wraps protoreflect.FileDescriptor.
type FileDescriptor struct{ FD protoreflect.FileDescriptor }

func WrapFile(fd protoreflect.FileDescriptor) *FileDescriptor { return &FileDescriptor{fd} }

func (f *FileDescriptor) GetServices() []*ServiceDescriptor {
	svcs := make([]*ServiceDescriptor, f.FD.Services().Len())
	for i := 0; i < f.FD.Services().Len(); i++ {
		svcs[i] = &ServiceDescriptor{f.FD.Services().Get(i)}
	}
	return svcs
}

func (f *FileDescriptor) GetMessageTypes() []*MessageDescriptor {
	msgs := make([]*MessageDescriptor, f.FD.Messages().Len())
	for i := 0; i < f.FD.Messages().Len(); i++ {
		msgs[i] = &MessageDescriptor{f.FD.Messages().Get(i)}
	}
	return msgs
}

func (f *FileDescriptor) UnwrapFile() protoreflect.FileDescriptor { return f.FD }

// ServiceDescriptor wraps protoreflect.ServiceDescriptor.
type ServiceDescriptor struct {
	SD protoreflect.ServiceDescriptor
}

func (s *ServiceDescriptor) GetFullyQualifiedName() string { return string(s.SD.FullName()) }

func (s *ServiceDescriptor) GetName() string { return string(s.SD.Name()) }

func (s *ServiceDescriptor) GetFile() *FileDescriptor { return &FileDescriptor{s.SD.ParentFile()} }

func (s *ServiceDescriptor) GetMethods() []*MethodDescriptor {
	m := make([]*MethodDescriptor, s.SD.Methods().Len())
	for i := 0; i < s.SD.Methods().Len(); i++ {
		m[i] = &MethodDescriptor{s.SD.Methods().Get(i)}
	}
	return m
}

// MethodDescriptor wraps protoreflect.MethodDescriptor.
type MethodDescriptor struct{ MD protoreflect.MethodDescriptor }

func (m *MethodDescriptor) GetName() string { return string(m.MD.Name()) }

func (m *MethodDescriptor) GetService() *ServiceDescriptor {
	return &ServiceDescriptor{m.MD.Parent().(protoreflect.ServiceDescriptor)}
}

func (m *MethodDescriptor) IsServerStreaming() bool { return m.MD.IsStreamingServer() }

func (m *MethodDescriptor) IsClientStreaming() bool { return m.MD.IsStreamingClient() }

func (m *MethodDescriptor) GetInputType() *MessageDescriptor { return &MessageDescriptor{m.MD.Input()} }

func (m *MethodDescriptor) GetOutputType() *MessageDescriptor {
	return &MessageDescriptor{m.MD.Output()}
}

// MessageDescriptor wraps protoreflect.MessageDescriptor.
type MessageDescriptor struct {
	MD protoreflect.MessageDescriptor
}

func (m *MessageDescriptor) UnwrapMessage() protoreflect.MessageDescriptor { return m.MD }
