# omnistrate-ctl Integration

Omnistrate CTL has a number of features that enable it to be seamlessly integrated with scripted and automated environments such as CI.

## Using omnistrate-ctl with GitHub Actions

Create secrets in your repository for your Omnistrate email and password and use omnistrate-ctl from your Github workflows using [Setup Omnistrate CTL Action](https://github.com/marketplace/actions/setup-omnistrate-ctl).

```
- name: Setup Omnistrate CTL
  uses: omnistrate-oss/setup-omnistrate-ctl@v1
  with:
    email: ${{ secrets.OMNISTRATE_USERNAME }}
    password: ${{ secrets.OMNISTRATE_PASSWORD }}
    version: latest # OPTIONAL

# Execute and example command
- name: Test CTL command
  run: |
    # rum simple command as an example
    omnistrate-ctl --version
    # omctl alias is also supported
    omctl --version
```

## Automating your CI/CD process with Omnistrate with GitHub Actions

Omnistrate allow you to Automate your CI/CD process and integrate with any CI/CD tool. Our community contributions provide a [guideline](https://github.com/omnistrate-community/ci-cd-example) on how to setup your build, test, release and fleet patching Actions on GitHub.

To get the example from the repo you can clone it using:

```
# Template availale at
gh repo clone omnistrate-community/ci-cd-example
```

## Using omnistrate-ctl with Docker

omnistrate-ctl is packaged and released in a container image that can be used to execute the command:

```
docker run -t ghcr.io/omnistrate-oss/omnistrate-ctl:latest
```

To log into the container and execute a series of commands, run the following command:

```
docker run -it --entrypoint /bin/sh -t ghcr.io/omnistrate-oss/omnistrate-ctl:latest
```

To persist the credentials across multiple container runs, run the following command

```
docker run -it -v ~/omnistrate-ctl:/omnistrate/ -t ghcr.io/omnistrate-oss/omnistrate-ctl:latest
```
