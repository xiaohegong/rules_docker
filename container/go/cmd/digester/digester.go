// Copyright 2017 The Bazel Authors. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//    http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License/
////////////////////////////////////
//This binary implements the ability to load a docker image, calculate its image manifest sha256 hash and output a digest file.
// It expects to be run with:
//     extract_config -tarball=image.tar -output=output.confi
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/bazelbuild/rules_docker/container/go/pkg/compat"
	"github.com/bazelbuild/rules_docker/container/go/pkg/utils"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
)

var (
	dst    = flag.String("dst", "", "The destination location of the digest file to write to.")
	src    = flag.String("src", "", "Path to the config.json when -format is legacy, path to the index.json when -format is oci or path to the image .tar file when -format is docker.")
	format = flag.String("format", "", "The format of the source image, (oci, legacy, or docker). The docker format should be a tarball of the image as generated by docker save.")
	layers utils.ArrayFlags
)

const (
	// manifestFile is the filename of image manifest.
	manifestFile = "manifest.json"
	// configFile is the filename of image config.
	configFile = "config.json"
	// indexManifestFile is the filename of image manifest config in OCI format.
	indexManifestFile = "index.json"
)

func main() {
	flag.Var(&layers, "layers", "The list of paths to the layers of this docker image, only used for legacy images.")
	flag.Parse()
	log.Println("Running the Image Digester to generate an image digest file...")

	if *dst == "" {
		log.Fatalln("Required option -dst was not specified.")
	}
	if *src == "" {
		log.Fatalln("Required option -src was not specified.")
	}
	if *format == "" {
		log.Fatalln("Required option -format was not specified.")
	}

	// Validates provided format and src path. Check if src is a tarball when pushing a docker image. Trim basename index.json or manifest.json if src is a directory, since we are pushing a OCI/legacy index.
	var imgSrc string
	if *format == "docker" && filepath.Ext(*src) != ".tar" {
		log.Fatalf("Invalid value for argument -src for -format=docker, got %q, want path to tarball file with extension .tar.", *src)
	}
	if *format == "legacy" && filepath.Base(*src) != configFile {
		log.Fatalf("Invalid value for argument -src for -format=legacy, got %q, want path to %s", *src, configFile)
	}
	if *format == "oci" && filepath.Base(*src) != indexManifestFile {
		log.Fatalf("Invalid value for argument -src for -format=oci, got %q, want path to %s", *src, indexManifestFile)
	}
	if *format == "oci" || *format == "legacy" {
		imgSrc = filepath.Dir(*src)
		log.Printf("Determined image source path to be %s based on -format=%s, -src=%s.", imgSrc, *format, *src)
	}
	if *format == "docker" {
		imgSrc = *src
	}
	if *format == "legacy" {
		manifestPath := filepath.Join(imgSrc, manifestFile)

		// TODO (xiaohegong): remove generate manifest after createImageConfig is merged.
		log.Printf("Generating image manifest to %s...", manifestPath)
		_, err := compat.GenerateManifest(imgSrc, manifestPath, filepath.Join(imgSrc, configFile), layers)
		if err != nil {
			log.Fatalf("Error generating %s from %s: %v", manifestFile, imgSrc, err)
		}
	}

	img, err := utils.ReadImage(imgSrc, *format)
	if err != nil {
		log.Fatalf("Error reading from %s: %v", imgSrc, err)
	}

	digest, err := img.Digest()
	if err != nil {
		log.Fatalf("Error getting image digest: %v", err)
	}

	err = WriteDigest(digest, *dst)
	if err != nil {
		log.Fatalf("Error outputting digest file to %s: %v", *dst, err)
	}

	log.Printf("Successfully generated image digest file at %s", *dst)
}

// WriteDigest outputs digest to a "digest file" at dst. Digest is the image manifest's sha256 hash.
func WriteDigest(digest v1.Hash, dst string) error {
	rawDigest := []byte(digest.Algorithm + ":" + digest.Hex)

	err := ioutil.WriteFile(dst, rawDigest, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "unable to write digest file to %s", dst)
	}

	return nil
}
