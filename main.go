package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v2"
)

var (
	pullUsername string
	pullPassword string

	pushUsername string
	pushPassword string

	configPath string
)

func init() {
	flag.StringVar(&pullUsername, "pull-username", "", "username for docker login in pull action")
	flag.StringVar(&pullPassword, "pull-password", "", "password for docker login in pull action")

	flag.StringVar(&pushUsername, "push-username", "", "username for docker login in push action")
	flag.StringVar(&pushPassword, "push-password", "", "password for docker login in push action")
	flag.StringVar(&configPath, "configPath", "", "migrate the configuration file for the image")

	flag.Parse()
}

func main() {
	if configPath == "" {
		log.Println("configPath is empty and exit")
		os.Exit(1)
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	var migrations ImageMigration

	err = yaml.Unmarshal(data, &migrations)
	if err != nil {
		panic(err)
	}
	for _, unit := range migrations.MigrationUnits {
		log.Println(fmt.Sprintf("start migrate image. unit=%#v", *unit))
		err := PullSourceImage(unit)
		if err != nil {
			log.Println(fmt.Sprintf("migrate image with a error:%v. unit=%#v", err, *unit))
			continue
		}
		err = MigrateToDestination(unit)
		if err != nil {
			log.Println(fmt.Sprintf("migrate image with a error:%v. unit=%#v", err, *unit))
			continue
		}
		log.Println(fmt.Sprintf("migrate image successfully. unit=%#v", *unit))
	}

}

// PullSourceImage
// pull image to local
func PullSourceImage(unit *MigrationUnit) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	pullOptions := types.ImagePullOptions{}
	if pullUsername != "" || pullPassword != "" {
		authConfig := types.AuthConfig{
			Username: pullUsername,
			Password: pullPassword,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return err
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		pullOptions.RegistryAuth = authStr
	}

	out, err := cli.ImagePull(context.Background(), unit.SourceImage, pullOptions)
	if err != nil {
		return err
	}
	defer out.Close()

	// show progress to stdout
	_, err = io.Copy(os.Stdout, out)
	return err
}

// MigrateToDestination
// rename and push the new image to registry
func MigrateToDestination(unit *MigrationUnit) error {
	// rename tag
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	err = cli.ImageTag(context.Background(), unit.SourceImage, unit.DestinationImage)
	if err != nil {
		return err
	}

	pushOption := types.ImagePushOptions{}
	if pushUsername != "" || pushPassword != "" {
		authConfig := types.AuthConfig{
			Username: pushUsername,
			Password: pushPassword,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return err
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		pushOption.RegistryAuth = authStr
	}

	// push
	out, err := cli.ImagePush(context.Background(), unit.DestinationImage, pushOption)
	if err != nil {
		return err
	}

	defer out.Close()
	_, err = io.Copy(os.Stdout, out)
	return err
}
