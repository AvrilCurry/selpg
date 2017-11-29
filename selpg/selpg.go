package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	argno            int       // 用来记录参数的下标
	linecount        int       // 用来记录当前的行数
	pagecount        int       // 用来记录当前的页数
	startpage        int       // 开始打印的页数
	endpage          int       // 结束打印的页数
	pageline         int       // 每一页最大的行数
	pageeof          bool      // 是否到达文件末尾
	destination      string    // 管道的位置
	pagetype         string    // 记录遍历文件的类型 “l” 或者 “f”
	inputfile        = ""      // 记录输入文件的位置
	programname      = ""      // 记录程序的名称
	isError          = false   // 是否出现错误
	parsed           = false   // 是否解析过命令行
	isInputFileExist = false   // 判断输入文件是否存在
	fin              *os.File  // inputfile指针
	fout             *os.File  // outfile指针
	consumer         *exec.Cmd // 管道处理
	_Error           error     // 错误变量
)

func init() {
	flag.IntVar(&startpage, "s", 1, "the start page")                          // option -s 后面带的是int, 所以声明为IntVar
	flag.IntVar(&endpage, "e", 1, "the end page")                              // option -e 后面带的是int, 所以声明为IntVar
	flag.IntVar(&pageline, "l", 72, "the lines of the page")                   // option -l 后面带的是int, 所以声明为IntVar
	flag.BoolVar(&pageeof, "f", false, "the end of the page")                  // option -f 后面不带有参数，所以声明为BoolVar
	flag.StringVar(&destination, "d", "", "the destination of the outputfile") // option -d 后面带的是string, 所以声明为StringVar
}

func usage() {
	fmt.Printf("\nUSAGE: %s -sstart_page -eend_page [ -f | -llines_per_page ][ -ddest ] [ in_filename ]\n\n", os.Args[0])
}

/* 在每一个有‘-’的option参数，如果option包含参数， 则参数和option之间有空格  如： -e 15 而不是 -e15 */
func processArgs() {
	flag.Parse()
	parsed = true

	args := os.Args // 获取每一个参数，以空格为分隔符的都一个数组
	argno = 0
	programname += args[0] // 用于下面错误发生时的输出

	/*check the command-line argument for validity*/

	/* Not enough arguments, minimum command line is "./selpg -s start -e end"*/
	if len(args) < 5 {
		log.Printf("Type: Error 1 occur at line 58. \n\t%s : not enough arguments.\n", args[0])
		usage()
		os.Exit(1)
	}

	/* Handle the 1st arg - start-page*/
	/* Check if has the option -s */
	if strings.Compare(args[1], "-s") != 0 {
		log.Printf("Type: Error 2 occur at line 66. \n\t%s : the first argument is not \"-s\"(startpage).\n", args[0])
		usage()
		os.Exit(2)
	} else {
		argno += 2
		/* Check the value of the start-page */
		if startpage < 1 || (startpage > int(^uint(0)>>1)) {
			log.Printf("Type: Error 3 occur at line 73. \n\t%s : invalid start page %d.\n", args[0], startpage)
			usage()
			os.Exit(3)
		}
	}

	/* Handle the 2nd arg - end-page*/
	/* Check if has the option -e */
	if strings.Compare(args[3], "-e") != 0 {
		log.Printf("Type: Error 4 occur at line 82. \n\t%s : the second argument is not \"-e\"(endpage).\n", args[0])
		usage()
		os.Exit(4)
	} else {
		argno += 2
		/* Check th value of the end-page */
		if endpage < 1 || (endpage > int(^uint(0)>>1)) || endpage < startpage {
			log.Printf("Type: Error 5 occur at line 89. \n\t%s : invalid end page %d.\n", args[0], endpage)
			usage()
			os.Exit(5)
		}
	}

	/* Handle other arguments start with '-'*/
	argno++
	for (argno <= (len(args) - 1)) && args[argno][0] == '-' {
		switch args[argno][1] {
		case 'l':
			/* Check the value of the pagelines */
			if pageline < 1 || (pageline > int(^uint(0)>>1)) { // int(^uint(0) >> 1) int的最大值
				log.Printf("Type: Error 6 occur at line 102. \n\t%s : invalid page lines %d.\n", args[0], pageline)
				usage()
				os.Exit(6)
			}
			pagetype = "l"
			argno += 2
		case 'f':
			/* Check whether the option -f has a value */
			if (argno+1 <= (len(args) - 1)) && args[argno+1][0] >= '0' && args[argno+1][0] <= '9' {
				log.Printf("Type: Error 7 occur at line 111. \n\t%s : option should be \"-f\".\n", args[0])
				usage()
				os.Exit(7)
			}
			pagetype = "f"
			argno++
		case 'd':
			/* Check the value of the option -d */
			if (argno+1 <= (len(args) - 1)) && strings.Compare(destination, "") == 0 {
				log.Printf("Type: Error 8 occur at line 120. \n\t%s : \"-d\" option requires a printer destination.\n", args[0])
				usage()
				os.Exit(8)
			}
			argno += 2
		default:
			/* Check if has other undefined arguments */
			log.Printf("Type: Error 9 occur at line 127. \n\t%s : unknow option %s.\n", args[0], args[argno][1:])
			usage()
			os.Exit(9)
		}
	}

	/* If there is an argument left -- the input-file */
	if argno <= len(args)-1 {
		inputfile += args[argno]

		/* check if the file exists */
		_, err := os.Stat(inputfile)
		if os.IsNotExist(err) {
			log.Printf("Type: Error 10 occur at line 140. \n\t%s : input file %s does not exists.\n", args[0], inputfile)
			usage()
			os.Exit(10)
		}

		/* Check if the file is readable */
		_, _err := os.OpenFile(inputfile, os.O_RDONLY, 0666)
		if _err != nil {
			log.Printf("Type: Error 11 occur at line 148. \n\t%s : input file %s exists but cannot be read.\n", args[0], inputfile)
			usage()
			os.Exit(11)
		}

	}
}

