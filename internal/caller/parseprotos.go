package caller

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/avysochin256/grpc-client-cli/internal/descwrap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

var errNoProtoFilesFound = errors.New("no proto files found")

func parseProtoFiles(protoDirs []string, protoImports []string) ([]*descwrap.FileDescriptor, error) {
	protofiles, err := findProtoFiles(protoDirs)
	if err != nil {
		return nil, err
	}

	if len(protofiles) == 0 {
		return nil, fmt.Errorf("%w: %s", errNoProtoFilesFound, protoDirs)
	}

	importPaths := []string{}
	for _, pd := range protoDirs {
		if path.Ext(pd) != "" {
			pd = path.Dir(pd)
		}

		importPaths = append(importPaths, pd)
	}

	importPaths = append(importPaths, protoImports...)

	tmp, err := os.CreateTemp("", "descset")
	if err != nil {
		return nil, err
	}
	tmp.Close()
	args := []string{}
	for _, ip := range importPaths {
		args = append(args, "-I", ip)
	}
	args = append(args, "--include_imports", "--descriptor_set_out="+tmp.Name())
	args = append(args, protofiles...)
	cmd := exec.Command("protoc", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(tmp.Name())
	os.Remove(tmp.Name())
	if err != nil {
		return nil, err
	}
	var set descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(b, &set); err != nil {
		return nil, err
	}
	res := make([]*descwrap.FileDescriptor, len(set.File))
	for i, fd := range set.File {
		f, err := protodesc.NewFile(fd, nil)
		if err != nil {
			return nil, err
		}
		res[i] = descwrap.WrapFile(f)
	}
	return res, nil
}

func findProtoFiles(paths []string) ([]string, error) {
	protofiles := []string{}
	for _, p := range paths {
		ext := path.Ext(p)
		if ext == ".proto" {
			protofiles = append(protofiles, filepath.Base(p))
			continue
		}

		// non proto extension - skip
		if ext != "" {
			continue
		}

		err := filepath.Walk(p, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if filepath.Ext(info.Name()) == ".proto" {
				protofiles = append(protofiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return protofiles, nil
}
