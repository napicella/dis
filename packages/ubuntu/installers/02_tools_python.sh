### -- Manifest
### provides: common/python
### depends_on: [common/mise]
### distro: [ubuntu]
### -- End

echo "Installing Python build dependencies"
# Required by mise to compile Python from source.
# Without these, mise fails with "Python installation is missing a lib directory".
sudo apt-get install -y \
    build-essential libssl-dev zlib1g-dev libbz2-dev libreadline-dev \
    libsqlite3-dev libncursesw5-dev xz-utils tk-dev libffi-dev liblzma-dev

echo "Installing Python via mise"
mise use --global python@latest
