### -- Manifest
### provides: common/docker
### depends_on: []
### distro: [amazon_linux]
### -- End
source $DIS_BINDING

# Docker is automatically configured for Cloud Desktops. You do not need to manually install Docker.
# If you experience issues with Docker on Cloud Desktop, refer to Troubleshooting your Docker installation.

if command -v docker &> /dev/null
then
    echo "docker is installed"
    exit 0
else
    echo "Docker is automatically configured for Cloud Desktops but does not look like it is installed"
    exit 1
fi