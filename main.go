package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/urfave/cli"
)

var home, _ = os.UserHomeDir()
var dotContenu = filepath.Join(home, ".contenu")

var cwd, _ = os.Getwd()
var contenuJSON = filepath.Join(cwd, "contenu.json")

type ConfigAccount struct {
	Domain   string
	Username string
	Token    string
}

type ConfigFile struct {
	SelectedAccount string
	Accounts        []ConfigAccount
}

type ProjectConfig struct {
	Domain string
	Repository string
}

var config ConfigFile
var project ProjectConfig

func fileExists(filename string) bool {

	info, err := os.Stat(filename)
	if os.IsNotExist(err) {

		return false

	}

	return !info.IsDir()

}

func saveConfig() {

	file, err := json.MarshalIndent(config, "", "\t")
	if err != nil {

		log.Fatalln(err)

	}

	err = ioutil.WriteFile(dotContenu, file, 0644)
	if err != nil {

		log.Fatalln(err)

	}

}

// func readDirConfig() {

// 	filepath.Join(home, ".contenu")

// }

func login(domain string, username string, password string) string {

	message := map[string]interface{}{

		"username": username,
		"password": password,
	}

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {

		log.Fatalln(err)

	}

	resp, err := http.Post(fmt.Sprintf("https://%s/api/login?respond_with=json", domain), "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {

		log.Fatalln(err)

	}

	var result map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {

		log.Fatalln(err)

	}

	if result["token"] != nil {

		return result["token"].(string)

	} else {

		return ""

	}

}

