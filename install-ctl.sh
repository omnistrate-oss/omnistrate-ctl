#!/bin/sh
set -e

RESET="\\033[0m"
RED="\\033[31;1m"
GREEN="\\033[32;1m"
YELLOW="\\033[33;1m"
BLUE="\\033[34;1m"
WHITE="\\033[37;1m"

say_green() {
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${GREEN}" "$1" "${RESET}"
    return 0
}

say_red() {
    printf "%b%s%b\\n" "${RED}" "$1" "${RESET}"
}

say_yellow() {
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${YELLOW}" "$1" "${RESET}"
    return 0
}

say_blue() {
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${BLUE}" "$1" "${RESET}"
    return 0
}

say_white() {
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${WHITE}" "$1" "${RESET}"
    return 0
}

print_unsupported_platform() {
    >&2 say_red "error: We're sorry, but it looks like Omnistrate CTL is not supported on your platform"
    >&2 say_red "       We support 64-bit versions of Linux, macOS, and Windows."
}

at_exit() {
    if [ "$?" -ne 0 ]; then
        >&2 say_red "We're sorry, but it looks like something might have gone wrong during installation."
        >&2 say_red "If you need help, please check https://omnistrate.com/support."
    fi
}

trap at_exit EXIT

INSTALL_ROOT=""
NO_EDIT_PATH=""
SILENT=""
while [ $# -gt 0 ]; do
    case "$1" in
    --silent)
        SILENT="--silent"
        ;;
    --install-root)
        INSTALL_ROOT=$2
        ;;
    --no-edit-path)
        NO_EDIT_PATH="true"
        ;;
    --version)
        VERSION=$2
        ;;
    esac
    shift
done

OS=""
case $(uname) in
"Linux"*) OS="linux" ;;
"Darwin"*) OS="darwin" ;;
"MINGW"*) OS="windows" ;;
"MSYS"*) OS="windows" ;;
*)
    print_unsupported_platform
    exit 1
    ;;
esac

ARCH=""
case $(uname -m) in
x86_64 | amd64) ARCH="amd64" ;;
arm64 | aarch64) ARCH="arm64" ;;
*)
    print_unsupported_platform
    exit 1
    ;;
esac

if [ -z "${VERSION}" ]; then
    BASE_URL="https://github.com/omnistrate-oss/omnistrate-ctl/releases/latest/download/omnistrate-ctl-${OS}-${ARCH}"
else
    BASE_URL="https://github.com/omnistrate-oss/omnistrate-ctl/releases/download/${VERSION}/omnistrate-ctl-${OS}-${ARCH}"
fi

if [ "$OS" = "windows" ]; then
    BASE_URL="${BASE_URL}.exe"
fi

OMNISTRATE_INSTALL_ROOT=${INSTALL_ROOT}
if [ "$OMNISTRATE_INSTALL_ROOT" = "" ]; then
    # Default to ~/.omnistrate
    OMNISTRATE_INSTALL_ROOT="${HOME}/.omnistrate"
fi
OMNISTRATE_CTL="${OMNISTRATE_INSTALL_ROOT}/bin/omnistrate-ctl"
OMCTL_SYMLINK="${OMNISTRATE_INSTALL_ROOT}/bin/omctl"

if [ -d "${OMNISTRATE_CTL}" ]; then
    say_red "error: ${OMNISTRATE_CTL} already exists and is a directory, refusing to proceed."
    exit 1
elif [ ! -f "${OMNISTRATE_CTL}" ]; then
    say_blue "=== Installing Omnistrate CTL ==="
else
    say_blue "=== Upgrading Omnistrate CTL ==="
fi

mkdir -p "${OMNISTRATE_INSTALL_ROOT}/bin"

say_blue "=== Downloading Omnistrate CTL for ${OS}-${ARCH} ==="

# Try compressed version first
if [ "$OS" = "windows" ]; then
    COMPRESSED_URL="${BASE_URL}.zip"
    TEMP_FILE="${OMNISTRATE_CTL}.exe"
    
    if curl -L --fail -o "${TEMP_FILE}.zip" "${COMPRESSED_URL}" 2>/dev/null; then
        say_blue "Downloaded compressed version, extracting..."
        unzip -q "${TEMP_FILE}.zip" -d "${OMNISTRATE_INSTALL_ROOT}/bin/"
        rm "${TEMP_FILE}.zip"
        mv "${OMNISTRATE_INSTALL_ROOT}/bin/omnistrate-ctl-${OS}-${ARCH}.exe" "${OMNISTRATE_INSTALL_ROOT}/bin/omnistrate-ctl.exe"
        OMNISTRATE_CTL="${OMNISTRATE_INSTALL_ROOT}/bin/omnistrate-ctl.exe"
    else
        say_yellow "Compressed version not available, downloading uncompressed version..."
        curl -L -o "${TEMP_FILE}" ${BASE_URL}
        mv "${TEMP_FILE}" "${OMNISTRATE_INSTALL_ROOT}/bin/omnistrate-ctl.exe"
        OMNISTRATE_CTL="${OMNISTRATE_INSTALL_ROOT}/bin/omnistrate-ctl.exe"
    fi
