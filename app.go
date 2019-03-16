package wsx

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/ije/gox/crypto/rs"
	"github.com/ije/gox/utils"
)

type App struct {
	dir          string
	packMode     string
	debug        bool
	debugPort    int
	debugProcess *os.Process
	building     bool
	buildLogFile string
	buildRecords []*AppBuildRecord
}

type AppBuildRecord struct {
	ID        string
	PackMode  string
	Output    string
	StartTime int64
	EndTime   int64
	Error     string
}

func InitApp(dir string, buildLogFile string, debug bool) (app *App, err error) {
	dir, err = filepath.Abs(dir)
	if err != nil {
		return
	}

	fi, err := os.Lstat(dir)
	if (err != nil && os.IsNotExist(err)) || (err == nil && !fi.IsDir()) {
		err = fmt.Errorf("app dir(%s) is not a valid directory", dir)
		return
	}

	var requireNode bool
	var packMode string
	if fi, err := os.Lstat(path.Join(dir, "webpack.config.js")); err == nil && !fi.IsDir() {
		requireNode = true
		packMode = "webpack"
	}

	if requireNode {
		// specail node version
		if binDir := os.Getenv("NODEBINDIR"); len(binDir) > 0 {
			os.Setenv("PATH", fmt.Sprintf("%s:%s", binDir, os.Getenv("PATH")))
		}
		os.Setenv("PATH", fmt.Sprintf("%s:%s", path.Join(dir, "node_modules/.bin"), os.Getenv("PATH")))

		_, err = exec.LookPath("npm")
		if err != nil {
			err = fmt.Errorf("missing nodejs environment")
			return
		}

		if fi, e := os.Lstat(path.Join(dir, "package.json")); e == nil && !fi.IsDir() {
			var m map[string]interface{}
			err = utils.ParseJSONFile(path.Join(dir, "package.json"), &m)
			if err != nil {
				err = fmt.Errorf("parse package.json: %v", err)
				return
			}

			_, ok := m["dependencies"]
			if !ok {
				_, ok = m["devDependencies"]
			}
			if ok {
				cmd := exec.Command("npm", "install")
				if !debug {
					cmd.Args = append(cmd.Args, "--production")
				}
				cmd.Dir = dir
				if debug {
					cmd.Stderr = os.Stderr
					cmd.Stdout = os.Stdout
					fmt.Println("[npm] check/install dependencies...")
				}
				err = cmd.Run()
				if err != nil {
					return
				}
			}
		}
	}

	switch packMode {
	case "webpack":
		_, err = exec.LookPath("webpack")
		if err == nil && debug {
			_, err = exec.LookPath("webpack-dev-server")
		}
		if err != nil {
			fmt.Println("[npm] install webpack/webpack-cli/webpack-dev-server...")
			cmd := exec.Command("npm", "install", "webpack", "webpack-cli", "webpack-dev-server")
			cmd.Dir = dir
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			cmd.Run()
		}

		_, err = exec.LookPath("webpack-cli")
		if err == nil && debug {
			_, err = exec.LookPath("webpack-dev-server")
		}
		if err != nil {
			return
		}
	}

	app = &App{
		dir:          dir,
		packMode:     packMode,
		buildLogFile: buildLogFile,
	}

	if len(app.buildLogFile) > 0 {
		utils.ParseJSONFile(app.buildLogFile, &app.buildRecords)
	}

	if debug {
		app.debug = true
		go app.startDebug()
	} else {
		app.Build()
	}
	return
}

func (app *App) Dir() string {
	return app.dir
}

func (app *App) BuildRecords() []*AppBuildRecord {
	return app.buildRecords
}

func (app *App) startDebug() {
	if app.debugProcess != nil {
		return
	}

	defer func() {
		app.debugProcess = nil
		app.debugPort = 0
	}()

	debugPort := 9000
	for {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", debugPort))
		if err == nil {
			l.Close()
			break
		}
		debugPort++
	}

	switch app.packMode {
	case "webpack":
		cmd := exec.Command("webpack-dev-server", "--hot", "--host=127.0.0.1", fmt.Sprintf("--port=%d", debugPort))
		cmd.Env = append(os.Environ(), "NODE_ENV=development")
		cmd.Dir = app.dir
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		fmt.Println("[webpack] start dev-server...")
		err := cmd.Start()
		if err != nil {
			return
		}

		app.debugPort = debugPort
		app.debugProcess = cmd.Process
		cmd.Wait()
	}

	return
}

func (app *App) Build() *AppBuildRecord {
	record := &AppBuildRecord{
		ID:        rs.Hex.String(32),
		PackMode:  app.packMode,
		StartTime: time.Now().UnixNano(),
	}
	if app.building {
		record.EndTime = record.StartTime
		record.Error = "other build process is running"
	} else {
		app.buildRecords = append(app.buildRecords, record)
		if len(app.buildLogFile) > 0 {
			utils.SaveJSONFile(app.buildLogFile, app.buildRecords)
		}
		go app.build(record)
	}
	return record
}

func (app *App) build(record *AppBuildRecord) {
	if app.building {
		return
	}

	app.building = true
	defer func() {
		app.building = false
	}()

	switch app.packMode {
	case "webpack":
		cmd := exec.Command("webpack-cli", "--hide-modules", "--color=false")
		cmd.Env = append(os.Environ(), "NODE_ENV=production")
		cmd.Dir = app.dir
		output, err := cmd.CombinedOutput()
		record.Output = string(output)
		record.EndTime = time.Now().UnixNano()
		if err != nil {
			record.Error = err.Error()
		}
		if len(app.buildLogFile) > 0 {
			utils.SaveJSONFile(app.buildLogFile, app.buildRecords)
		}
	}
}
