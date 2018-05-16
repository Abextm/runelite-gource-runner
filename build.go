// Copyright (c) 2018 Abex
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package main

import (
	"bufio"
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
)

// Friendly name > git
var repos = map[string]string{
	"RuneLite":            "https://github.com/runelite/runelite.git",
	"runelite.net":        "https://github.com/runelite/runelite.net.git",
	"updater":             "https://github.com/runelite/updater.git",
	"static.runelite.net": "https://github.com/runelite/static.runelite.net.git",
	"launcher":            "https://github.com/runelite/launcher.git",
}

// Replace commit name with handle
var nameMap = map[string]string{
	"Max Weber":       "Abex",
	"UniquePassive":   "Lotto",
	"Charlie Waters":  "ChaoticConundrum",
	"Cameron Moberg":  "Noremac201",
	"Tomas Slusny":    "deathbeam",
	"joshpfox":        "josharoo",
	"Julian Tith":     "AvonGenesis",
	"Vagrant User":    "deathbeam",
	"Nickolaj Jepsen": "Fire-Proof",
	"Frederik Engels": "Dreyri",
}

// Map of handle > github name or ./file
var githubNames = map[string]string{
	"Abex":                  "AbexTM",
	"Adam":                  "Adam-",
	"Lotto":                 "devLotto",
	"Kronos":                "KronosDesign",
	"Jakob Vogel":           "jakobvogel",
	"Julian Nowaczek":       "jnowaczek",
	"Devin French":          "devinfrench",
	"C Gibson":              "GibsDev",
	"Jeremy Plsek":          "jplesk",
	"Joshua Filby":          "Joshua-F",
	"Kyle Stevenson":        "kylestev",
	"Jonathan Beaudoin":     "Jonatino",
	"Bart van Helvert":      "bartvhelvert",
	"Ben Moyer":             "bmoyer",
	"Kyle Fricilone":        "kfricilone",
	"Hunter W Bennett":      "hunterwb",
	"Tyler Hardy":           "tylerthardy",
	"David Kosub":           "Tape",
	"Nicholas Bailey":       "baileyn",
	"kokkue":                "utsukami",
	"Abel Briggs":           "abelbriggs1",
	"Martin Tuskevicius":    "DudeMartin",
	"roweyman":              "RobinWeymans",
	"Snorflake":             "ar1a",
	"Runelite auto updater": "./updater.png",
	"ModMatK":               "./mmk.png",
}

var repoDir = `repos`
var userDir = `users`

func updateRepos() {
	for name, repo := range repos {
		fmt.Printf("Updating %q\n", name)
		dirname := filepath.Join(repoDir, name)
		// clone/fetch repo
		{
			var cmd *exec.Cmd
			if _, err := os.Stat(dirname); os.IsNotExist(err) {
				cmd = exec.Command("git", "clone", repo, dirname)
			} else {
				cmd = exec.Command("git", "fetch", "origin")
				cmd.Dir = dirname
			}
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}
		// generate gource log
		{
			cmd := exec.Command("gource", "--output-custom-log", dirname+".log", dirname)
			cmd.Stderr = os.Stderr
			cmd.SysProcAttr = &syscall.SysProcAttr{ // Otherwise gource closes our terminal
				CreationFlags: 8,
			}
			err := cmd.Run()
			if err != nil {
				panic(err)
			}
		}
	}
}

type Line struct {
	Time uint64
	Name string
	Type string
	File string
}

func buildLog() {
	streams := map[string]chan *Line{}
	next := map[string]*Line{}
	// Create readers
	for sname := range repos {
		name := sname
		s := make(chan *Line)
		streams[name] = s
		go func() {
			fi, err := os.Open(filepath.Join(repoDir, name+".log"))
			if err != nil {
				panic(err)
			}
			defer fi.Close()
			bf := bufio.NewReader(fi)
			for {
				line, err := bf.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						close(s)
						return
					}
					panic(err)
				}
				for i := len(line) - 1; i >= 0; i-- {
					if line[i] >= 32 {
						line = line[:i+1]
						break
					}
				}
				sl := strings.Split(line, "|")
				time, err := strconv.ParseUint(sl[0], 10, 64)
				if err != nil {
					panic(err)
				}
				s <- &Line{
					time,
					sl[1],
					sl[2],
					sl[3],
				}
			}
		}()
	}

	fi, err := os.Create("built_log.log")
	if err != nil {
		panic(err)
	}
	defer fi.Close()

	lastTS := uint64(0)
	time := 0

	files := map[string]struct{}{}

	for {
		// find next item
		min := uint64(math.MaxUint64)
		var minK string
		for k := range streams {
			l := next[k]
			if l == nil {
				var ok bool
				l, ok = <-streams[k]
				if !ok {
					delete(streams, k)
					continue
				}
				next[k] = l
			}
			if min > l.Time {
				min = l.Time
				minK = k
			}
		}
		if len(streams) <= 0 {
			break
		}
		minV := next[minK]
		next[minK] = nil

		// Don't show this, only the updater touches it
		if strings.HasPrefix(minV.File, "/api") {
			continue
		}

		// turn timestamp into commit #
		if lastTS != minV.Time {
			lastTS = minV.Time
			time++
		}

		// swap names
		name, ok := nameMap[minV.Name]
		if !ok {
			name = minV.Name
		}
		// Add name to list implicitly
		if _, ok := githubNames[name]; !ok {
			githubNames[name] = name
		}

		path := minK + minV.File
		files[path] = struct{}{}

		fmt.Fprintf(fi, "%v|%v|%v|/%v\n", time, name, minV.Type, path)
	}

	for file, _ := range files {
		fmt.Fprintf(fi, "%v|%v|%v|/%v\n", time+10, "ModMatK", "D", file)
	}

	var ghc *github.Client
	var oauth *http.Client

	// Get user avatars
	for sn, gn := range githubNames {
		func() {
			var src io.Reader
			if strings.HasPrefix(gn, "./") {
				// From FS
				ifi, err := os.Open(gn)
				if err != nil {
					panic(err)
				}
				defer ifi.Close()
				src = ifi
			} else {
				// Try to find the user on github, and download their avatar
				if ghc == nil {
					oauth = oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
						AccessToken: string(os.Args[1]),
					}))
					ghc = github.NewClient(oauth)
				}
				user, _, err := ghc.Users.Get(context.Background(), gn)
				if err != nil {
					fmt.Printf("Cannot find user %q: %v\n", gn, err)
					return
				}
				res, err := oauth.Get(user.GetAvatarURL())
				if err != nil {
					panic(err)
				}
				defer res.Body.Close()
				src = res.Body
			}

			fi, err := os.OpenFile(filepath.Join(userDir, sn+".png"), os.O_CREATE|os.O_RDWR|os.O_EXCL, 0777)
			if err != nil {
				if os.IsExist(err) {
					return
				}
				panic(err)
			}
			defer fi.Close()

			// Gource complains about 5% of images, and will crash, so reencode them
			img, _, err := image.Decode(src)
			if err != nil {
				panic(err)
			}

			nimg := image.NewNRGBA(img.Bounds())
			draw.Draw(nimg, nimg.Bounds(), img, image.ZP, draw.Src)
			err = png.Encode(fi, nimg)
			if err != nil {
				panic(err)
			}
		}()
	}
}

func main() {
	updateRepos()
	buildLog()
}
