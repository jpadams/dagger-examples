package main

import (
	"context"
	"fmt"
	"os"
	"golang.org/x/sync/errgroup"

	"dagger.io/dagger"
)

func main() {
	err := doCi()
	if err != nil {
		fmt.Println(err)
	}
}

func doCi() error {
	ctx := context.Background()

	// create a Dagger client
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		return err
	}
	defer client.Close()

	src := client.Host().Directory(".") // get the projects source directory
    nodeCache := client.CacheVolume("node") // create a cache volume

	npm, err := client.Container().From("node:18"). // initialize new container from npm image
		WithDirectory("/src", src, dagger.ContainerWithDirectoryOpts{
            Exclude: []string{"node_modules/"},
        }).
		WithMountedCache("/src/node_modules", nodeCache).
		WithWorkdir("/src").
		WithExec([]string{"npm", "install"}).Sync(ctx)  // execute npm install
	if err != nil {
		return err
	}

	lint := npm.WithExec([]string{"npm", "run", "lint"})
	test := npm.WithExec([]string{"npm", "run", "test"})

	eg, gctx := errgroup.WithContext(ctx)
    eg.Go(func() error {
        _, err = lint.Sync(gctx) 
		return err
    })
    eg.Go(func() error {
        _, err = test.Sync(gctx) 
		return err
    })
    eg.Wait()

	build, err := npm.WithExec([]string{"npm", "run", "build"}).Stdout(ctx)
	if err != nil {
		return err
	}
	// print output to console
	fmt.Println(build)

	return nil
}
