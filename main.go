package main

import (
	"bytes"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type BuildrProps struct {
	ProjectName string `yaml:"project-name"`
}

type BuildrCmd struct {
	Filename string
	FileData string
}

const (
	runCmdName = "run"
)

var (
	runCmd      = flag.NewFlagSet(runCmdName, flag.ExitOnError)
	buildrProps BuildrProps
	buildrEnvs  = make(map[string]interface{})
	environment string
)

func init() {
	runCmd.StringVar(&environment, "e", "test", "environment to run buildr upon")
}

func main() {
	switch os.Args[1] {
	case runCmdName:
		runCmd.Parse(os.Args[2:])
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}

	parseBuildrProperties()
	parseBuildrEnvs()

	cmds := getInterpolatedCmdFiles()

	for _, cmd := range cmds {
		fmt.Print("\033[0;32m")
		fmt.Println(fmt.Sprintf("====> Executing script: %v ", cmd.Filename))
		fmt.Println("--------------")
		fmt.Print("\033[0m")

		cmd := exec.Command("sh", "-c", cmd.FileData)

		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		if err := cmd.Start(); err != nil {
			fmt.Println(err)
		}

		output, _ := cmd.CombinedOutput()

		fmt.Printf("%s\n", output)

		if err := cmd.Wait(); err != nil {
			fmt.Println(err)
		}

		fmt.Println()
	}
}

func parseBuildrProperties() {
	brFile, err := ioutil.ReadFile(".buildr.properties")

	if err != nil {
		fmt.Println(err)
	}

	yaml.Unmarshal(brFile, &buildrProps)
}

func parseBuildrEnvs() {
	envTpl, err := template.ParseFiles(fmt.Sprintf("./buildr/%v/env.buildr", environment))

	if err != nil {
		fmt.Println(err)
	}

	rawEnvs := make(map[string]interface{})

	err = yaml.Unmarshal(
		execTemplate(envTpl, buildrProps),
		&rawEnvs,
	)

	if err != nil {
		fmt.Println(err)
	}

	for k, v := range rawEnvs {
		buildrEnvs[strings.ToUpper(strings.Replace(k, "-", "_", -1))] = v
	}

}

func execTemplate(tpl *template.Template, data interface{}) []byte {
	var out bytes.Buffer

	tpl.Execute(&out, &data)

	return out.Bytes()
}

func getInterpolatedCmdFiles() []BuildrCmd {
	var cmds []BuildrCmd

	filepath.Walk(fmt.Sprintf("./buildr/%v/", environment), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".cmd.buildr") {
			envTpl, err := template.ParseFiles(path)

			if err != nil {
				return err
			}

			cmd := BuildrCmd{
				Filename: path,
				FileData: string(execTemplate(envTpl, buildrEnvs)),
			}

			cmds = append(cmds, cmd)
		}

		return nil
	})

	return cmds
}
