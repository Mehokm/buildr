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

	"github.com/fatih/color"
	"gopkg.in/yaml.v2"
)

type BuildrProps struct {
	ProjectName string `yaml:"project-name"`
}

type BuildrCmd struct {
	Filename string
	Filedata string
}

const (
	runCmdName      = "run"
	buildEnvCmdName = "build-env"
)

var (
	gColor      = color.New(color.FgGreen)
	runCmd      = flag.NewFlagSet(runCmdName, flag.ExitOnError)
	buildEnvCmd = flag.NewFlagSet(buildEnvCmdName, flag.ExitOnError)

	buildrProps BuildrProps
	buildrEnvs  = make(map[string]interface{})
	environment string
)

func init() {
	runCmd.StringVar(&environment, "e", "test", "environment to run buildr upon")
	buildEnvCmd.StringVar(&environment, "e", "test", "environment to run buildr upon")
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("must have a command.")
		os.Exit(2)
	}

	parseBuildrProperties()

	switch os.Args[1] {
	case runCmdName:
		runCmd.Parse(os.Args[2:])

		parseBuildrEnvs()

		runCmds()

		os.Exit(0)
	case buildEnvCmdName:
		buildEnvCmd.Parse(os.Args[2:])

		parseBuildrEnvs()

		fmt.Println("Generating Runfile...")

		generateRunfile()

		fmt.Println("Done!")

		os.Exit(0)
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(2)
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
	envTpl, err := template.ParseFiles(fmt.Sprintf("./.buildr/%v/env.buildr", environment))

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

	filepath.Walk(fmt.Sprintf("./.buildr/%v/", environment), func(path string, info os.FileInfo, err error) error {
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
				Filedata: string(execTemplate(envTpl, buildrEnvs)),
			}

			cmds = append(cmds, cmd)
		}

		return nil
	})

	return cmds
}

func runCmds() {
	cmds := getInterpolatedCmdFiles()

	for _, cmd := range cmds {
		gColor.Println(fmt.Sprintf("====> Executing script: %v ", cmd.Filename))
		gColor.Println("--------------")

		cmd := exec.Command("sh", "-c", cmd.Filedata)

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

func generateRunfile() {
	var b bytes.Buffer

	b.WriteString("#!/bin/bash")
	b.WriteString("\n# Auto-generated by buildr.  Do not modify.\n")

	for env, value := range buildrEnvs {
		b.WriteString(fmt.Sprintf("%s=\"%s\" ", env, value))
	}

	b.WriteString("go run *.go")

	if _, err := os.Stat("./.buildr/bin"); err != nil {
		if os.IsNotExist(err) {
			os.Mkdir("./.buildr/bin", 0777)
		}
	}

	err := ioutil.WriteFile("./.buildr/bin/Runfile", b.Bytes(), 0777)

	if err != nil {
		fmt.Println(err)

		os.Exit(2)
	}
}
