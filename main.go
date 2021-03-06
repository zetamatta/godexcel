package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/transform"
	"golang.org/x/text/width"

	"github.com/zetamatta/go-texts/mbcs"
	"github.com/zetamatta/pipe2excel/excel"
)

var noWidth = flag.Bool("n", false, "do not change column-width")

var zenkaku = flag.Bool("z", false, "Convert to zenkaku")

var colWidth = flag.Float64("w", 1.50, "Set column-width")

// ConvRC return "XN" string to (n,x) which starts from (1,1)
func ConvRC(rc string) (int, int) {
	col := 0
	i := 0
	length := len(rc)
	for i < length {
		index := strings.IndexByte("ABCDEFGHIJKLMNOPQRSTUVWXYZ", rc[i])
		if index < 0 {
			break
		}
		col = col*26 + index + 1
		i++
	}
	row := 0
	for i < length {
		index := strings.IndexByte("0123456789", rc[i])
		if index < 0 {
			break
		}
		row = row*10 + index
		i++
	}
	return row, col
}

var rcPattern = regexp.MustCompile(`^[A-Za-z]+[0-9]+$`)

var xlsPattern = regexp.MustCompile(`.*\.xls[xm]?$`)

func main1(args []string) error {
	if len(args) <= 0 {
		flag.PrintDefaults()
		return nil
	}
	top := 1
	left := 1
	bottom := 200
	right := 200

	if len(args) >= 1 && rcPattern.MatchString(args[0]) {
		top, left = ConvRC(args[0])
		args = args[1:]
		if len(args) >= 1 && rcPattern.MatchString(args[0]) {
			bottom, right = ConvRC(args[0])
			args = args[1:]
		}
	}

	excel1, err := excel.New(true)
	if err != nil {
		return err
	}
	defer excel1.Close()

	var book1 *excel.Book
	if len(args) > 0 && xlsPattern.MatchString(args[0]) {
		fname, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}
		book1, err = excel1.Open(fname)
		args = args[1:]
	} else {
		book1, err = excel1.NewBook()
	}
	if err != nil {
		return err
	}
	defer book1.Release()

	sheet, err := book1.Item(1)
	if err != nil {
		return err
	}
	defer sheet.Release()

	isWidthSet := make(map[int]struct{})

	row := top
	for _, fname := range args {
		fd, err := os.Open(fname)
		if err != nil {
			return err
		}
		defer fd.Close()
		reader := mbcs.NewAutoDetectReader(fd, mbcs.ConsoleCP())
		if *zenkaku {
			reader = transform.NewReader(reader, width.Widen)
		}
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			col := left
			line := scanner.Text()
			for _, c := range line {
				if !*noWidth {
					if _, ok := isWidthSet[col]; !ok {
						_column, err := sheet.GetProperty("Columns", col)
						if err != nil {
							return err
						}
						column := _column.ToIDispatch()
						column.PutProperty("ColumnWidth", *colWidth)
						column.Release()
						isWidthSet[col] = struct{}{}
					}
				}
				_cell, err := sheet.GetProperty("Cells", row, col)
				if err != nil {
					return err
				}
				cell := _cell.ToIDispatch()
				cell.PutProperty("NumberFormatLocal", "@")
				cell.PutProperty("Value", string(c))
				cell.Release()
				col++
				if col > right {
					col = left
					row++
				}
			}
			row++
			if row > bottom {
				return nil
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if err := main1(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}
