package main

import (
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	limitCount := flag.Int("n", 10, "file list size")
	ignoreHiddenFiles := flag.Bool("f", false, "ignore hidden files")
	flag.Parse()

	obj := new(lister)
	obj.count = *limitCount
	obj.ignoreHiddenFiles = *ignoreHiddenFiles

	args := flag.Args()

	for _, n := range args {
		obj.walk(n)
	}

	for _, f := range obj.files {
		fmt.Printf("%6v %s \n", humanize.Bytes(uint64(f.info.Size())), f.path)
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

	numFiles int
	sumSize  int64
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
		return obj.compareFile(f, obj.files[i])
	})

	obj.files = append(obj.files, f)
	obj.files = append(obj.files[:i+1], obj.files[i:n]...)
	obj.files[i] = f

	if len(obj.files) > obj.count {
		obj.files = obj.files[:obj.count]
	}

	return nil
}

func (obj *lister) compareFile(a, b file) bool {
	return a.info.Size() > b.info.Size()
}
