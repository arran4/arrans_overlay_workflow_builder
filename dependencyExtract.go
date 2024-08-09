package arrans_overlay_workflow_builder

import (
	"debug/elf"
	"fmt"
	"io"
	"log"
	"os"
)

func ReadDependencies(file string, program *Program) ([]string, error) {
	unknownSymbols := []string{}
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("opening file for symbols: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("Error closing %s: %s", file, err)
		}
	}()
	unknownSymbols, err = ReadDependenciesFromReader(program, f, unknownSymbols)
	if err != nil {
		return nil, fmt.Errorf("file %s: %w", file, err)
	}
	return unknownSymbols, err
}

func ReadDependenciesFromReader(program *Program, f io.ReaderAt, unknownSymbols []string) ([]string, error) {
	e, err := elf.NewFile(f)
	if err != nil {
		return nil, fmt.Errorf("reading elf: %w", err)
	}
	defer func() {
		if err := e.Close(); err != nil {
			log.Printf("Error closing elf: %s", err)
		}
	}()
	importedLibraries, err := e.ImportedLibraries()
	if err != nil {
		return nil, fmt.Errorf("reading imported libraries: %w", err)
	}
	libraries := make(map[string]struct{}, len(importedLibraries))
	for _, symbol := range importedLibraries {
		if symbol == "" {
			continue
		}
		libraries[symbol] = struct{}{}
	}
	addedDeps := map[string]struct{}{}
	for library := range libraries {
		dep, ok := lookupSymbol(library)
		if !ok {
			unknownSymbols = append(unknownSymbols, library)
		}
		if dep == "" {
			continue
		}
		if _, ok := addedDeps[dep]; ok {
			continue
		}
		program.Dependencies = append(program.Dependencies, dep)
		addedDeps[dep] = struct{}{}
	}

	return unknownSymbols, nil
}

var (
	// Once you have installed the correct dep use `equery b <name>` to determine which package if you're unsure
	symbolMap = map[string]string{
		"libpthread.so.0":           "sys-libs/glibc",
		"libpthread.so":             "sys-libs/glibc",
		"libc.so.6":                 "sys-libs/glibc",
		"libdl.so.2":                "sys-libs/glibc",
		"libdl.so":                  "sys-libs/glibc",
		"libc.so":                   "sys-libs/glibc",
		"ld-linux-x86-64.so.1":      "sys-libs/glibc",
		"ld-linux-x86-64.so.2":      "sys-libs/glibc",
		"ld-linux-x86-64.so":        "sys-libs/glibc",
		"ld-linux-aarch64.so":       "sys-libs/glibc",
		"ld-linux-aarch64.so.1":     "sys-libs/glibc",
		"ld-linux-aarch64.so.2":     "sys-libs/glibc",
		"ld-linux-armv7.so":         "sys-libs/glibc",
		"ld-linux-armv7.so.1":       "sys-libs/glibc",
		"ld-linux-armv7.so.2":       "sys-libs/glibc",
		"ld-linux-powerpc.so":       "sys-libs/glibc",
		"ld-linux-powerpc.so.1":     "sys-libs/glibc",
		"ld-linux-powerpc.so.2":     "sys-libs/glibc",
		"ld-linux-powerpc64.so":     "sys-libs/glibc",
		"ld-linux-powerpc64.so.1":   "sys-libs/glibc",
		"ld-linux-powerpc64.so.2":   "sys-libs/glibc",
		"ld-linux-powerpc64le.so":   "sys-libs/glibc",
		"ld-linux-powerpc64le.so.1": "sys-libs/glibc",
		"ld-linux-powerpc64le.so.2": "sys-libs/glibc",
		"ld-linux-riscv64gc.so":     "sys-libs/glibc",
		"ld-linux-riscv64gc.so.1":   "sys-libs/glibc",
		"ld-linux-riscv64gc.so.2":   "sys-libs/glibc",
		"ld-linux-s390x.so":         "sys-libs/glibc",
		"ld-linux-s390x.so.1":       "sys-libs/glibc",
		"ld-linux-s390x.so.2":       "sys-libs/glibc",
		"ld-linux-x86_64.so":        "sys-libs/glibc",
		"ld-linux-x86_64.so.1":      "sys-libs/glibc",
		"ld-linux-x86_64.so.2":      "sys-libs/glibc",
		"ld-linux-i686.so":          "sys-libs/glibc",
		"ld-linux-i686.so.1":        "sys-libs/glibc",
		"ld-linux-i686.so.2":        "sys-libs/glibc",
		"ld-linux-armhf.so":         "sys-libs/glibc",
		"ld-linux-armhf.so.1":       "sys-libs/glibc",
		"ld-linux-armhf.so.2":       "sys-libs/glibc",
		"libz.so.1":                 "sys-libs/zlib",
		"libz.so":                   "sys-libs/zlib",
		"libthai.so":                "dev-libs/libthai",
		"libthai.so.0":              "dev-libs/libthai",
		"libresolv.so.2":            "sys-libs/glibc",
	}
)

func lookupSymbol(library string) (string, bool) {
	r, ok := symbolMap[library]
	return r, ok
}
