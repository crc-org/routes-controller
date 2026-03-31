// Package tools tracks dependencies for build tools
//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
)
