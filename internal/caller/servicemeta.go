package caller

import (
	"context"
	"time"

	"github.com/avysochin256/grpc-client-cli/internal/descwrap"
	refl "github.com/avysochin256/grpc-client-cli/internal/reflection"
	"github.com/avysochin256/grpc-client-cli/internal/rpc"
	"google.golang.org/grpc"
)

type serviceMetaData struct {
	connFact       *rpc.GrpcConnFactory
	target         string
	deadline       int
	protoImports   []string
	reflectVersion GrpcReflectVersion

	serviceMetaBase
}

type ServiceMetaDataConfig struct {
	ConnFact       *rpc.GrpcConnFactory
	Target         string
	ProtoImports   []string
	Deadline       int
	ReflectVersion GrpcReflectVersion
}

// NewServiceMetaData returns new instance of ServiceMetaData
// that reads service metadata by calling grpc Reflection service of the target
func NewServiceMetaData(cfg *ServiceMetaDataConfig) ServiceMetaData {
	return &serviceMetaData{
		connFact:       cfg.ConnFact,
		target:         cfg.Target,
		deadline:       cfg.Deadline,
		protoImports:   cfg.ProtoImports,
		reflectVersion: cfg.ReflectVersion,
	}
}

func (s *serviceMetaData) GetServiceMetaDataList(ctx context.Context) (ServiceMetaList, error) {
	conn, err := s.connFact.GetConn(s.target)
	if err != nil {
		return nil, err
	}
	callctx, cancel := context.WithTimeout(ctx, time.Duration(s.deadline)*time.Second)
	defer cancel()
	rc := s.grpcReflectClient(callctx, conn)

	services, err := rc.ListServices()
	if err != nil {
		defer rc.Reset()
		return nil, err
	}

	res := make([]*ServiceMeta, len(services))
	for i, svc := range services {
		svcDesc, err := rc.ResolveService(svc)
		// sometimes ResolveService throws an error
		// when different proto files have different dependency protos named identically
		// For example service1.proto has common_types.proto and service2.proto has the same dependency
		// protoreflect library caches dependencies by name
		// so if we get an error, we can just recreate Client to reset internal cache and try again
		if err != nil {
			rc.Reset()
			// try only once here
			rc = s.grpcReflectClient(callctx, conn)
			svcDesc, err = rc.ResolveService(svc)
			if err != nil {
				defer rc.Reset()
				return nil, err
			}
		}

		svcData := &ServiceMeta{
			File: descwrap.WrapFile(svcDesc.ParentFile()),
			Name: string(svcDesc.FullName()),
		}
		methods := make([]*descwrap.MethodDescriptor, svcDesc.Methods().Len())
		for j := 0; j < svcDesc.Methods().Len(); j++ {
			methods[j] = &descwrap.MethodDescriptor{MD: svcDesc.Methods().Get(j)}
		}
		svcData.Methods = methods

		for _, m := range svcData.Methods {
			u := newJsonNamesUpdater()
			u.updateJSONNames(m.GetInputType().UnwrapMessage())
			u.updateJSONNames(m.GetOutputType().UnwrapMessage())
		}
		res[i] = svcData
	}

	defer rc.Reset()
	return res, nil
}

func (s *serviceMetaData) GetAdditionalFiles() ([]*descwrap.FileDescriptor, error) {
	return s.serviceMetaBase.GetAdditionalFiles(s.protoImports)
}

func (s *serviceMetaData) grpcReflectClient(ctx context.Context, conn grpc.ClientConnInterface) *refl.Client {
	if s.reflectVersion == GrpcReflectAuto {
		c, _ := refl.NewClient(ctx, conn, refl.Auto)
		return c
	}
	c, _ := refl.NewClient(ctx, conn, refl.V1Alpha)
	return c
}
