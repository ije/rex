package webx

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"time"
)

var debugPort = 9000

type App struct {
	root         string
	packingMode  string
	debuging     bool
	debugProcess *os.Process
	building     bool
	buildLog     []string
}

func initApp(root string) (app *App, err error) {
	fi, err := os.Lstat(root)
	if (err != nil && os.IsNotExist(err)) || (err == nil && !fi.IsDir()) {
		err = errf("root(%s) is not a valid directory", root)
		return
	}

	var needNodeJs bool
	var packingMode string
	if _, err := os.Lstat(path.Join(root, "webpack.config.js")); err == nil || os.IsExist(err) {
		packingMode = "webpack"
		needNodeJs = true
	}

	if needNodeJs {
		// specail node version
		if binDir := os.Getenv("NODEBINDIR"); len(binDir) > 0 {
			os.Setenv("PATH", strf("%s:%s", binDir, os.Getenv("PATH")))
		}

		_, err = exec.LookPath("npm")
		if err != nil {
			err = errf("server shutdown: missing nodejs environment")
			return
		}

		if _, e := os.Lstat(path.Join(root, "package.json")); e == nil || os.IsExist(e) {
			cmd := exec.Command("npm", "install")
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
		}
	}

	switch packingMode {
	case "webpack":
		_, err = exec.LookPath("webpack")
		if err == nil && config.Debug {
			_, err = exec.LookPath("webpack-dev-server")
		}
		if err != nil {
			args := []string{"install", "-g", "webpack"}
			if config.Debug {
				args = append(args, "webpack-dev-server")
			}
			cmd := exec.Command("npm", args...)
			if config.Debug {
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
		root:        root,
		packingMode: packingMode,
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
		err = errf("app is debuging")
		return
	}

	app.debuging = true
	defer func() {
		app.debugProcess = nil
		app.debuging = false
	}()

	for {
		l, err := net.Listen("tcp", strf(":%d", debugPort))
		if err == nil {
			l.Close()
			break
		}
		debugPort++
	}

	switch app.packingMode {
	case "webpack":
		cmd := exec.Command("webpack-dev-server", "--hot", "--host=127.0.0.1", strf("--port=%d", debugPort))
		cmd.Env = append(os.Environ(), "NODE_ENV=development")
		cmd.Dir = app.root
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		fmt.Println("[webpack] start dev-server...")
		err = cmd.Start()
		if err != nil {
			return err
		}

		app.debugProcess = cmd.Process
		err = cmd.Wait()
	}

	return
}

func (app *App) Build() (err error) {
	if app.building {
		err = errf("app is building")
		return
	}

	app.building = true
	defer func() {
		app.building = false
	}()

	switch app.packingMode {
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
		app.buildLog = append(app.buildLog, strf(`%s [%s] %s`, time.Now().Format("2006/01/02 15:04:05"), level, msg))
	}

	return
}
