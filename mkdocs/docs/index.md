# omnistrate-ctl Installation

Omnistrate CTL is a command line tool designed to streamline the creation, deployment, and management of your Omnistrate SaaS. Use it to build services from docker-compose files, manage service plans, and interact with the Omnistrate platform efficiently.

## Obtaining omnistrate-ctl

We provide CTL in the following formats:

### Install from Binaries

To run the CTL in your local environment, you can use the CTL binaries.
To obtain the latest version of the CTL binaries, execute the following command in your terminal:

```sh
curl -fsSL https://raw.githubusercontent.com/omnistrate-oss/omnistrate-ctl/main/install-ctl.sh | sh
```

This command will automatically download and install the latest version of the CTL binaries onto your system.

Run the following command to verify the installation. If the installation was successful, you should see the version number of the CTL displayed in the terminal.

```sh
omnistrate-ctl --version # or omnistrate-ctl -v
```

### Homebrew Tap

CTL can be installed using Homebrew. Homebrew can be installed on MacOS or Linux. It can be installed following the instructions in https://docs.brew.sh/Installation
To install the latest version of CTL using Homebrew, execute the following command in your terminal:

```
brew tap omnistrate/tap
brew install omnistrate/tap/omnistrate-ctl
```

Homebrew will automatically download and install the latest version of the CTL binaries onto your system.

### Docker Image

To integrate CTL into your CI/CD pipeline, you can use the CTL docker image.
The latest version of the CTL can be found in the docker image `ghcr.io/omnistrate-oss/omnistrate-ctl:latest`.

### GitHub Action

To integrate CTL into your GitHub workflows, you can use the GitHub Action: [Setup Omnistrate CTL](https://github.com/marketplace/actions/setup-omnistrate-ctl).
Using the GitHub Action you can execute any command on the CTL to automate your deployment workflow in GitHub.

## Login to Omnistrate

Omnistrate provides flexible login options to accommodate different environments and workflows. Below are the methods you can use to log in when running CTL locally. To log in using the Docker image or GitHub Action, refer to the [Integrating with Omnistrate](integrating.md) guide.

### 1. Interactive Login with SSO or Email and Password

This method is the most straightforward and allows you to log in using a single sign-on (SSO) provider or your email and password.

```sh
omctl login
```

- You will be prompted to select a login method. Choose to log in either with a single sign-on provider or by entering your email and password interactively.

### 2. Login with Email and Password (Command Line)

For a more automated approach, you can provide your email and password directly through the command line:

```sh
omctl login --email your_email@example.com --password your_password
```

- Replace `your_email@example.com` and `your_password` with your actual Omnistrate credentials.

### 3. Login Using Environment Variables

You can set your credentials as environment variables and use them to log in. This method is useful for scripting and automation:

```sh
export OMNISTRATE_USER_NAME=your_email@example.com
export OMNISTRATE_PASSWORD=your_password
./omnistrate-ctl-darwin-arm64 login --email "$OMNISTRATE_USER_NAME" --password "$OMNISTRATE_PASSWORD"
```

- This method requires you to replace the placeholders with your actual credentials and execute the appropriate binary for your system.

### 4. Login with Email and Password from stdin (Using a File)

For added security, you can store your password in a file and pipe it to the `omnistrate-ctl` command:

```sh
cat ~/omnistrate_pass.txt | omnistrate-ctl login --email your_email@example.com --password-stdin
```

- This method reads the password from the `omnistrate_pass.txt` file and logs in using the provided email. Make sure to replace the email with your actual email and ensure that your password file is securely stored.

### 5. Login with Email and Password from stdin (Using an Environment Variable)

Alternatively, you can store your password in an environment variable and use `echo` to pass it to the `omnistrate-ctl` command:

```sh
echo $OMNISTRATE_PASSWORD | omnistrate-ctl login --email your_email@example.com --password-stdin
```

- This method echoes the password stored in the `$OMNISTRATE_PASSWORD` environment variable and logs in using your email. Ensure that your environment variable is set securely.
