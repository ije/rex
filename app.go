package webx

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
	root         string
	packMode     string
	debugPort    int
	debugProcess *os.Process
	building     bool
	buildLog     []AppBuildRecord
	buildLogFile string
}

type AppBuildRecord struct {
	ID        string
	PackMode  string
	Output    string
	BuildTime int64
	Error     string
}

func InitApp(root string, buildLogFile string, debug bool) (app *App, err error) {
	root, err = filepath.Abs(root)
	if err != nil {
		return
	}

	fi, err := os.Lstat(root)
	if (err != nil && os.IsNotExist(err)) || (err == nil && !fi.IsDir()) {
		err = fmt.Errorf("app root(%s) is not a valid directory", root)
		return
	}

	var requireNode bool
	var packMode string
	if fi, err := os.Lstat(path.Join(root, "webpack.config.js")); err == nil && !fi.IsDir() {
		requireNode = true
		packMode = "webpack"
	}

	if requireNode {
		// specail node version
		if binDir := os.Getenv("NODEBINDIR"); len(binDir) > 0 {
			os.Setenv("PATH", fmt.Sprintf("%s:%s", binDir, os.Getenv("PATH")))
		}
		os.Setenv("PATH", fmt.Sprintf("%s:%s", path.Join(root, "node_modules/.bin"), os.Getenv("PATH")))

		_, err = exec.LookPath("npm")
		if err != nil {
			err = fmt.Errorf("missing nodejs environment")
			return
		}

		if fi, e := os.Lstat(path.Join(root, "package.json")); e == nil && !fi.IsDir() {
			var m map[string]interface{}
			err = utils.ParseJSONFile(path.Join(root, "package.json"), &m)
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
				cmd.Dir = root
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
			cmd.Dir = root
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
		root:         root,
		packMode:     packMode,
		buildLogFile: buildLogFile,
	}

	if len(app.buildLogFile) > 0 {
		utils.ParseJSONFile(app.buildLogFile, &app.buildLog)
	}

	if debug {
		go app.Debug()
	} else {
		app.Build()
	}
	return
}

func (app *App) Root() string {
	return app.root
}

func (app *App) BuildLog() []AppBuildRecord {
	return app.buildLog
}

func (app *App) Debug() {
	if app.debugProcess != nil {
		return
	}

	defer func() {
		app.debugProcess = nil
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
		cmd.Dir = app.root
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

func (app *App) Build() {
	if app.building {
		return
	}

	go app.build(rs.Hex.String(32))
}

func (app *App) build(id string) {
	app.building = true
	defer func() {
		app.building = false
	}()

	switch app.packMode {
	case "webpack":
		since := time.Since(time.Now())
		cmd := exec.Command("webpack-cli", "--hide-modules", "--color=false")
		cmd.Env = append(os.Environ(), "NODE_ENV=production")
		cmd.Dir = app.root
		output, err := cmd.CombinedOutput()
		record := AppBuildRecord{
			ID:        id,
			PackMode:  app.packMode,
			Output:    string(output),
			BuildTime: int64(since / time.Millisecond),
		}
		if err != nil {
			record.Error = err.Error()
		}
		app.buildLog = append(app.buildLog, record)
		if len(app.buildLogFile) > 0 {
			utils.SaveJSONFile(app.buildLogFile, app.buildLog)
		}
	}
}