func processInput() {
	/* set the input source */
	/* 这里其实分了两种情况，第一种是命令行包含输入文件， 另一种是不包含输入文件
	不包含输入文件其实就是重定向，如果不包括 < 就是默认来自用户输入，但重定向都可以通过os.Stdin获得*/
	if strings.Compare(inputfile, "") != 0 {
		var err error
		fin, err = os.Open(inputfile) // 打开文件
		if err != nil {
			log.Printf("Type: Error 12 occur at line 162. \n\t%s : input file %s cannot open.\n", programname, inputfile)
			usage()
			os.Exit(12)
		}
	} else {
		fin = os.Stdin // 将os.Stdin赋值给fin
	}

	/* Set the output destination */
	/* 对于输出也有两种情况， 一种是输出到屏幕或者是重定向的文件，另一种情况是将输出传送到管道中 */
	if strings.Compare(destination, "") == 0 {
		fout = os.Stdout
	} else {
		var err error
		consumer = exec.Command("lp -d" + destination) // 返回cmd结构来执行带有相关参数的命令
		consumer.Start()                               // 开始执行命令
		if err != nil {
			log.Printf("Type: Error 13 occur at line 178. \n\t%s : could not open pipe to \"%s\".\n", programname, destination)
			usage()
			os.Exit(13)
		}
	}

	/* read data and write data based on pagetype */
	buffreader := bufio.NewReader(fin)  // 声明一个带缓存的bufio.Reader对象，默认大小是4096
	buffwriter := bufio.NewWriter(fout) // 声明一个带缓存的bufio.Writer对象，默认大小是4096

	/* 当命令行有 option -l 或者  既没有 -l 也没有 -f，默认是 -l 72 */
	if strings.Compare(pagetype, "l") == 0 || strings.Compare(pagetype, "") == 0 {
		linecount = 0
		pagecount = 1

		for true {
			var line string
			line, _Error = buffreader.ReadString('\n') // 以"\n"为分隔符读取每一行
			if _Error != nil || _Error == io.EOF {     // 遇到出错或EOF就不在读取
				break
			}
			linecount++
			if linecount > pageline {
				linecount = 1
				pagecount++
			}

			if pagecount >= startpage && pagecount <= endpage {
				if strings.Compare(destination, "") != 0 {
					consumer.Stdin = strings.NewReader(line) // 向cmd命令的Stdin传入一行数据
				} else {
					_, err := buffwriter.WriteString(line) // 将每一行的数据中的数据写进文件或者屏幕
					/* 刷新缓存 缓存的大小是 4096 bytes, 如果缓存满了的时候就不在向缓存中写数据，
					所以当输入文件的内容比较多时，输出的数据可能会比预期要少，因此需要及时刷新，将
					缓存中的数据提交给io.Writer */
					buffwriter.Flush()
					if err != nil {
						log.Printf("Type: Error 14 occur at line 210. \n\t%s : error[%s] occur on writing to output file.\n", programname, err.Error())
						usage()
						os.Exit(14)
					}
				}
			}
		}
	} else if strings.Compare(pagetype, "f") == 0 {
		pagecount = 1

		for true {
			var word byte
			word, _Error = buffreader.ReadByte() // 将每一个byte数据写进文件或者屏幕
			if _Error == io.EOF {
				break
			} else if word == '\f' {
				pagecount++
			}

			if pagecount >= startpage && pagecount <= endpage {
				if strings.Compare(destination, "") != 0 {
					consumer.Stdin = strings.NewReader(string(word)) // 向cmd命令的Stdin传入一个byte
				} else {
					err := buffwriter.WriteByte(word) // 将每一个byte写进文件或者屏幕
					buffwriter.Flush()                // 同理，需要及时刷新
					if err != nil {
						log.Printf("Type: Error 14 occur at line 235. \n\t%s : error[%s] occur when writing to the output file.\n", programname, err.Error())
						usage()
						os.Exit(14)
					}
				}
			}
		}
	}

	if pagecount < startpage {
		log.Printf("Type: Error 15 occur at line 245. \n\t%s : start-page(%d) is greater than total-page(%d), no output written.\n", programname, startpage, pagecount)
		usage()
		os.Exit(15)
	} else if pagecount < endpage {
		log.Printf("Type: Error 16 occur at line 249. \n\t%s : end-page(%d) is greater than total-page(%d), less output than expected.\n", programname, endpage, pagecount)
		usage()
		os.Exit(16)
	}

	if _Error != nil && _Error != io.EOF {
		log.Printf("Type: Error 17 occur at line 255. \n\t%s : error[%s] occur when reading from the input file.\n", programname, _Error.Error())
		fin.Close()
		usage()
		os.Exit(14)
	} else if _Error != nil && _Error == io.EOF {
		log.Printf("Type: Error 17 occur at line 260. \n\t%s : Done.\n", programname)
		fin.Close()  // 关闭输入流
		fout.Close() // 关闭输出流

		if strings.Compare(destination, "") != 0 {

			/*  Wait等待command退出，和Start一起使用，如果命令能够顺利
			执行完并顺利退出则返回nil，否则的话便会返回error，其中Wait会是放掉
			所有与cmd命令相关的资源 */
			consumer.Wait()
		}

		usage()
		os.Exit(17)
	}

}

func main() {
	processArgs()
	processInput()

	return
}
