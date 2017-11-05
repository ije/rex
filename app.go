package webx

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/ije/gox/utils"
)

type App struct {
	root         string
	packMode     string
	debuging     bool
	debugPort    int
	debugProcess *os.Process
	building     bool
	buildLog     []string
}

func initApp(root string) (app *App, err error) {
	fi, err := os.Lstat(root)
	if (err != nil && os.IsNotExist(err)) || (err == nil && !fi.IsDir()) {
		err = fmt.Errorf("root(%s) is not a valid directory", root)
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
				if !config.Debug {
					cmd.Args = append(cmd.Args, "--production")
				}
				cmd.Dir = root
				if config.Debug {
					cmd.Stderr = os.Stderr
					cmd.Stdout = os.Stdout
					fmt.Println("[npm] check/install dependencies...")
				}
				err = cmd.Run()
				if err != nil {
					return
				}
				os.Setenv("PATH", fmt.Sprintf("%s:%s", path.Join(root, "node_modules/.bin"), os.Getenv("PATH")))
			}
		}
	}

	switch packMode {
	case "webpack":
		_, err = exec.LookPath("webpack")
		if err == nil && config.Debug {
			_, err = exec.LookPath("webpack-dev-server")
		}
		if err != nil {
			cmd := exec.Command("npm", "-g", "webpack")
			if config.Debug {
				cmd.Args = append(cmd.Args, "webpack-dev-server")
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				fmt.Println("[npm] install webpack/webpack-dev-server...")
			}
			cmd.Run()
		}

		_, err = exec.LookPath("webpack")
		if err == nil && config.Debug {
			_, err = exec.LookPath("webpack-dev-server")
		}
		if err != nil {
			return
		}
	}

	app = &App{
		root:     root,
		packMode: packMode,
	}

	if config.Debug {
		go app.Debug()
	} else {
		go app.Build()
	}
	return
}

func (app *App) Root() string {
	return app.root
}

func (app *App) BuildLog() []string {
	return app.buildLog
}

func (app *App) Debug() (err error) {
	if app.debuging {
		err = fmt.Errorf("app is debuging")
		return
	}

	app.debuging = true
	defer func() {
		app.debugProcess = nil
		app.debuging = false
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
		err = cmd.Start()
		if err != nil {
			return err
		}

		app.debugPort = debugPort
		app.debugProcess = cmd.Process
		err = cmd.Wait()
	}

	return
}

func (app *App) Build() (err error) {
	if app.building {
		err = fmt.Errorf("app is building")
		return
	}

	app.building = true
	defer func() {
		app.building = false
	}()

	switch app.packMode {
	case "webpack":
		cmd := exec.Command("webpack", "--hide-modules", "--color=false")
		cmd.Env = append(os.Environ(), "NODE_ENV=production")
		cmd.Dir = app.root
		var output []byte
		var level string
		var msg string
		output, err = cmd.CombinedOutput()
		if err != nil {
			level = "error"
			msg = err.Error()
		} else {
			level = "info"
			msg = string(output)
		}
		app.buildLog = append(app.buildLog, fmt.Sprintf(`%s [%s] %s`, time.Now().Format("2006/01/02 15:04:05"), level, msg))
	}

	return
}
