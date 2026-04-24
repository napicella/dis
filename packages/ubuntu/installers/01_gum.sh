### -- Manifest
### provides: common/gum
### depends_on: [common/os-libs]
### distro: [ubuntu]
### -- End
source $DIS_BINDING

if command -v gum &> /dev/null
then
    echo "gum is installed"
    return
fi

sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://repo.charm.sh/apt/gpg.key | sudo gpg --dearmor -o /etc/apt/keyrings/charm.gpg
echo "deb [signed-by=/etc/apt/keyrings/charm.gpg] https://repo.charm.sh/apt/ * *" | sudo tee /etc/apt/sources.list.d/charm.list
sudo apt update && sudo apt install -y gum
