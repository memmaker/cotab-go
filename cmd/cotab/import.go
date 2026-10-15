package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"cotab/dax"
	"cotab/ega"
	"cotab/internal/gameimport"
)

// markerFiles: a folder holding all of these is a Curse of the Azure Bonds install.
// Pool of Radiance has neither the big pictures nor the sky archive.
var markerFiles = []string{"BIGPIC1.DAX", "SKY.DAX", "GAME.OVR"}

// isMarker tells whether name is one of markerFiles.
func isMarker(name string) bool {
	base := strings.ToUpper(path.Base(name))
	for _, m := range markerFiles {
		if base == m {
			return true
		}
	}
	return false
}

// gameFile tells which files the engine reads: the .DAX archives, the base item
// table, the configuration and saved games.
func gameFile(name string) bool {
	base := strings.ToUpper(path.Base(name))
	switch {
	case strings.HasSuffix(base, ".DAX"):
		return true
	case base == "ITEMS" || base == "CURSE.CFG":
		return true
	case strings.HasPrefix(base, "SAVGAM") || strings.HasPrefix(base, "CHRDAT"):
		return true // saved games and saved characters
	}
	return false
}

// importGame (-import, for goplay) finds a Curse of the Azure Bonds install in
// whatever src is (folders, zips, CD images, the GOG installer), copies the files
// the engine reads into dst and writes dst/COVER.PNG from the title picture.
func importGame(src, dst string) error {
	files, err := gameimport.Find(src, func(name string) bool { return gameFile(name) || isMarker(name) })
	if err != nil {
		return err
	}
	// Group by folder and take the one holding every marker.
	byDir := map[string]map[string][]byte{}
	for name, b := range files {
		d := path.Dir(name)
		if byDir[d] == nil {
			byDir[d] = map[string][]byte{}
		}
		byDir[d][strings.ToUpper(path.Base(name))] = b
	}
	var game map[string][]byte
	for _, fs := range byDir {
		ok := true
		for _, m := range markerFiles {
			if _, has := fs[m]; !has {
				ok = false
			}
		}
		if ok {
			game = fs
			break
		}
	}
	if game == nil {
		return fmt.Errorf("no Curse of the Azure Bonds found in %s", src)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for name, b := range game {
		if !gameFile(name) {
			continue // a marker only
		}
		if err := os.WriteFile(filepath.Join(dst, name), b, 0o644); err != nil {
			return err
		}
	}
	return writeCover(dst)
}

// writeCover renders TITLE.DAX's title picture.
func writeCover(dst string) error {
	l, err := dax.NewLoader(dst)
	if err != nil {
		return err
	}
	var b []byte
	for _, id := range []int{1, 2, 3, 4} {
		if b = l.Block("TITLE", id); b != nil {
			break
		}
	}
	if b == nil {
		return errors.New("TITLE.DAX has no title picture")
	}
	im := dax.ParseImage(b, false, 0)
	pal := make(color.Palette, 16)
	for i, c := range ega.Palette {
		pal[i] = color.RGBA{c[0], c[1], c[2], 255}
	}
	img := image.NewPaletted(image.Rect(0, 0, ega.W, ega.H), pal)
	d, w := im.Frame(0), im.Width*8
	for y := 0; y < im.Height && y < ega.H; y++ {
		for x := 0; x < w && x < ega.W; x++ {
			img.SetColorIndex(x, y, d[y*w+x]&15)
		}
	}
	f, err := os.Create(filepath.Join(dst, "COVER.PNG"))
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
