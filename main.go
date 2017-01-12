package main

//go:generate esc -o templates.go ./templates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fsnotify/fsnotify"
	"github.com/subosito/snowboard/drafter"
	snowboard "github.com/subosito/snowboard/parser"
	"github.com/urfave/cli"
)

var versionStr string
var engine snowboard.Parser

func main() {
	engine = drafter.Engine{}

	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Fprintf(c.App.Writer, "Snowboard version: %s\n", c.App.Version)
		fmt.Fprintf(c.App.Writer, "Drafter version: %s\n", engine.Version())
	}

	if versionStr == "" {
		versionStr = "HEAD"
	}

	app := cli.NewApp()
	app.Name = "snowboard"
	app.Usage = "API blueprint toolkit"
	app.Version = versionStr
	app.Commands = []cli.Command{
		{
			Name:  "lint",
			Usage: "Validate API blueprint",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "i",
					Usage: "API blueprint file",
				},
			},
			Action: func(c *cli.Context) error {
				return validate(c, c.String("i"))
			},
		},
		{
			Name:  "html",
			Usage: "Render HTML documentation",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "i",
					Usage: "API blueprint file",
				},
				cli.StringFlag{
					Name:  "o",
					Value: "index.html",
					Usage: "HTML file",
				},
				cli.StringFlag{
					Name:  "t",
					Value: "alpha",
					Usage: "Template for HTML documentation",
				},
				cli.BoolFlag{
					Name:  "s",
					Usage: "Serve HTML via HTTP server",
				},
				cli.StringFlag{
					Name:  "b",
					Value: "127.0.0.1:8088",
					Usage: "HTTP server listen address",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Bool("s") {
					return watchHTML(c, c.String("i"), c.String("o"), c.String("t"))
				}

				return renderHTML(c, c.String("i"), c.String("o"), c.String("t"))
			},
		},
		{
			Name:  "apib",
			Usage: "Render API blueprint",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "i",
					Usage: "API blueprint file",
				},
				cli.StringFlag{
					Name:  "o",
					Usage: "API blueprint output file",
				},
			},
			Action: func(c *cli.Context) error {
				return renderAPIB(c, c.String("i"), c.String("o"))
			},
		},
	}
	app.Run(os.Args)
}

func readFile(fn string) ([]byte, error) {
	info, err := os.Stat(fn)
	if err != nil {
		return nil, errors.New("File is not exist")
	}

	if info.IsDir() {
		return nil, errors.New("File is a directory")
	}

	return ioutil.ReadFile(fn)
}

func readTemplate(fn string) ([]byte, error) {
	tf, err := readFile(fn)
	if err == nil {
		return tf, nil
	}

	fs := FS(false)
	ff, err := fs.Open("/templates/" + fn + ".html")
	if err != nil {
		return nil, err
	}

	defer ff.Close()
	return ioutil.ReadAll(ff)
}

func renderHTML(c *cli.Context, input, output, tplFile string) error {
	bp, err := snowboard.Load(input, engine)
	if err != nil {
		return err
	}

	of, err := os.Create(output)
	if err != nil {
		return err
	}
	defer of.Close()

	tf, err := readTemplate(tplFile)
	if err != nil {
		return err
	}

	err = snowboard.HTML(string(tf), of, bp)
	if err != nil {
		return err
	}

	fmt.Fprintln(c.App.Writer, "HTML has been generated!")
	return nil
}

func renderAPIB(c *cli.Context, input, output string) error {
	b, err := snowboard.Read(input)
	if err != nil {
		return err
	}

	of, err := os.Create(output)
	if err != nil {
		return err
	}
	defer of.Close()

	_, err = io.Copy(of, bytes.NewReader(b))
	if err != nil {
		return err
	}

	fmt.Fprintln(c.App.Writer, "API blueprint has been generated!")
	return nil
}

func validate(c *cli.Context, input string) error {
	b, err := readFile(input)
	if err != nil {
		return err
	}

	bf := bytes.NewReader(b)

	out, err := snowboard.Validate(bf, engine)
	if err != nil {
		return err
	}

	if out == nil {
		fmt.Fprintln(c.App.Writer, "OK")
		return nil
	}

	var buf bytes.Buffer

	s := "--------"
	w := tabwriter.NewWriter(&buf, 8, 0, 0, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Row\tCol\tDescription")
	fmt.Fprintf(w, "%s\t%s\t%s\n", s, s, strings.Repeat(s, 8))

	for _, n := range out.Annotations {
		for _, m := range n.SourceMaps {
			fmt.Fprintf(w, "%d\t%d\t%s\n", m.Row, m.Col, n.Description)
		}
	}

	w.Flush()

	if len(out.Annotations) > 0 {
		return errors.New(buf.String())
	}

	return nil
}

func watchHTML(c *cli.Context, input, output, tplFile string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					renderHTML(c, input, output, tplFile)
				}
			case err := <-watcher.Errors:
				fmt.Fprintln(c.App.Writer, err)
			}
		}
	}()

	err = watcher.Add(input)
	if err != nil {
		return err
	}

	if _, err = os.Stat(tplFile); err == nil {
		err = watcher.Add(tplFile)
		if err != nil {
			return err
		}
	}

	renderHTML(c, input, output, tplFile)
	serveHTML(c.String("b"), output)

	<-done

	return nil
}

func serveHTML(bind, output string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, output)
	})

	return http.ListenAndServe(bind, nil)
}
