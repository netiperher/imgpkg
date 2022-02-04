// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	goui "github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/api"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
)

var (
	// DescribeOutputType Possible output options
	DescribeOutputType = []string{"text", "yaml"}
)

// DescribeOptions Command Line options that can be provided to the describe command
type DescribeOptions struct {
	ui goui.UI

	BundleFlags   BundleFlags
	RegistryFlags RegistryFlags

	Concurrency int
	OutputType  string
}

// NewDescribeOptions constructor for building a DescribeOptions, holding values derived via flags
func NewDescribeOptions(ui *goui.ConfUI) *DescribeOptions {
	return &DescribeOptions{ui: ui}
}

// NewDescribeCmd constructor for the describe command
func NewDescribeCmd(o *DescribeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe the images and bundles associated with a give bundle",
		RunE:  func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: `
    # Describe a bundle
    imgpkg describe -b carvel.dev/app1-bundle`,
	}

	o.BundleFlags.SetCopy(cmd)
	o.RegistryFlags.Set(cmd)
	cmd.Flags().IntVar(&o.Concurrency, "concurrency", 5, "Concurrency")
	cmd.Flags().StringVarP(&o.OutputType, "output-type", "o", "text", "Type of output possible values: [text, yaml]")
	return cmd
}

// Run functions called when the describe command is provided in the command line
func (d *DescribeOptions) Run() error {
	err := d.validateFlags()
	if err != nil {
		return err
	}

	levelLogger := util.NewUILevelLogger(util.LogWarn, d.ui)
	description, err := api.DescribeBundle(
		d.BundleFlags.Bundle,
		api.DescribeOpts{
			Logger:      levelLogger,
			Concurrency: d.Concurrency,
		},
		d.RegistryFlags.AsRegistryOpts())
	if err != nil {
		return err
	}

	if d.OutputType == "text" {
		p := bundleTextPrinter{ui: d.ui}
		p.Print(description)
	}
	return nil
}

func (d *DescribeOptions) validateFlags() error {
	outputType := ""
	for _, s := range DescribeOutputType {
		if s == d.OutputType {
			outputType = s
			break
		}
	}
	if outputType == "" {
		return fmt.Errorf("--output-type can only have the following values [text, yaml]")
	}
	return nil
}

// Bundle SHA: aaaaad700949154e429d28661d01c99d53a38af0d5275842ccbf0bf6dbef8ca4
//Tags: latest, v1.0.0
//
//Authors:
//  Carvel Team <carvel@vmware.com>
//Websites:
//  carvel.dev/imgpkg
//Metadata:
//  - Some Version: 1.0.0
//  - Other Information: Some text here
//
//Images:
//  - Image: new.registry.io/simple-app-install-package@sha256:d211dd700949154e429d28661d01c99d53a38af0d5275842ccbf0bf6dbef8ca4
//    Type: Bundle
//    Origin: my.registry.io/bundle1@sha256:d211dd700949154e429d28661d01c99d53a38af0d5275842ccbf0bf6dbef8ca4
//    Images:
//      - Image: new.registry.io/simple-app-install-package@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
//        Type: Image
//        Origin: registry.io/img1@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
//        Annotations:
//          kbld.carvel.dev/id: my.registry.io/simple-application
//
//      - Image: new.registry.io/simple-app-install-package@sha256:47ae428a887c41ba0aedf87d560eb305a8aa522ffb80ac1c96a37b16df038e0f
//        Type: Image
//        Origin: registry.io/img2@sha256:47ae428a887c41ba0aedf87d560eb305a8aa522ffb80ac1c96a37b16df038e0f
//  - Image: new.registry.io/simple-app-install-package@sha256:47ae428a887c41ba0aedf87d560eb305a8aa522ffb80ac1c96a37b16df038e0f
//    Type: Image
//    Origin: registry.io/img2@sha256:47ae428a887c41ba0aedf87d560eb305a8aa522ffb80ac1c96a37b16df038e0f

// ./imgpkg describe -b localhost:5000/describe-test-not-collocated@sha256:f35a6d5e5596919c6bd4f62164ee6f8ccd919d0d8a04b3a5fb382af33dd7da9d
// ./imgpkg describe -b localhost:5000/describe-test-collocated@sha256:f35a6d5e5596919c6bd4f62164ee6f8ccd919d0d8a04b3a5fb382af33dd7da9d
type bundleTextPrinter struct {
	ui goui.UI
}

func (p bundleTextPrinter) Print(description api.BundleDescription) {
	logger := util.NewUIPrefixedWriter("", p.ui)
	bundleRef, err := regname.ParseReference(description.Image)
	if err != nil {
		panic(fmt.Sprintf("Internal consistency: expected %s to be a digest reference", description.Image))
	}
	logger.BeginLinef("Bundle SHA: %s\n", bundleRef.Identifier())

	logger.BeginLinef("\n")
	p.printerRec(description, p.ui)
}

func (p bundleTextPrinter) printerRec(description api.BundleDescription, logger goui.UI) {
	logger.BeginLinef("Images:\n")
	indentLogger := goui.NewIndentingUI(logger)
	for _, b := range description.Content.Bundles {
		indentLogger.BeginLinef("Image: %s\n", b.Image)
		indentLogger.BeginLinef("Type: Bundle\n")
		indentLogger.BeginLinef("Origin: %s\n", b.Origin)
		p.printerRec(b, indentLogger)
	}

	for _, image := range description.Content.Images {
		indentLogger.BeginLinef("Image: %s\n", image.Image)
		indentLogger.BeginLinef("Type: Image\n")
		indentLogger.BeginLinef("Origin: %s\n", image.Origin)
	}
}