else
    COMPRESSED_URL="${BASE_URL}.tar.gz"
    
    if curl -L --fail -o "${OMNISTRATE_CTL}.tar.gz" "${COMPRESSED_URL}" 2>/dev/null; then
        say_blue "Downloaded compressed version, extracting..."
        tar -xzf "${OMNISTRATE_CTL}.tar.gz" -C "${OMNISTRATE_INSTALL_ROOT}/bin/" --strip-components=0
        rm "${OMNISTRATE_CTL}.tar.gz"
        mv "${OMNISTRATE_INSTALL_ROOT}/bin/omnistrate-ctl-${OS}-${ARCH}" "${OMNISTRATE_CTL}"
    else
        say_yellow "Compressed version not available, downloading uncompressed version..."
        curl -L -o "${OMNISTRATE_CTL}" ${BASE_URL}
    fi
fi

# If the file is not executable, we need to make it executable
if [ "$OS" != "windows" ]; then
    chmod +x "${OMNISTRATE_CTL}"
fi
say_green "Omnistrate CTL downloaded to ${OMNISTRATE_CTL}"

# Create symlink for omctl (not on Windows)
if [ "$OS" != "windows" ]; then
    if [ ! -f "${OMCTL_SYMLINK}" ]; then
        ln -s "${OMNISTRATE_CTL}" "${OMCTL_SYMLINK}"
        say_green "Symlink created: omctl -> ${OMNISTRATE_CTL}"
    else
        say_yellow "Symlink omctl already exists."
    fi
fi

# Now that we have installed Omnistrate, if it is not already on the path, let's add a line to the
# user's profile to add the folder to the PATH for future sessions.
if [ "${NO_EDIT_PATH}" != "true" ] && [ "$OS" != "windows" ]; then
    # If we can, we'll add a line to the user's .profile adding ${OMNISTRATE_INSTALL_ROOT}/bin to the PATH
    SHELL_NAME=$(basename "${SHELL}")
    PROFILE_FILE=""

    case "${SHELL_NAME}" in
    "bash")
        # Terminal.app on macOS prefers .bash_profile to .bashrc, so we prefer that
        # file when trying to put our export into a profile. On *NIX, .bashrc is
        # preferred as it is sourced for new interactive shells.
        if [ "$(uname)" != "Darwin" ]; then
            if [ -e "${HOME}/.bashrc" ]; then
                PROFILE_FILE="${HOME}/.bashrc"
            elif [ -e "${HOME}/.bash_profile" ]; then
                PROFILE_FILE="${HOME}/.bash_profile"
            fi
        else
            if [ -e "${HOME}/.bash_profile" ]; then
                PROFILE_FILE="${HOME}/.bash_profile"
            elif [ -e "${HOME}/.bashrc" ]; then
                PROFILE_FILE="${HOME}/.bashrc"
            fi
        fi
        ;;
    "zsh")
        if [ -e "${ZDOTDIR:-$HOME}/.zshrc" ]; then
            PROFILE_FILE="${ZDOTDIR:-$HOME}/.zshrc"
        fi
        ;;
    esac

    if [ -n "${PROFILE_FILE}" ]; then
        LINE_TO_ADD="export PATH=\$PATH:${OMNISTRATE_INSTALL_ROOT}/bin"
        if ! grep -q "# add Omnistrate CTL to the PATH" "${PROFILE_FILE}"; then
            say_white "+ Adding ${OMNISTRATE_INSTALL_ROOT}/bin to \$PATH in ${PROFILE_FILE}"
            printf "\\n# add Omnistrate CTL to the PATH\\n%s\\n" "${LINE_TO_ADD}" >>"${PROFILE_FILE}"
        fi

        EXTRA_INSTALL_STEP="+ Please restart your shell or add ${OMNISTRATE_INSTALL_ROOT}/bin to your \$PATH"
    else
        EXTRA_INSTALL_STEP="+ Please add ${OMNISTRATE_INSTALL_ROOT}/bin to your \$PATH"
    fi
fi
say_blue
say_blue "=== Omnistrate CTL is now installed! ==="
if [ -n "${EXTRA_INSTALL_STEP}" ]; then
    say_white "${EXTRA_INSTALL_STEP}"
fi
say_green "+ Get started with Omnistrate CTL: https://docs.omnistrate.com/getting-started/ctl-reference/#getting-started"