func clone(account ConfigAccount, url string, branch string) bool {

	message := map[string]interface{}{

		"url": url,
		"branch": branch,
	}

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {

		log.Fatalln(err)

	}

	resp, err := http.Post(fmt.Sprintf("https://%s/api/clone?token=%s", account.Domain, account.Token), "application/json", bytes.NewBuffer(bytesRepresentation))
	if err != nil {

		log.Fatalln(err)

	}

	var result map[string]interface{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {

		log.Fatalln(err)

	}

	return result["message"].(string) == "success"

}

func getAccount(account string) ConfigAccount {

	split := strings.Split(account, "@")

	if len(split) == 1 {

		for _, acc := range config.Accounts {

			if strings.Contains(acc.Domain, split[0]) {
	
				return acc
	
			}
	
		}

		return ConfigAccount{}

	}

	username := split[0]
	domain := split[1]

	for _, acc := range config.Accounts {

		if strings.Contains(acc.Username, username) && strings.Contains(acc.Domain, domain) {

			return acc

		}

	}

	return ConfigAccount{}

}

var ignoreProjectDomain bool
func getSelectedAccount() ConfigAccount {

	if project.Domain != "" && !ignoreProjectDomain {

		for _, acc := range config.Accounts {
			
			if acc.Domain == project.Domain {

				return acc

			}

		}

		fmt.Fprintln(color.Output, color.HiRedString("! Could not find account for domain %s", color.HiWhiteString(project.Domain)))
		os.Exit(0)
		return ConfigAccount{}

	} else {

		return getAccount(config.SelectedAccount)

	}

}

func proceedWithProfile(force bool) {

	var cont bool

	if (getSelectedAccount() == ConfigAccount{}) {

		fmt.Fprintln(color.Output, color.HiRedString("! Please select an account using `contenu select <username@domain>`"))
		return

	}

	if !force {

		survey.AskOne(&survey.Confirm{

			Message: "Do you want to proceed with the selected account?",
		}, &cont, survey.WithValidator(survey.Required))

		if !cont {

			fmt.Fprintln(color.Output, color.HiRedString("! Exiting ContenuCLI"))
			os.Exit(0)

		}

	}

}

func main() {

	if fileExists(dotContenu) {

		data, err := ioutil.ReadFile(dotContenu)
		if err != nil {

			log.Fatalln(err)

		}

		json.Unmarshal(data, &config)

	} else {

		config = ConfigFile{}

	}

	
	if fileExists(contenuJSON) {

		data, err := ioutil.ReadFile(contenuJSON)
		if err != nil {

			log.Fatalln(err)

		}

		json.Unmarshal(data, &project)

	} else {

		project = ProjectConfig{}

	}

	saveConfig()

	app := &cli.App{

		Name:    "contenu",
		Usage:   "Interact with a Contenu CMS instance",
		Version: "1.0.0",
		Flags: []cli.Flag{

			cli.BoolFlag{

				Name: "force, f",
				Usage: "bypasses account confirmation",

			},

			cli.BoolFlag{

				Name: "ignore-project-domain, i",
				Usage: "ignores the account domain specified in contenu.json",

			},

		},
		Action: func(context *cli.Context) error {

			ignoreProjectDomain = context.Bool("ignore-project-domain")

			if (getSelectedAccount() == ConfigAccount{}) {

				fmt.Fprintln(color.Output, color.HiCyanString("* ContenuCLI v%s | No account selected", context.App.Version))

			} else {

				var detail string

				if project.Domain != "" && !ignoreProjectDomain {

					detail = " (project)"

				} else {
					
					detail = ""

				}

				fmt.Fprintln(color.Output, color.HiCyanString("* ContenuCLI v%s | %s@%s%s", context.App.Version, getSelectedAccount().Username, getSelectedAccount().Domain, detail))

			}

			if context.Args().Get(0) == "select" && context.Args().Get(1) != "" {

				account := getAccount(context.Args().Get(1))

				if (account == ConfigAccount{}) {

					fmt.Fprintln(color.Output, color.HiRedString("! Could not select account"))

				} else {

					fmt.Fprintln(color.Output, color.HiGreenString("* Account %s %s", color.HiWhiteString("%s@%s", account.Username, account.Domain), color.HiGreenString("selected!")))
					config.SelectedAccount = context.Args().Get(1)
					saveConfig()

				}

			} else if context.Args().Get(0) == "push" {

				proceedWithProfile(context.Bool("force"))

				var repo string

				if context.Args().Get(1) != "" {

					repo = context.Args().Get(1)

				} else {

					if (project != ProjectConfig{}) {

						if project.Repository != "" {

							repo = project.Repository

						} else {

							fmt.Fprintln(color.Output, color.HiRedString("! Repository not specified in `contenu.json`"))
							return nil

						}

					} else {
					
						fmt.Fprintln(color.Output, color.HiRedString("! `contenu.json` not found"))
						return nil

					}

				}

				var branch string

				if context.String("branch") != "" {

					branch = context.String("branch")

				} else if context.String("tag") != "" {

					branch = context.String("tag")

				} else {

					branch = "master"

				}

				cloneOutput := clone(getSelectedAccount(), repo, branch)

				if cloneOutput {

					fmt.Fprintln(color.Output, color.HiGreenString("* Push succesfull!"))

				} else {

					fmt.Fprintln(color.Output, color.HiRedString("! Push failed"))

				}

			} else if context.Args().Get(0) == "login" && context.Args().Get(1) != "" {

				fmt.Fprintln(color.Output, color.HiCyanString("* Initiating login procedure for domain %s%s", color.HiWhiteString(context.Args().Get(1)), color.HiCyanString("...")))

				var username, password string

				survey.AskOne(&survey.Input{

					Message: "Username:",
				}, &username, survey.WithValidator(survey.Required))

				survey.AskOne(&survey.Password{

					Message: "Password:",
				}, &password, survey.WithValidator(survey.Required))

				loginOutput := login(context.Args().Get(1), username, password)

				if loginOutput == "" {

					fmt.Fprintln(color.Output, color.HiRedString("! Invalid credentials"))

				} else {

					fmt.Fprintln(color.Output, color.HiGreenString("* Added account to .contenu file!"))

					config.Accounts = append(config.Accounts, ConfigAccount{

						Domain:   context.Args().Get(1),
						Username: username,
						Token:    loginOutput,
					})
					saveConfig()

				}

			} else {

				fmt.Fprintln(color.Output, color.HiRedString("! Invalid command"))

			}

			return nil

		},
	}

	err := app.Run(os.Args)
	if err != nil {

		log.Fatal(err)

	}

}
