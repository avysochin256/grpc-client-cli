package caller

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/avysochin256/grpc-client-cli/internal/descwrap"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func init() {
	// don't panic on proto registration conflicts, return errors instead
	os.Setenv("GOLANG_PROTOBUF_REGISTRATION_CONFLICT", "warn")
}

type ServiceMetaData interface {
	GetServiceMetaDataList(context.Context) (ServiceMetaList, error)
	GetAdditionalFiles() ([]*descwrap.FileDescriptor, error)
}

type ServiceMeta struct {
	Name    string
	Methods []*descwrap.MethodDescriptor
	File    *descwrap.FileDescriptor
}

type ServiceMetaList []*ServiceMeta

func (l ServiceMetaList) Files() []*descwrap.FileDescriptor {
	res := make([]*descwrap.FileDescriptor, len(l))
	for i, m := range l {
		res[i] = m.File
	}

	return res
}

type serviceMetaBase struct{}

func (s serviceMetaBase) GetAdditionalFiles(protoImports []string) ([]*descwrap.FileDescriptor, error) {
	if len(protoImports) == 0 {
		return nil, nil
	}
	fileDesc, err := parseProtoFiles(protoImports, nil)
	if err != nil {
		if errors.Is(err, errNoProtoFilesFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error parsing additional proto files: %w", err)
	}
	return fileDesc, nil
}

func RegisterFiles(fds ...*descwrap.FileDescriptor) error {
	errs := []error{}
	for _, fd := range fds {
		protoFile := fd.UnwrapFile()
		_, err := protoregistry.GlobalFiles.FindFileByPath(protoFile.Path())
		if errors.Is(err, protoregistry.NotFound) && shouldRegister(protoFile) {
			if err := protoregistry.GlobalFiles.RegisterFile(protoFile); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func shouldRegister(fd protoreflect.FileDescriptor) bool {
	for i := 0; i < fd.Messages().Len(); i++ {
		msg := fd.Messages().Get(i)
		_, err := protoregistry.GlobalTypes.FindMessageByURL(string(msg.FullName()))
		if errors.Is(err, protoregistry.NotFound) {
			return true
		}
	}

	return false
}
