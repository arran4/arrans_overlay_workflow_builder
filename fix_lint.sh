#!/bin/bash
sed -i -e '/for _, f := range ggbtd.IUseFlags() {/,/}/c\
	pkgMd.Use[0].Flags = append(pkgMd.Use[0].Flags, ggbtd.IUseFlags()...)
' generateGithubBinaryWorkflow.go
