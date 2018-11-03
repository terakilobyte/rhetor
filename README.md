# rhetor

When you first clone this repo, you'll need to run the following commands
from your `$GOPATH/src/github.com/docker` folder.

```
rm -rf docker/vendor
rm -rf cli/vendor
rm -rf distribution/vendor
```

You will need the MongoDB Training AWS credentials stored in your default location (`~/.aws/...`) and you will need to set the
environmental variable `DOCKER_API_VERSION` to your Docker API version.

`docker version | grep 'API version'`

`export DOCKER_API_VERSION=<api version from previous command>`

You will also need to define an environmental variabled called `RHETOR_AWS_PROFILE` that is set to the AWS profile you would like to use,
for example mine is `RHETOR_AWS_PROFILE=tkb`
