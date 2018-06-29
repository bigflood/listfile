package main

//go:generate stringer -type=Value

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
)

type Value int

const (
	valueStart Value = iota
	SizeValue
	DateValue
	NameValue
	valueLast
)

func main() {
	limitCount := flag.Int("n", 10, "file list size")
	reverseOrder := flag.Bool("r", false, "sort reverse order")
	ignoreHiddenFiles := flag.Bool("f", false, "ignore hidden files")
	fileValue := flag.String("v", "size", "value type (size, date, name)")
	flag.Parse()

	valueType, err := strToValue(*fileValue)
	if err != nil {
		fmt.Println(err)
		flag.PrintDefaults()
		os.Exit(-1)
	}

	obj := new(lister)
	obj.count = *limitCount
	obj.ignoreHiddenFiles = *ignoreHiddenFiles
	obj.reverseOrder = *reverseOrder
	obj.valueType = valueType
	obj.init()

	args := flag.Args()

	if len(args) == 0 {
		args = []string{"."}
	}

	for _, n := range args {
		absPath, _ := filepath.Abs(n)
		obj.walk(absPath)
	}

	for _, f := range obj.files {
		obj.printFile(f)
	}

	fmt.Printf("%v files (%v) \n", humanize.Comma(int64(obj.numFiles)), humanize.Bytes(uint64(obj.sumSize)))
}

type file struct {
	path string
	info os.FileInfo
}

type lister struct {
	files []file

	count             int
	ignoreHiddenFiles bool
	reverseOrder      bool
	valueType         Value

	numFiles int
	sumSize  int64

	compareFile func(a, b file) int
	printFile   func(f file)
}

func (obj *lister) walk(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return obj.walkDir(path)

	}

	return obj.addFile(path, info)
}

func (obj *lister) walkDir(dirPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return filepath.SkipDir
		}

		if obj.ignoreHiddenFiles && info.Name()[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() {
			obj.addFile(path, info)
		}
		return nil
	})
}

func (obj *lister) addFile(path string, info os.FileInfo) error {
	obj.sumSize += info.Size()
	obj.numFiles++

	f := file{
		path: path,
		info: info,
	}

	n := len(obj.files)
	i := sort.Search(n, func(i int) bool {
		return obj.compareFile(f, obj.files[i]) < 0
	})

	obj.files = append(obj.files, f)
	obj.files = append(obj.files[:i+1], obj.files[i:n]...)
	obj.files[i] = f

	if len(obj.files) > obj.count {
		obj.files = obj.files[:obj.count]
	}

	return nil
}

func (obj *lister) init() {
	switch obj.valueType {
	case SizeValue:
		obj.compareFile = func(a, b file) int {
			cmp := a.info.Size() - b.info.Size()
			return -int(cmp)
		}

		obj.printFile = func(f file) {
			fmt.Printf("%6v %s \n", humanize.Bytes(uint64(f.info.Size())), f.path)
		}
	case DateValue:
		obj.compareFile = func(a, b file) int {
			cmp := a.info.ModTime().Sub(b.info.ModTime())
			return -int(cmp)
		}

		obj.printFile = func(f file) {
			fmt.Printf("%6v  %s \n", f.info.ModTime().Format("2006-01-02 15:04:05 MST"), f.path)
		}
	case NameValue:
		obj.compareFile = func(a, b file) int {
			cmp := strings.Compare(a.info.Name(), b.info.Name())
			return cmp
		}

		obj.printFile = func(f file) {
			fmt.Printf("%20v %s \n", f.info.Name(), f.path)
		}
	}

	if obj.reverseOrder {
		originalFunc := obj.compareFile
		obj.compareFile = func(a, b file) int {
			return -originalFunc(a, b)
		}
	}
}

func strToValue(s string) (Value, error) {
	s += "value"

	for v := valueStart + 1; v < valueLast; v++ {
		if strings.ToLower(v.String()) == s {
			return v, nil
		}
	}
	return 0, errors.New("unknown value: " + s)
}
