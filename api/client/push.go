package client

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/docker/distribution/reference"
	Cli "github.com/docker/docker/cli"
	flag "github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/registry"
)

// CmdPush pushes an image or repository to the registry.
//
// Usage: docker push NAME[:TAG]
func (cli *DockerCli) CmdPush(args ...string) error {
	cmd := Cli.Subcmd("push", []string{"NAME[:TAG]"}, Cli.DockerCommands["push"].Description, true)

	// Trying to add force options to push.
	force := cmd.Bool([]string{"f", "-force"}, false, "Force push of the image")

	// Get
	images := CmdImages("-all")

	addTrustedFlags(cmd, false)
	cmd.Require(flag.Min, 1) // Changed from .Exact to .Min

	cmd.ParseFlags(args, true)

	ref, err := reference.ParseNamed(cmd.Arg(0))
	if err != nil {
		return err
	}

	var tag string
	switch x := ref.(type) {
	case reference.Digested:
		return errors.New("cannot push a digest reference")
	case reference.Tagged:
		tag = x.Tag()
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}
	// Resolve the Auth config relevant for this server
	authConfig := registry.ResolveAuthConfig(cli.configFile, repoInfo.Index)
	// If we're not using a custom registry, we know the restrictions
	// applied to repository names and can warn the user in advance.
	// Custom repositories can have different rules, and we must also
	// allow pushing by image ID.
	if repoInfo.Official {
		username := authConfig.Username
		if username == "" {
			username = "<user>"
		}
		return fmt.Errorf("You cannot push a \"root\" repository. Please rename your repository to <user>/<repo> (ex: %s/%s)", username, repoInfo.LocalName)
	}

	// START:		CPEN 321 TEAM 37------------------------------------------------------------------------
	// If you previously created an image named username/imageName
	// 		and you create a new image called username/imageName
	//		and try to push it
	// Then docker push will escape
	//		and warn you that the image exists
	//		and if you want to overwrite it, use the docker push -f flag

	//if imageNameExists() {
	if *force == false {
		return errors.New("This image name already exists, if you want to overwrite it use docker push -f")
	}
	//}
	// END:			CPEN 321 TEAM 37------------------------------------------------------------------------

	if isTrusted() {
		return cli.trustedPush(repoInfo, tag, authConfig)
	}

	v := url.Values{}
	v.Set("tag", tag)

	_, _, err = cli.clientRequestAttemptLogin("POST", "/images/"+ref.Name()+"/push?"+v.Encode(), nil, cli.out, repoInfo.Index, "push")
	return err
}
