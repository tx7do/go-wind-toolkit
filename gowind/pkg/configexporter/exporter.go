package configexporter

import (
	"errors"
	"log"

	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal/consul"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal/etcd"
	"github.com/tx7do/go-wind-toolkit/gowind/pkg/configexporter/internal/nacos"
)

func NewExporter(
	typeName string,
	endpoint string,
	prefix string,
	projectRootPath string,
	group string,
	env string,
	namespaceId string,
	mergeSingle bool,
) internal.Exporter {
	opts := &internal.Options{
		Service:     internal.ImporterType(typeName),
		Endpoint:    endpoint,
		ProjectName: prefix,
		ProjectRoot: projectRootPath,
		Group:       group,
		Env:         env,
		NamespaceId: namespaceId,
		MergeSingle: mergeSingle,
	}

	switch opts.Service {
	default:
		log.Fatalf("Unsupported exporter type: %s", opts.Service)
		return nil // Unsupported type

	case internal.Apollo:
		return nil

	case internal.Consul:
		return consul.NewExporter(opts)

	case internal.Etcd:
		return etcd.NewExporter(opts)

	case internal.Kubernetes:
		return nil

	case internal.Nacos:
		return nacos.NewExporter(opts)

	case internal.Polaris:
		return nil
	}
}

func Export(
	typeName string,
	endpoint string,
	prefix string,
	projectRootPath string,
	group string,
	env string,
	namespaceId string,
	mergeSingle bool,
) error {
	exporter := NewExporter(
		typeName,
		endpoint,
		prefix,
		projectRootPath,
		group,
		env,
		namespaceId,
		mergeSingle,
	)
	if exporter == nil {
		return errors.New("exporter is nil")
	}
	return exporter.Export()
}
