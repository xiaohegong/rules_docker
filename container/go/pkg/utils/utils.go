package utils

import (
	"github.com/bazelbuild/rules_docker/container/go/pkg/compat"
	"github.com/bazelbuild/rules_docker/container/go/pkg/oci"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/pkg/errors"
)

// ReadImage returns a v1.Image after reading an legacy layout, an OCI layout or a Docker tarball from src.
func ReadImage(src, format string) (v1.Image, error) {
	if format == "oci" {
		return oci.Read(src)
	}
	if format == "legacy" {
		return compat.Read(src)
	}
	if format == "docker" {
		return tarball.ImageFromPath(src, nil)
	}

	return nil, errors.Errorf("unknown image format %q", format)
}
