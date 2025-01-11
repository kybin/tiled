package main

import (
	"archive/zip"
	"image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
)

type TileSet struct {
	Reference string
	Link      string
	Unzip     string
	Author    string
	License   string
	File      string
	TileSize  int
	BaseImg   image.Image
}

func main() {
	f, err := os.Open("tileset.toml")
	if err != nil {
		log.Fatal(err)
	}
	d := toml.NewDecoder(f)
	tilesets := make(map[string]*TileSet)
	_, err = d.Decode(&tilesets)
	if err != nil {
		log.Fatal(err)
	}
	for _, ts := range tilesets {
		if ts.Unzip == "" {
			err := download(ts.Link, ts.File)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			tf, err := ioutil.TempFile(os.TempDir(), "")
			err = download(ts.Link, tf.Name())
			if err != nil {
				log.Fatal(err)
			}
			err = unzip(tf.Name(), ts.Unzip, ts.File)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func download(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	io.Copy(out, resp.Body)
	return nil
}

func unzip(zipname, src, dest string) error {
	reader, err := zip.OpenReader(zipname)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, f := range reader.File {
		if f.Name != src {
			continue
		}
		in, err := f.Open()
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer out.Close()
		io.Copy(out, in)
		break
	}
	return nil
}
