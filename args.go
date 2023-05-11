// Package main contains the main function and other related functions for the Tabby Editor program.
package main

import (
	"flag"
	"github.com/mattn/go-gtk/gdk"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
)

var listener net.Listener
var tabby_args []string
var pfocus_line *int
var pstandalone *bool

// open_files_from_args opens all files passed as arguments to the Tabby Editor program.
func open_files_from_args() {
	for _, s := range tabby_args {
		open_file_from_args(prefixed_path(s), *pfocus_line)
	}
}

// tabby_server continuously listens to incoming requests for opening files on a Unix socket.
func tabby_server() {
	var focus_line int
	buf := make([]byte, 1024)

	for {
		c, _ := listener.Accept()
		if nil != c {
			nread, err := c.Read(buf)
			if 0 >= nread {
				tabby_log("server: read from unix socket: " + err.Error())
				c.Close()
				continue
			}

			// At this point buf contains '\n' separated file names preceeded by focus
			// line number. Double '\n' at the end of list.

			gdk.ThreadsEnter()

			opened_cnt := 0
			s := buf[:]
			for cnt := 0; ; cnt++ {
				en := strings.Index(string(s), "\n")
				if 0 == en {
					break
				}
				if 0 == cnt {
					focus_line, _ = strconv.Atoi(string(s[:en]))
				} else {
					if open_file_from_args(string(s[:en]), focus_line) {
						opened_cnt++
					}
				}
				s = s[en+1:]
			}
			if opened_cnt > 0 {
				main_window.Present()
				file_tree_store()
				new_file := file_stack_pop()
				file_save_current()
				file_switch_to(new_file)
			}

			gdk.ThreadsLeave()

			c.Close()
		} else {
			// Dirty hack! There is no way to distinguish two cases:
			// 1) Accept returns error because socket was closed on tabby exit.
			// 2) Real error occured.
			// Commenting this line out to avoid misleading error messages on exit.
			//tabby_log(e.String())
			return
		}
	}
}

// provide_tabby_server handles the logic for starting a new instance of Tabby Editor as either server or client.
func provide_tabby_server(cnt int) bool {
	if cnt > 3 {
		return true
	}
	if *pstandalone {
		return true
	}

	if runtime.GOOS == "windows" {
		return true
	}
	user := os.Getenv("USER")
	socket_name := "/tmp/tabby-" + user
	listener, _ = net.Listen("unix", socket_name)
	if nil == listener {
		// Assume that socket or file with such name already exists.
		conn, _ := net.Dial("unix", socket_name)
		if nil == conn {
			// Socket exists but we cannot connect to it. Delete socket file then
			// and repeat the logic.
			os.Remove(socket_name)
			return provide_tabby_server(cnt + 1)
		}
		// Dial succeeded.
		for y := 0; y < len(tabby_args); y++ {
			println(tabby_args[y])
		}
		defer conn.Close()
		if len(tabby_args) > 0 {
			conn.Write([]byte(pack_tabby_args()))
		}
		return false
	}
	// Ok, this instance of tabby becomes a server.
	go tabby_server()
	return true
}

// init_args initializes the command-line arguments passed to Tabby Editor and starts a new instance as necessary.
func init_args() bool {
	pfocus_line = flag.Int("f", 1, "Focus line")
	pstandalone = flag.Bool("s", false, "Forces to open new instance of tabby.")
	flag.Parse()
	tabby_args = flag.Args()

	return provide_tabby_server(0)
}

// pack_tabby_args packs the command-line arguments passed to Tabby Editor for sending to the server.
func pack_tabby_args() string {
	res := strconv.Itoa(*pfocus_line) + "\n"
	for _, s := range tabby_args {
		res += prefixed_path(s) + "\n"
	}
	res += "\n"
	return res
}

// simplified_path simplifies the file path by removing redundant details like /./ and /../ elements.
func simplified_path(file string) string {
	res := file
	for {
		i := strings.Index(res, "/./")
		if -1 == i {
			break
		}
		res = res[:i+1] + res[i+3:]
	}
	for {
		i := strings.Index(res, "/../")
		if -1 == i {
			break
		}
		prev_slash := i - 1
		for ; '/' != res[prev_slash]; prev_slash-- {
		}
		res = res[:prev_slash+1] + res[i+4:]
	}
	return res
}

// prefixed_path returns a file path prefixed with the current working directory if the path is relative.
func prefixed_path(file string) string {
	if '/' != file[0] {
		// Relative file name.
		wd, err := os.Getwd()
		if "" == wd {
			tabby_log(err.Error())
		} else {
			file = wd + "/" + file
		}
	}
	return file
}

// open_file_from_args opens a file passed as an argument with the specified focus line.
// Returns true if successful, false otherwise.
func open_file_from_args(file string, focus_line int) bool {
	split_file := strings.SplitN(file, ":", 2)
	if len(split_file) >= 2 {
		focus_line, _ = strconv.Atoi(split_file[1])
	}
	file = simplified_path(split_file[0])
	if false == session_open_and_read_file(file) {
	  return false
	}
	rec, found := file_map[file]
	if found {
		cur_line := 1
		var y int
		for y = 0; y < len(rec.buf); y++ {
			if cur_line == focus_line {
				break
			}
			if rec.buf[y] == '\n' {
				cur_line++
			}
		}
		rec.sel_be = y
		rec.sel_en = y
	} else {
		return false
	}
	return true
}